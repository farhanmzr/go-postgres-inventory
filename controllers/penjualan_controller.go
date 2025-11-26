// controllers/purchase_request_user.go
package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SalesRequestInput struct {
	ManualCode  *string     `json:"manual_code"` // biarkan null; admin yang isi nanti
	SalesDate   time.Time   `json:"sales_date"`  // wajib <= today
	Username    string      `json:"username"`    // auto nama user
	WarehouseID uint        `json:"warehouse_id" binding:"required"`
	CustomerID  uint        `json:"customer_id" binding:"required"`
	Payment     string      `json:"payment" binding:"required"` // "CASH" | "CREDIT"
	Items       []SalesItem `json:"items" binding:"required,min=1"`
}

type SalesItem struct {
	BarangID  uint  `json:"barang_id" binding:"required"`
	Qty       int64 `json:"qty" binding:"required,gt=0"`
	SellPrice int64 `json:"sell_price" binding:"required,gt=0"`
}

func CreatePenjualan(c *gin.Context) {
	var in SalesRequestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Payload tidak valid", "error": err.Error()})
		return
	}

	// validasi tanggal tidak ke depan (gunakan UTC agar konsisten)
	loc, _ := time.LoadLocation("Asia/Jakarta")
	// hari ini (tanpa jam)
	today := time.Now().In(loc).Truncate(24 * time.Hour)
	// tanggal request (tanpa jam)
	reqDate := in.SalesDate.In(loc).Truncate(24 * time.Hour)
	// kalau tanggal request > hari ini -> ke depan
	if reqDate.After(today) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Tanggal pembelian tidak boleh ke depan"})
		return
	}

	// validasi payment
	if in.Payment != "CASH" && in.Payment != "CREDIT" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Metode pembayaran tidak valid"})
		return
	}

	// --- normalize user_id ---
	rawID, _ := c.Get("user_id")
	var userID uint
	switch v := rawID.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case int64:
		userID = uint(v)
	case float64:
		userID = uint(v)
	case string:
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			userID = uint(n)
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid"})
		return
	}

	// --- cek FK gudang & customer ---
	var cnt int64
	if err := config.DB.Model(&models.Gudang{}).Where("id = ?", in.WarehouseID).Count(&cnt).Error; err != nil || cnt == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Gudang tidak ditemukan"})
		return
	}
	if err := config.DB.Model(&models.Customer{}).Where("id = ?", in.CustomerID).Count(&cnt).Error; err != nil || cnt == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Customer tidak ditemukan"})
		return
	}

	// --- opsional: pastikan semua barang_id ada & memang milik gudang tsb ---
	for _, it := range in.Items {
		var exist int64
		if err := config.DB.Model(&models.GudangBarang{}).
			Where("barang_id = ? AND gudang_id = ?", it.BarangID, in.WarehouseID).
			Count(&exist).Error; err != nil || exist == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": fmt.Sprintf("Barang %d tidak ditemukan di gudang %d", it.BarangID, in.WarehouseID),
			})
			return
		}
	}

	// ===== transaksi + retry untuk antisipasi race =====
	const maxRetries = 3
	var lastErr error

	for range maxRetries {
		lastErr = config.DB.Transaction(func(tx *gorm.DB) error {
			// a) Lock row terakhir user ini (bukan agregat)
			var last models.SalesRequest
			if err := tx.
				Where("created_by_id = ?", userID).
				Order("trans_seq DESC").
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Limit(1).
				Find(&last).Error; err != nil {
				return err
			}

			nextSeq := uint(1)
			if last.ID != 0 {
				nextSeq = last.TransSeq + 1
			}
			transCode := fmt.Sprintf("SL-%d-%d", userID, nextSeq)

			// b) siapkan items
			items := make([]models.SalesReqItem, 0, len(in.Items))
			for _, it := range in.Items {
				items = append(items, models.SalesReqItem{
					BarangID:  it.BarangID,
					Qty:       it.Qty,
					SellPrice: it.SellPrice,
					LineTotal: it.Qty * it.SellPrice,
				})
			}

			// c) insert header
			data := models.SalesRequest{
				TransCode:   transCode,
				TransSeq:    nextSeq,
				ManualCode:  in.ManualCode,
				Username:    in.Username,
				SalesDate:   in.SalesDate,
				WarehouseID: in.WarehouseID,
				CustomerID:  in.CustomerID,
				Payment:     models.PaymentMethod(in.Payment),
				Status:      models.StatusPending,
				Items:       items,
				CreatedByID: userID,
			}

			if err := tx.Create(&data).Error; err != nil {
				// jika bentrok unik, bubble up dengan kode supaya kita retry
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					return fmt.Errorf("unique_violation: %w", err)
				}
				return err
			}
			return nil
		})

		if lastErr == nil {
			// sukses
			c.JSON(http.StatusCreated, gin.H{"message": "Berhasil membuat Penjualan (PENDING)"})
			return
		}

		// kalau bentrok unik, retry
		if strings.Contains(lastErr.Error(), "unique_violation") {
			continue
		}
		break
	}

	// jika masih gagal
	c.JSON(http.StatusInternalServerError, gin.H{
		"message": "Gagal membuat permintaan penjualan",
		"error":   lastErr.Error(),
	})
}

func SalesReqUserList(c *gin.Context) {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	switch status {
	case string(models.StatusPending), string(models.StatusApproved), string(models.StatusRejected):
	default:
		status = string(models.StatusPending)
	}

	// ambil user_id dari context
	rawID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak ditemukan"})
		return
	}
	var userID uint
	switch v := rawID.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case int64:
		userID = uint(v)
	case float64:
		userID = uint(v)
	case string:
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			userID = uint(n)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid"})
		return
	}

	var rows []models.SalesRequest
	if err := config.DB.Preload("Customer").
		Preload("Warehouse").
		Preload("Items.Barang").
		Where("status = ? AND created_by_id = ?", status, userID).
		Order("id DESC").
		Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"message": "Gagal mengambil data", "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Berhasil mengambil data Penjualan", "data": rows})
}

func SalesInvoiceDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id tidak valid"})
		return
	}
	var inv models.SalesInvoice
	if err := config.DB.
		Preload("Items.Barang").
		First(&inv, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "invoice tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "gagal mengambil data", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil data Invoice", "data": inv})
}
