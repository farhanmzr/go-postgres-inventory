// controllers/purchase_request_user.go
package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SalesRequestInput struct {
	TransCode    string      `json:"trans_code"`    // dari UI boleh isi nomor transaksi (opsional), kalau kosong server generate
	ManualCode   *string     `json:"manual_code"`   // biarkan null; admin yang isi nanti
	PurchaseDate time.Time   `json:"purchase_date"` // wajib <= today
	Username     string      `json:"username"`      // auto nama user
	WarehouseID  uint        `json:"warehouse_id" binding:"required"`
	CustomerID   uint        `json:"customer_id" binding:"required"`
	Payment      string      `json:"payment" binding:"required"` // "CASH" | "CREDIT"
	Items        []SalesItem `json:"items" binding:"required,min=1"`
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
	today := time.Now().In(loc).Truncate(24 * time.Hour)
	if in.PurchaseDate.In(loc).After(today) {
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
		if err := config.DB.Model(&models.Barang{}).
			Where("id = ? AND gudang_id = ?", it.BarangID, in.WarehouseID).
			Count(&exist).Error; err != nil || exist == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("Barang %d tidak ditemukan di gudang %d", it.BarangID, in.WarehouseID)})
			return
		}
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// siapkan items
		items := make([]models.SalesReqItem, 0, len(in.Items))
		for _, it := range in.Items {
			items = append(items, models.SalesReqItem{
				BarangID:  it.BarangID,
				Qty:       it.Qty,
				SellPrice: it.SellPrice,
				LineTotal: it.Qty * it.SellPrice,
			})
		}

		penjualanData := models.SalesRequest{
			TransCode:    in.TransCode,
			ManualCode:   in.ManualCode,
			Username:     in.Username,
			PurchaseDate: in.PurchaseDate,
			WarehouseID:  in.WarehouseID,
			CustomerID:   in.CustomerID,
			Payment:      models.PaymentMethod(in.Payment),
			Status:       models.StatusPending,
			Items:        items,
			CreatedByID:  userID,
		}
		return tx.Create(&penjualanData).Error
	})

	if err != nil {
		// log error ke stdout juga biar ketahuan
		fmt.Printf("PurchaseReqCreate error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat permintaan pembelian", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Berhasil melakukan Penjualan"})
}

func PenjualanMyList(c *gin.Context) {
	// --- normalize user_id from context (hindari panic) ---
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
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid (tipe tidak dikenal)"})
		return
	}

	var rows []models.PurchaseRequest
	if err := config.DB.
		Where("created_by_id = ?", userID).
		Preload("Supplier").
		Preload("Warehouse").
		Preload("Items.Barang").
		Order("id DESC").
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal mengambil data",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil semua data Penjualan", "data": rows})
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
