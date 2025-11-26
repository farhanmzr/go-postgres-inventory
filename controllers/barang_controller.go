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
		Nama         string `json:"nama"`
		Kode         string `json:"kode"`
		Satuan       string `json:"satuan"`
		Merek        string `json:"merek"`
		MadeIn       string `json:"made_in"`
		GrupBarangID uint   `json:"grup_barang_id"`
		StokMinimal  int    `json:"stok_minimal"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Cek apakah kode barang sudah ada di master
	var exist models.Barang
	if err := config.DB.Where("kode = ?", input.Kode).First(&exist).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode barang sudah digunakan"})
		return
	}

	barang := models.Barang{
		Nama:         input.Nama,
		Kode:         input.Kode,
		Satuan:       input.Satuan,
		Merek:        input.Merek,
		MadeIn:       input.MadeIn,
		GrupBarangID: input.GrupBarangID,
		StokMinimal:  input.StokMinimal,
	}

	if err := config.DB.Create(&barang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Preload("GrupBarang").First(&barang, barang.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Barang master berhasil ditambahkan",
		"data":    barang,
	})
}

func GetAllBarang(c *gin.Context) {
	var barangs []models.Barang
	if err := config.DB.Preload("GrupBarang").Find(&barangs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil data Barang", "data": barangs})
}

func GetBarangByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var barang models.Barang
	if err := config.DB.Preload("GrupBarang").First(&barang, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Barang tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil detail data Barang", "data": barang})
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
		Nama         string `json:"nama"`
		Kode         string `json:"kode"`
		Satuan       string `json:"satuan"`
		Merek        string `json:"merek"`
		MadeIn       string `json:"made_in"`
		GrupBarangID uint   `json:"grup_barang_id"`
		StokMinimal  int    `json:"stok_minimal"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Optional: cek kode baru tidak duplikat
	if input.Kode != barang.Kode {
		var exist models.Barang
		if err := config.DB.Where("kode = ?", input.Kode).First(&exist).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode barang sudah digunakan"})
			return
		}
	}

	// Pakai map supaya lebih fleksibel, dan tidak sentuh field stok
	updateData := map[string]any{
		"nama":           input.Nama,
		"kode":           input.Kode,
		"satuan":         input.Satuan,
		"merek":          input.Merek,
		"made_in":        input.MadeIn,
		"grup_barang_id": input.GrupBarangID,
		"stok_minimal":   input.StokMinimal,
	}

	if err := config.DB.Model(&barang).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	config.DB.Preload("GrupBarang").First(&barang, barang.ID)

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

	// === CEK DI PEMBELIAN ===
	var count int64
	if err := config.DB.Model(&models.PurchaseReqItem{}).
		Where("barang_id = ?", barang.ID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal cek relasi pembelian"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Barang sudah pernah dipakai di transaksi PEMBELIAN, tidak bisa dihapus",
		})
		return
	}

	// === CEK DI PENJUALAN (SESUIKAN NAMA MODELINYA) ===
	// Contoh: models.SalesItem
	if err := config.DB.Model(&models.SalesReqItem{}).
		Where("barang_id = ?", barang.ID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal cek relasi penjualan"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Barang sudah pernah dipakai di transaksi PENJUALAN, tidak bisa dihapus",
		})
		return
	}

	// === CEK DI PEMAKAIAN ===
	if err := config.DB.Model(&models.UsageItem{}).
		Where("barang_id = ?", barang.ID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal cek relasi pemakaian"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Barang sudah pernah dipakai di transaksi PEMAKAIAN, tidak bisa dihapus",
		})
		return
	}

	// Kalau lolos semua cek, aman untuk dihapus
	if err := config.DB.Delete(&barang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hapus barang"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil dihapus"})
}
