// controllers/purchase_request_admin.go
package controllers

import (
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
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