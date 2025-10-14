package controllers

import (
	"net/http"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
)

func CreateBarang(c *gin.Context) {
	var barang models.Barang
	if err := c.ShouldBindJSON(&barang); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	config.DB.Create(&barang)
	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil ditambahkan", "data": barang})
}

func GetAllBarang(c *gin.Context) {
	var barang []models.Barang
	config.DB.Find(&barang)
	c.JSON(http.StatusOK, gin.H{"data": barang})
}

func GetBarangByID(c *gin.Context) {
	id := c.Param("id")
	var barang models.Barang

	if err := config.DB.First(&barang, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Barang tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": barang})
}

func UpdateBarang(c *gin.Context) {
	id := c.Param("id")
	var barang models.Barang

	if err := config.DB.First(&barang, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Barang tidak ditemukan"})
		return
	}

	var input models.Barang
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	config.DB.Model(&barang).Updates(input)
	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil diupdate", "data": barang})
}

func DeleteBarang(c *gin.Context) {
	id := c.Param("id")
	var barang models.Barang

	if err := config.DB.First(&barang, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Barang tidak ditemukan"})
		return
	}

	config.DB.Delete(&barang)
	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil dihapus"})
}
