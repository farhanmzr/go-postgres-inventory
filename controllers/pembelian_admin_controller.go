// controllers/purchase_request_admin.go
package controllers

import (

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
)

type RejectBody struct {
	Reason string `json:"reason" binding:"required"`
}

func PurchaseReqPendingList(c *gin.Context) {
	var rows []models.PurchaseRequest
	if err := config.DB.Preload("Supplier").Preload("Warehouse").Preload("Items.Barang").
		Where("status = ?", models.StatusPending).Order("id DESC").
		Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"message": "Gagal mengambil data", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": rows})
}

func PurchaseReqApprove(c *gin.Context) {
	var pr models.PurchaseRequest
	if err := config.DB.Preload("Items").First(&pr, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"message": "Data tidak ditemukan"})
		return
	}
	if pr.Status != models.StatusPending {
		c.JSON(400, gin.H{"message": "Hanya PENDING yang bisa di-approve"})
		return
	}
	if err := config.DB.Model(&pr).Updates(map[string]any{
		"status":        models.StatusApproved,
		"reject_reason": nil,
	}).Error; err != nil {
		c.JSON(500, gin.H{"message": "Gagal approve"})
		return
	}
	c.JSON(200, gin.H{"message": "Approved"})
}

func PurchaseReqReject(c *gin.Context) {
	var pr models.PurchaseRequest
	if err := config.DB.First(&pr, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"message": "Data tidak ditemukan"})
		return
	}
	if pr.Status != models.StatusPending {
		c.JSON(400, gin.H{"message": "Hanya PENDING yang bisa di-reject"})
		return
	}
	var body RejectBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"message": "Alasan wajib diisi"})
		return
	}
	if err := config.DB.Model(&pr).Updates(map[string]any{
		"status":        models.StatusRejected,
		"reject_reason": body.Reason,
	}).Error; err != nil {
		c.JSON(500, gin.H{"message": "Gagal reject"})
		return
	}
	c.JSON(200, gin.H{"message": "Rejected"})
}
