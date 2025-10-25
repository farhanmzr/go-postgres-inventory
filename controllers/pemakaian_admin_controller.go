package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ItemDecisionInput struct {
	ItemID uint    `json:"item_id" binding:"required"`
	Action string  `json:"action"  binding:"required"` // "APPROVE" | "REJECT"
	Note   *string `json:"note"`
}

func UsageItemDecide(c *gin.Context) {
	var in ItemDecisionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Payload tidak valid", "error": err.Error()})
		return
	}

	var item models.UsageItem
	if err := config.DB.First(&item, in.ItemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Item tidak ditemukan"})
		return
	}

	var target models.UsageItemStatus
	switch in.Action {
	case "APPROVE":
		target = models.ItemApproved
	case "REJECT":
		target = models.ItemRejected
	default:
		c.JSON(http.StatusBadRequest, gin.H{"message": "Action tidak dikenal"})
		return
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// lock item agar konsisten
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, in.ItemID).Error; err != nil {
			return err
		}

		// APPROVE → kurangi stok sekali saja
		if target == models.ItemApproved && !item.StockApplied {
			// lock row barang
			var stok int
			if err := tx.
				Table("barangs").
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Select("stok").
				Where("id = ?", item.BarangID).
				Scan(&stok).Error; err != nil {
				return err
			}

			if stok < int(item.Qty) {
				return errors.New("stok tidak mencukupi")
			}

			// update stok
			if err := tx.Exec("UPDATE barangs SET stok = stok - ? WHERE id = ?", item.Qty, item.BarangID).Error; err != nil {
				return err
			}
			// tandai status & stock_applied=true
			if err := tx.Model(&models.UsageItem{}).
				Where("id = ?", item.ID).
				Updates(map[string]any{
					"item_status":   target,
					"note":          in.Note,
					"stock_applied": true,
				}).Error; err != nil {
				return err
			}
		} else {
			// REJECT atau APPROVE ulang (idempotent)
			if err := tx.Model(&models.UsageItem{}).
				Where("id = ?", item.ID).
				Updates(map[string]any{
					"item_status": target,
					"note":        in.Note,
				}).Error; err != nil {
				return err
			}
		}

		// header → SUDAH_DIPROSES kalau tidak ada PENDING
		var pending int64
		if err := tx.Model(&models.UsageItem{}).
			Where("usage_request_id = ? AND item_status = 'PENDING'", item.UsageRequestID).
			Count(&pending).Error; err != nil {
			return err
		}
		hdr := models.UsageBelumDiproses
		if pending == 0 {
			hdr = models.UsageSudahDiproses
		}
		return tx.Model(&models.UsageRequest{}).
			Where("id = ?", item.UsageRequestID).
			Update("status", hdr).Error
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Gagal memproses item", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Item diproses"})
}

// GET /admin/pemakaian/:id
func UsageDetail(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id tidak boleh kosong"})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id tidak valid"})
		return
	}

	var row models.UsageRequest
	if err := config.DB.
		Preload("Items.Barang").
		Preload("Items.Customer").
		First(&row, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "pemakaian tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "gagal mengambil data", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": row})
}