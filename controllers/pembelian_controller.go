package controllers

import (
	"net/http"
	"time"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
)

// 🟢 USER: Membuat pembelian baru
func CreatePembelian(c *gin.Context) {
	var input struct {
		BarangID uint `json:"barang_id"`
		Jumlah   int  `json:"jumlah"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	userID, _ := c.Get("user_id") // dari token JWT middleware nanti

	pembelian := models.Pembelian{
		UserID:   userID.(uint),
		BarangID: input.BarangID,
		Jumlah:   input.Jumlah,
		Status:   "pending",
		Tanggal:  time.Now(),
	}

	config.DB.Create(&pembelian)
	c.JSON(http.StatusOK, gin.H{"message": "Pembelian dikirim untuk verifikasi", "data": pembelian})
}

// 🔵 ADMIN: Melihat semua pembelian (pending, approved, rejected)
func GetAllPembelian(c *gin.Context) {
	var pembelian []models.Pembelian
	config.DB.Preload("User").Preload("Barang").Find(&pembelian)
	c.JSON(http.StatusOK, gin.H{"data": pembelian})
}

// 🟡 ADMIN: Verifikasi (approve/reject)
func UpdatePembelianStatus(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Status string `json:"status"` // "approved" atau "rejected"
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	var pembelian models.Pembelian
	if err := config.DB.Preload("Barang").First(&pembelian, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pembelian tidak ditemukan"})
		return
	}

	if input.Status == "approved" {
		// Kurangi stok barang
		if pembelian.Barang.Stok < pembelian.Jumlah {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok tidak cukup"})
			return
		}
		pembelian.Barang.Stok -= pembelian.Jumlah
		config.DB.Save(&pembelian.Barang)
	}

	pembelian.Status = input.Status
	config.DB.Save(&pembelian)

	c.JSON(http.StatusOK, gin.H{"message": "Status diperbarui", "data": pembelian})
}
