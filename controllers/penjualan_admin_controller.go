// controllers/purchase_request_admin.go
package controllers

import (
	"errors"
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"math"
	"net/http"
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
			dec := tx.Model(&models.Barang{}).
				Where("id = ? AND gudang_id = ?", it.BarangID, pr.WarehouseID).
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
			// ambil COST dari barang di gudang tsb (last buy price / WA / FIFO — di sini contoh last buy price)
			var barang models.Barang
			if err := tx.Where("id = ? AND gudang_id = ?", it.BarangID, pr.WarehouseID).
				First(&barang).Error; err != nil {
				return err
			}

			// asumsi barang.HargaBeli disimpan float64 → konversi ke int64 sesuai kebijakan (pembulatan ke terdekat)
			cost := int64(math.Round(barang.HargaBeli))

			// guard untuk qty > 0
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
