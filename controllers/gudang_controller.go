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
		Nama     string `json:"nama"`
		Kode     string `json:"kode"`
		Lokasi   string `json:"lokasi"`
		Kas      string `json:"kas"`
		KodeKas  string `json:"kode_kas"`
		Bank     string `json:"bank"`
		KodeBank string `json:"kode_bank"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Cek apakah kode gudang sudah ada
	var exist models.Gudang
	if err := config.DB.Where("kode = ?", input.Kode).First(&exist).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode gudang sudah digunakan"})
		return
	}

	gudang := models.Gudang{
		Nama:     input.Nama,
		Kode:     input.Kode,
		Lokasi:   input.Lokasi,
		Kas:      input.Kas,
		KodeKas:  input.KodeKas,
		Bank:     input.Bank,
		KodeBank: input.KodeBank,
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
		Nama     string `json:"nama"`
		Kode     string `json:"kode"`
		Lokasi   string `json:"lokasi"`
		Kas      string `json:"kas"`
		KodeKas  string `json:"kode_kas"`
		Bank     string `json:"bank"`
		KodeBank string `json:"kode_bank"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Cek apakah kode gudang sudah ada
	var exist models.Gudang
	if err := config.DB.Where("kode = ?", input.Kode).First(&exist).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode gudang sudah digunakan"})
		return
	}

	updateData := models.Gudang{
		Nama:     input.Nama,
		Kode:     input.Kode,
		Lokasi:   input.Lokasi,
		Kas:      input.Kas,
		KodeKas:  input.KodeKas,
		Bank:     input.Bank,
		KodeBank: input.KodeBank,
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

	var count int64

	// 1) Cek masih ada barang di gudang ini atau tidak
	if err := config.DB.Model(&models.Barang{}).
		Where("gudang_id = ?", gudang.ID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal cek barang di gudang"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Gudang tidak bisa dihapus karena masih ada barang di dalamnya",
		})
		return
	}

	// 2) Cek sudah pernah dipakai di PEMBELIAN
	if err := config.DB.Model(&models.PurchaseRequest{}).
		Where("warehouse_id = ?", gudang.ID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal cek transaksi pembelian"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Gudang sudah pernah dipakai di transaksi PEMBELIAN, tidak bisa dihapus",
		})
		return
	}

	// 3) Cek sudah pernah dipakai di PEMAKAIAN
	if err := config.DB.Model(&models.UsageRequest{}).
		Where("warehouse_id = ?", gudang.ID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal cek transaksi pemakaian"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Gudang sudah pernah dipakai di transaksi PEMAKAIAN, tidak bisa dihapus",
		})
		return
	}

	// 4) Cek PENJUALAN kalau ada field warehouse_id di sana
	if err := config.DB.Model(&models.SalesRequest{}).
		Where("warehouse_id = ?", gudang.ID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal cek transaksi penjualan"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Gudang sudah pernah dipakai di transaksi PENJUALAN, tidak bisa dihapus",
		})
		return
	}

	// Kalau semua cek lolos -> aman untuk dihapus
	if err := config.DB.Delete(&gudang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus gudang"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gudang berhasil dihapus"})
}


