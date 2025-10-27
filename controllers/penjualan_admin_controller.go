// controllers/purchase_request_admin.go
package controllers

import (
	"errors"
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
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

func SalesReqPendingList(c *gin.Context) {
	var rows []models.SalesRequest
	if err := config.DB.Preload("Supplier").Preload("Warehouse").Preload("Items.Barang").
		Where("status = ?", models.StatusPending).Order("id DESC").
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
			line := it.Qty * it.SellPrice // asumsikan ada field SellPrice di SalesItem
			subtotal += line
			invItems = append(invItems, models.SalesInvoiceItem{
				BarangID:  it.BarangID,
				Qty:       it.Qty,
				Price:     it.SellPrice,
				LineTotal: line,
			})
		}
		discount := int64(0)
		tax := int64(0)
		grand := subtotal - discount + tax

		inv := models.SalesInvoice{
			InvoiceNo:      pr.TransCode,
			SalesRequestID: pr.ID,
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
