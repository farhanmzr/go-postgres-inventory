package controllers

import (
	"net/http"
	"strconv"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"go-postgres-inventory/utils"

	"github.com/gin-gonic/gin"
)

func CreateGudang(c *gin.Context) {
	var input struct {
		Nama   string `json:"nama"`
		Kode   string `json:"kode"`
		Lokasi string `json:"lokasi"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	gudang := models.Gudang{
		Nama:   input.Nama,
		Kode:   input.Kode,
		Lokasi: input.Lokasi,
	}

	if err := config.DB.Create(&gudang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gudang berhasil ditambahkan", "data": gudang})
}

func GetAllGudang(c *gin.Context) {
	var gudangs []models.Gudang
	if err := config.DB.Find(&gudangs).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, "Gagal mengambil data gudang", err)
		return
	}
	utils.Success(c, "Berhasil mengambil data gudang", gudangs)
}

func GetGudangByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var gudang models.Gudang
	if err := config.DB.First(&gudang, id).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, "Gudang tidak ditemukan", err)
		return
	}

	utils.Success(c, "Berhasil mengambil detail data gudang", gudang)
}

func UpdateGudang(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var gudang models.Gudang
	if err := config.DB.First(&gudang, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gudang tidak ditemukan"})
		return
	}

	var input struct {
		Nama   string `json:"nama"`
		Kode   string `json:"kode"`
		Lokasi string `json:"lokasi"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	updateData := models.Gudang{
		Nama:   input.Nama,
		Kode:   input.Kode,
		Lokasi: input.Lokasi,
	}

	if err := config.DB.Model(&gudang).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gudang berhasil diupdate", "data": gudang})
}

func DeleteGudang(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var gudang models.Gudang
	if err := config.DB.First(&gudang, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gudang tidak ditemukan"})
		return
	}

	if err := config.DB.Delete(&gudang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus gudang"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gudang berhasil dihapus"})
}
