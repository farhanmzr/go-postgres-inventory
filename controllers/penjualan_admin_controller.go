// controllers/purchase_request_admin.go
package controllers

import (
	"errors"
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RejectBody struct {
	Reason string `json:"reason" binding:"required"`
}

// controllers/purchase_request_admin.go

func SalesReqAdminList(c *gin.Context) {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	// hanya izinkan 3 status
	switch status {
	case string(models.StatusPending), string(models.StatusApproved), string(models.StatusRejected):
		// ok
	default:
		// default: PENDING (atau 400 kalau mau strict)
		status = string(models.StatusPending)
	}

	var rows []models.SalesRequest
	if err := config.DB.Preload("Customer").
		Preload("Warehouse").
		Preload("Items.Barang").
		Where("status = ?", status).
		Order("id DESC").
		Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"message": "Gagal mengambil data", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Berhasil mengambil semua data Penjualan", "data": rows})
}

var (
	errNotFound             = errors.New("NOT_FOUND")
	errBadStatus            = errors.New("BAD_STATUS")
	errAlreadyProcessed     = errors.New("REQUEST_ALREADY_PROCESSED")
	errBarangNotInWarehouse = errors.New("BARANG_NOT_IN_WAREHOUSE")
)

func SalesReqApprove(c *gin.Context) {
	id := c.Param("id")

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Lock PR agar tidak diproses bersamaan
		var pr models.SalesRequest
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Items").
			Preload("Customer").
			First(&pr, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errNotFound
			}
			return err
		}

		if pr.Status != models.StatusPending {
			return errBadStatus
		}

		// 2) Idempotent: set APPROVED hanya jika masih PENDING
		res := tx.Model(&models.SalesRequest{}).
			Where("id = ? AND status = ?", pr.ID, models.StatusPending).
			Updates(map[string]any{
				"status":        models.StatusApproved,
				"reject_reason": gorm.Expr("NULL"),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errAlreadyProcessed
		}

		// 3) Kurangi stok per item (atomic) + guard gudang
		for _, it := range pr.Items {
			dec := tx.Model(&models.GudangBarang{}).
				Where("barang_id = ? AND gudang_id = ?", it.BarangID, pr.WarehouseID).
				UpdateColumn("stok", gorm.Expr("stok - ?", it.Qty))
			if dec.Error != nil {
				return dec.Error
			}
			if dec.RowsAffected == 0 {
				return errBarangNotInWarehouse
			}
		}

		// 4) Buat invoice penjualan otomatis
		var subtotal int64 = 0
		invItems := make([]models.SalesInvoiceItem, 0, len(pr.Items))
		for _, it := range pr.Items {
			// Ambil COST dari GudangBarang (harga beli terakhir per gudang)
			var gb models.GudangBarang
			if err := tx.
				Where("barang_id = ? AND gudang_id = ?", it.BarangID, pr.WarehouseID).
				First(&gb).Error; err != nil {
				return err
			}

			cost := int64(math.Round(gb.HargaBeli)) // asumsi float64 → int64

			netPrice := it.SellPrice
			netLine := netPrice * it.Qty
			profitPer := netPrice - cost
			profitTot := profitPer * it.Qty

			invItems = append(invItems, models.SalesInvoiceItem{
				BarangID:      it.BarangID,
				Qty:           it.Qty,
				Price:         netPrice,
				CostPrice:     cost,
				ProfitPerUnit: profitPer,
				ProfitTotal:   profitTot,
				LineTotal:     netLine,
			})
			subtotal += netLine
		}

		discount := int64(0)
		tax := int64(0)
		grand := subtotal - discount + tax

		inv := models.SalesInvoice{
			SalesRequestID: pr.ID,
			InvoiceNo:      pr.TransCode,
			Username:       pr.Username, // pastikan relasi Customer ter-preload
			Payment:        pr.Payment,
			InvoiceDate:    time.Now().UTC(),
			Subtotal:       subtotal,
			Discount:       discount,
			Tax:            tax,
			GrandTotal:     grand,
			Items:          invItems,
		}
		if err := tx.Create(&inv).Error; err != nil {
			return err
		}

		// 5) Jika CASH/BANK -> uang masuk ke wallet saat approve
		if pr.Payment == models.PaymentCash || pr.Payment == models.PaymentBank {
			if pr.WalletID == nil || *pr.WalletID == 0 {
				return errors.New("wallet_id wajib untuk CASH/BANK")
			}

			// (optional) validasi wallet gudang + type cocok
			var w models.WarehouseWallet
			if err := tx.First(&w, *pr.WalletID).Error; err != nil {
				return err
			}
			if w.GudangID != pr.WarehouseID {
				return errors.New("wallet bukan milik gudang ini")
			}
			if !w.IsActive {
				return errors.New("wallet tidak aktif")
			}
			if pr.Payment == models.PaymentCash && w.Type != models.WalletCash {
				return errors.New("payment CASH harus pilih wallet tipe CASH")
			}
			if pr.Payment == models.PaymentBank && w.Type != models.WalletBank {
				return errors.New("payment BANK harus pilih wallet tipe BANK")
			}

			// saldo IN
			if err := applyWalletDelta(
				tx,
				*pr.WalletID,
				pr.WarehouseID,
				+inv.GrandTotal,
				models.WalletTxSalesPaid, // boleh rename jadi SALES_PAID kalau mau
				"sales_request",
				pr.ID,
				pr.CreatedByID,
				"Penjualan "+string(pr.Payment),
				inv.InvoiceDate,
			); err != nil {
				return err
			}
		}

		// 6) Jika payment CREDIT -> buat Piutang (model baru)
		if pr.Payment == models.PaymentCredit {
			due := inv.InvoiceDate.AddDate(0, 0, 7)

			piuItems := make([]models.PiutangItem, 0, len(invItems))
			for _, iv := range invItems {
				var b models.Barang
				if err := tx.Select("id, nama, kode").First(&b, iv.BarangID).Error; err != nil {
					return err
				}
				piuItems = append(piuItems, models.PiutangItem{
					BarangID:  iv.BarangID,
					Nama:      b.Nama,
					Kode:      b.Kode,
					Qty:       iv.Qty,
					Price:     iv.Price,
					LineTotal: iv.LineTotal,
				})
			}

			piu := models.Piutang{
				UserID:         pr.CreatedByID,
				UserName:       pr.Username,
				SalesRequestID: pr.ID,
				InvoiceNo:      inv.InvoiceNo,
				InvoiceDate:    inv.InvoiceDate,
				DueDate:        due,
				Total:          inv.GrandTotal,
				TotalPaid:      0,
				IsPaid:         false,
				Items:          piuItems,
			}
			if err := tx.Create(&piu).Error; err != nil {
				return err
			}
		}

		return nil
	})

	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"message": "Approved & invoice dibuat"})
	case errors.Is(err, errNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Data tidak ditemukan"})
	case errors.Is(err, errBadStatus):
		c.JSON(http.StatusBadRequest, gin.H{"message": "Hanya PENDING yang bisa di-approve"})
	case errors.Is(err, errAlreadyProcessed):
		c.JSON(http.StatusConflict, gin.H{"message": "Request sudah diproses"})
	case errors.Is(err, errBarangNotInWarehouse):
		c.JSON(http.StatusBadRequest, gin.H{"message": "Barang tidak sesuai gudang PR"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal approve", "error": err.Error()})
	}
}

func SalesReqReject(c *gin.Context) {
	id := c.Param("id")

	type RejectBody struct {
		Reason string `json:"reason" binding:"required"`
	}
	var body RejectBody
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Reason) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Alasan wajib diisi"})
		return
	}
	reason := strings.TrimSpace(body.Reason)

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// Lock PR agar tidak diproses bersamaan
		var pr models.SalesRequest
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&pr, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errNotFound
			}
			return err
		}

		if pr.Status != models.StatusPending {
			return errBadStatus
		}

		// Idempotent: update hanya jika masih PENDING
		res := tx.Model(&models.SalesRequest{}).
			Where("id = ? AND status = ?", pr.ID, models.StatusPending).
			Updates(map[string]any{
				"status":        models.StatusRejected,
				"reject_reason": reason,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errAlreadyProcessed
		}

		return nil
	})

	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"message": "Rejected"})
	case errors.Is(err, errNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "Data tidak ditemukan"})
	case errors.Is(err, errBadStatus):
		c.JSON(http.StatusBadRequest, gin.H{"message": "Hanya PENDING yang bisa di-reject"})
	case errors.Is(err, errAlreadyProcessed):
		c.JSON(http.StatusConflict, gin.H{"message": "Request sudah diproses"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal reject", "error": err.Error()})
	}
}

func DeletePenjualanAdmin(c *gin.Context) {
	if _, err := currentAdminID(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized", "error": err.Error()})
		return
	}

	id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id tidak valid"})
		return
	}
	id := uint(id64)

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var sr models.SalesRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&sr, id).Error; err != nil {
			return err
		}

		if sr.Status != models.StatusPending && sr.Status != models.StatusRejected {
			return errors.New("tidak bisa delete: hanya PENDING/REJECTED")
		}

		var invCnt int64
		if err := tx.Model(&models.SalesInvoice{}).
			Where("sales_request_id = ?", sr.ID).
			Count(&invCnt).Error; err != nil {
			return err
		}
		if invCnt > 0 {
			return errors.New("tidak bisa delete: invoice sudah ada")
		}

		var piuCnt int64
		if err := tx.Model(&models.Piutang{}).
			Where("sales_request_id = ?", sr.ID).
			Count(&piuCnt).Error; err != nil {
			return err
		}
		if piuCnt > 0 {
			return errors.New("tidak bisa delete: piutang sudah ada")
		}

		if err := tx.Where("sales_request_id = ?", sr.ID).
			Delete(&models.SalesReqItem{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&sr).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			code = http.StatusNotFound
		}
		c.JSON(code, gin.H{"message": "Gagal menghapus Penjualan", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil menghapus Penjualan"})
}
