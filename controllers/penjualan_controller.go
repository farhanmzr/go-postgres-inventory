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
	ManualCode  *string     `json:"manual_code"`
	SalesDate   time.Time   `json:"sales_date"`
	Username    string      `json:"username"`
	WarehouseID uint        `json:"warehouse_id" binding:"required"`
	CustomerID  uint        `json:"customer_id" binding:"required"`
	Payment     string      `json:"payment" binding:"required"` // CASH | BANK | CREDIT
	WalletID    *uint       `json:"wallet_id"`                  // wajib untuk CASH/BANK
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

	// validasi payment
	if in.Payment != "CASH" && in.Payment != "BANK" && in.Payment != "CREDIT" {
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

			for _, it := range in.Items {
				// lock row stok supaya aman dari race
				var gb models.GudangBarang
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("barang_id = ? AND gudang_id = ?", it.BarangID, in.WarehouseID).
					First(&gb).Error; err != nil {
					return err
				}
				if int64(gb.Stok) < it.Qty {
					return fmt.Errorf("stok tidak cukup untuk barang_id=%d (stok=%d, minta=%d)", it.BarangID, gb.Stok, it.Qty)
				}
			}

			pm := models.PaymentMethod(in.Payment)
			if pm == models.PaymentCash || pm == models.PaymentBank {
				if in.WalletID == nil || *in.WalletID == 0 {
					return fmt.Errorf("wallet_id wajib untuk payment %s", in.Payment)
				}

				// cek wallet milik gudang dan tipe cocok
				var w models.WarehouseWallet
				if err := tx.First(&w, *in.WalletID).Error; err != nil {
					return err
				}
				if w.GudangID != in.WarehouseID {
					return fmt.Errorf("wallet bukan milik gudang ini")
				}
				if !w.IsActive {
					return fmt.Errorf("wallet tidak aktif")
				}
				if pm == models.PaymentCash && w.Type != models.WalletCash {
					return fmt.Errorf("payment CASH harus pilih wallet tipe CASH (laci)")
				}
				if pm == models.PaymentBank && w.Type != models.WalletBank {
					return fmt.Errorf("payment BANK harus pilih wallet tipe BANK")
				}
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
				Payment:     pm,
				WalletID:    in.WalletID,
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

func deletePenjualanCore(tx *gorm.DB, srID uint, actorID uint, checkOwner bool) error {
    // lock SR + preload items
    var sr models.SalesRequest
    if err := tx.Clauses(clauseUpdateLock()).
        Preload("Items").
        First(&sr, srID).Error; err != nil {
        return err
    }

    if checkOwner && sr.CreatedByID != actorID {
        return errors.New("forbidden")
    }

    // CASE 1: PENDING/REJECTED → belum ada efek stok & uang
    if sr.Status == models.StatusPending || sr.Status == models.StatusRejected {
        if err := tx.Where("sales_request_id = ?", sr.ID).
            Delete(&models.SalesReqItem{}).Error; err != nil {
            return err
        }
        return tx.Delete(&sr).Error
    }

    // CASE 2: APPROVED → reversal semua
    if sr.Status != models.StatusApproved {
        return fmt.Errorf("status tidak valid: %s", sr.Status)
    }

    // ambil invoice
    var inv models.SalesInvoice
    if err := tx.Where("sales_request_id = ?", sr.ID).First(&inv).Error; err != nil {
        return err
    }

    // 1) balikin stok (stok + qty)
    for _, it := range sr.Items {
        if err := tx.Model(&models.GudangBarang{}).
            Where("barang_id = ? AND gudang_id = ?", it.BarangID, sr.WarehouseID).
            UpdateColumn("stok", gorm.Expr("stok + ?", it.Qty)).Error; err != nil {
            return err
        }
    }

    // 2) reversal uang/piutang
    switch sr.Payment {
    case models.PaymentCash, models.PaymentBank:
        if sr.WalletID == nil || *sr.WalletID == 0 {
            return errors.New("wallet_id kosong pada penjualan CASH/BANK")
        }
        // uang yang dulu masuk harus keluar sekarang
        if err := applyWalletDelta(
            tx,
            *sr.WalletID,
            sr.WarehouseID,
            -inv.GrandTotal,
            models.WalletTxSalesRefund,
            "sales_refund",
            sr.ID,
            actorID,
            "Reverse penjualan (hapus)",
            time.Now().UTC(),
        ); err != nil {
            return err
        }

    case models.PaymentCredit:
        // cari piutang berdasarkan sales_request_id
        var p models.Piutang
        err := tx.Where("sales_request_id = ?", sr.ID).First(&p).Error
        if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
            return err
        }
        if err == nil {
            // reverse semua receipt dulu kalau ada
            if p.TotalPaid > 0 {
                if err := refundAllPiutangReceipts(tx, p.ID, sr.WarehouseID, actorID); err != nil {
                    return err
                }
            }
            // hapus piutang (items cascade)
            if err := tx.Delete(&p).Error; err != nil {
                return err
            }
        }

    default:
        return fmt.Errorf("payment tidak dikenal: %s", sr.Payment)
    }

    // 3) hapus invoice
    if err := tx.Where("sales_request_id = ?", sr.ID).
        Delete(&models.SalesInvoice{}).Error; err != nil {
        return err
    }

    // 4) hapus items SR
    if err := tx.Where("sales_request_id = ?", sr.ID).
        Delete(&models.SalesReqItem{}).Error; err != nil {
        return err
    }

    // 5) hapus header SR
    return tx.Delete(&sr).Error
}

func DeletePenjualanUser(c *gin.Context) {
    uid, err := currentUserID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message":"Unauthorized", "error": err.Error()})
        return
    }

    id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"message":"id tidak valid"})
        return
    }

    err = config.DB.Transaction(func(tx *gorm.DB) error {
        return deletePenjualanCore(tx, uint(id64), uid, true)
    })

    if err != nil {
        code := http.StatusBadRequest
        if errors.Is(err, gorm.ErrRecordNotFound) { code = http.StatusNotFound }
        c.JSON(code, gin.H{"message":"Gagal hapus penjualan", "error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message":"Penjualan berhasil dihapus (reversal stok & uang/piutang)"})
}


