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
		// lock item
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&item, in.ItemID).Error; err != nil {
			return err
		}

		// ambil header untuk dapat WarehouseID
		var header models.UsageRequest
		if err := tx.First(&header, item.UsageRequestID).Error; err != nil {
			return err
		}

		if target == models.ItemApproved && !item.StockApplied {
			// lock row stok di gudang_barangs
			var gb models.GudangBarang
			if err := tx.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("gudang_id = ? AND barang_id = ?", header.WarehouseID, item.BarangID).
				First(&gb).Error; err != nil {
				return err
			}

			if gb.Stok < int(item.Qty) {
				return errors.New("stok tidak mencukupi")
			}

			// update stok di GudangBarang
			if err := tx.Model(&models.GudangBarang{}).
				Where("id = ?", gb.ID).
				UpdateColumn("stok", gorm.Expr("stok - ?", item.Qty)).Error; err != nil {
				return err
			}

			// update status item
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
			// REJECT / re-approve
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

func AdminGetAllPemakaian(c *gin.Context) {

	var grups []models.UsageRequest
	if err := config.DB.
		Preload("Warehouse").
		Preload("Customer").
		Preload("Items").          // ⬅️ tambahkan ini
		Preload("Items.Barang").   // ⬅️ ini baru jalan karena field-nya ada
		Preload("Items.Customer"). // ⬅️ opsional, kalau mau ikut ditampilkan
		Order("id DESC").
		Find(&grups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil data", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": grups})
}

func DeleteUsageAdmin(c *gin.Context) {
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
		var u models.UsageRequest

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Items").
			First(&u, id).Error; err != nil {
			return err
		}

		if u.Status != models.UsageBelumDiproses {
			return errors.New("tidak bisa delete: status sudah diproses")
		}

		for _, it := range u.Items {
			if it.ItemStatus != models.ItemPending || it.StockApplied {
				return errors.New("tidak bisa delete: ada item sudah diproses")
			}
		}

		return tx.Delete(&u).Error
	})

	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			code = http.StatusNotFound
		}
		c.JSON(code, gin.H{"message": "Gagal menghapus pemakaian", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil menghapus pemakaian"})
}
