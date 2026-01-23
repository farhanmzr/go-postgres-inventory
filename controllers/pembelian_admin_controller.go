// controllers/purchase_request_admin.go
package controllers

import (
	"errors"
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func PurchaseReqList(c *gin.Context) {
	var rows []models.PurchaseRequest
	if err := config.DB.Preload("Supplier").Preload("Warehouse").Preload("Items.Barang").Order("id DESC").
		Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"message": "Gagal mengambil data", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Berhasil mengambil semua data Pembelian", "data": rows})
}

func DeletePembelianAdmin(c *gin.Context) {
    // admin-only
    if _, err := currentAdminID(c); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized", "error": err.Error()})
        return
    }

    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"message": "id tidak valid"})
        return
    }

    var pr models.PurchaseRequest
    if err := config.DB.Preload("Items").First(&pr, uint(id)).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"message": "Pembelian tidak ditemukan"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil data pembelian", "error": err.Error()})
        return
    }

    err = config.DB.Transaction(func(tx *gorm.DB) error {
        return deletePembelianCore(tx, &pr)
    })
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Gagal menghapus Pembelian", "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Berhasil menghapus Pembelian (admin)"})
}


