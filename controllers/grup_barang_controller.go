package controllers

import (
	"net/http"
	"strconv"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
)

func CreateGrupBarang(c *gin.Context) {
	var input struct {
		Nama string `json:"nama"`
		Kode string `json:"kode"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Cek apakah kode grup barang sudah ada
	var exist models.GrupBarang
	if err := config.DB.Where("kode = ?", input.Kode).First(&exist).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode grup barang sudah digunakan"})
		return
	}

	grupBarang := models.GrupBarang{
		Nama: input.Nama,
		Kode: input.Kode,
	}

	if err := config.DB.Create(&grupBarang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Grup Barang berhasil ditambahkan", "data": grupBarang})
}

func GetAllGrupBarang(c *gin.Context) {
	var grups []models.GrupBarang
	if err := config.DB.Order("updated_at DESC").Find(&grups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data grup barang",
		"data":    grups,
	})
}

func GetGrupBarangByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var grup models.GrupBarang
	if err := config.DB.First(&grup, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kode grup tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil detail data grup barang", "data": grup})
}

func UpdateGrupBarang(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var grup models.GrupBarang
	if err := config.DB.First(&grup, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grup Barang tidak ditemukan"})
		return
	}

	// request body: hanya nama
	var input struct {
		Nama string `json:"nama" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama tidak valid"})
		return
	}

	// update hanya kolom nama
	if err := config.DB.Model(&grup).Update("nama", input.Nama).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// reload data terbaru
	config.DB.First(&grup, grup.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Nama Grup Barang berhasil diupdate",
		"data":    grup,
	})
}

func DeleteGrupBarang(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var grup models.GrupBarang
	if err := config.DB.First(&grup, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grup Barang tidak ditemukan"})
		return
	}

	if err := config.DB.Delete(&grup).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus Grup Barang"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Grup Barang berhasil dihapus"})
}
