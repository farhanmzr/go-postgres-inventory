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
    adminID, err := currentAdminID(c)
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
        return deletePembelianCore(tx, uint(id64), adminID, false) // ✅ admin tidak cek owner
    })

    if err != nil {
        code := http.StatusBadRequest
        if errors.Is(err, gorm.ErrRecordNotFound) { code = http.StatusNotFound }
        c.JSON(code, gin.H{"message":"Gagal hapus pembelian (admin)", "error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message":"Pembelian berhasil dihapus (admin) (reversal stok & uang)"})
}



