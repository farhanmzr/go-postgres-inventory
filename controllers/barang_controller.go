package controllers

import (
	"net/http"
	"strconv"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
)

func CreateBarang(c *gin.Context) {
	var input struct {
		Nama        string  `json:"nama"`
		Kode        string  `json:"kode"`
		GudangID    uint    `json:"gudang_id"`
		LokasiSusun string  `json:"lokasi_susun"`
		Satuan      string  `json:"satuan"`
		Merek       string  `json:"merek"`
		MadeIn      string  `json:"made_in"`
		KodeGrupID  uint    `json:"kode_grup_id"`
		HargaBeli   float64 `json:"harga_beli"`
		HargaJual   float64 `json:"harga_jual"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	barang := models.Barang{
		Nama:        input.Nama,
		Kode:        input.Kode,
		GudangID:    input.GudangID,
		LokasiSusun: input.LokasiSusun,
		Satuan:      input.Satuan,
		Merek:       input.Merek,
		MadeIn:      input.MadeIn,
		KodeGrupID:  input.KodeGrupID,
		HargaBeli:   input.HargaBeli,
		HargaJual:   input.HargaJual,
	}

	if err := config.DB.Create(&barang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil ditambahkan", "data": barang})
}

func GetAllBarang(c *gin.Context) {
	var barangs []models.Barang
	if err := config.DB.Preload("Gudang").Preload("KodeGrup").Find(&barangs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": barangs})
}

func GetBarangByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var barang models.Barang
	if err := config.DB.Preload("Gudang").Preload("KodeGrup").First(&barang, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Barang tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": barang})
}

func UpdateBarang(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var barang models.Barang
	if err := config.DB.First(&barang, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Barang tidak ditemukan"})
		return
	}

	var input struct {
		Nama        string  `json:"nama"`
		Kode        string  `json:"kode"`
		GudangID    uint    `json:"gudang_id"`
		LokasiSusun string  `json:"lokasi_susun"`
		Satuan      string  `json:"satuan"`
		Merek       string  `json:"merek"`
		MadeIn      string  `json:"made_in"`
		KodeGrupID  uint    `json:"kode_grup_id"`
		HargaBeli   float64 `json:"harga_beli"`
		HargaJual   float64 `json:"harga_jual"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	updateData := models.Barang{
		Nama:        input.Nama,
		Kode:        input.Kode,
		GudangID:    input.GudangID,
		LokasiSusun: input.LokasiSusun,
		Satuan:      input.Satuan,
		Merek:       input.Merek,
		MadeIn:      input.MadeIn,
		KodeGrupID:  input.KodeGrupID,
		HargaBeli:   input.HargaBeli,
		HargaJual:   input.HargaJual,
	}

	if err := config.DB.Model(&barang).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil diupdate", "data": barang})
}

func DeleteBarang(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var barang models.Barang
	if err := config.DB.First(&barang, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Barang tidak ditemukan"})
		return
	}

	if err := config.DB.Delete(&barang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus barang"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil dihapus"})
}
