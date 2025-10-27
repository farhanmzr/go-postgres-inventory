package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CreateBarang(c *gin.Context) {
	var input struct {
		Nama         string  `json:"nama"`
		Kode         string  `json:"kode"`
		GudangID     uint    `json:"gudang_id"`
		LokasiSusun  string  `json:"lokasi_susun"`
		Satuan       string  `json:"satuan"`
		Merek        string  `json:"merek"`
		MadeIn       string  `json:"made_in"`
		GrupBarangID uint    `json:"grup_barang_id"`
		HargaBeli    float64 `json:"harga_beli"`
		HargaJual    float64 `json:"harga_jual"`
		Stok         int     `json:"stok"`
		StokMinimal  int     `json:"stok_minimal"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Cek apakah kode barang sudah ada
	var exist models.Barang
	if err := config.DB.Where("kode = ?", input.Kode).First(&exist).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode barang sudah digunakan"})
		return
	}

	barang := models.Barang{
		Nama:         input.Nama,
		Kode:         input.Kode,
		GudangID:     input.GudangID,
		LokasiSusun:  input.LokasiSusun,
		Satuan:       input.Satuan,
		Merek:        input.Merek,
		MadeIn:       input.MadeIn,
		GrupBarangID: input.GrupBarangID,
		HargaBeli:    input.HargaBeli,
		HargaJual:    input.HargaJual,
		Stok:         input.Stok,
		StokMinimal:  input.StokMinimal,
	}

	// Simpan ke DB
	if err := config.DB.Create(&barang).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ambil lagi dari DB dengan Preload
	if err := config.DB.Preload("Gudang").Preload("GrupBarang").First(&barang, barang.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil ditambahkan", "data": barang})
}

func GetAllBarang(c *gin.Context) {
	var barangs []models.Barang
	if err := config.DB.Preload("Gudang").Preload("GrupBarang").Find(&barangs).Error; err != nil {
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
	if err := config.DB.Preload("Gudang").Preload("GrupBarang").First(&barang, id).Error; err != nil {
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
		Nama         string  `json:"nama"`
		Kode         string  `json:"kode"`
		GudangID     uint    `json:"gudang_id"`
		LokasiSusun  string  `json:"lokasi_susun"`
		Satuan       string  `json:"satuan"`
		Merek        string  `json:"merek"`
		MadeIn       string  `json:"made_in"`
		GrupBarangID uint    `json:"grup_barang_id"`
		HargaBeli    float64 `json:"harga_beli"`
		HargaJual    float64 `json:"harga_jual"`
		Stok         int     `json:"stok"`
		StokMinimal  int     `json:"stok_minimal"`
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

	updateData := models.Barang{
		Nama:         input.Nama,
		Kode:         input.Kode,
		GudangID:     input.GudangID,
		LokasiSusun:  input.LokasiSusun,
		Satuan:       input.Satuan,
		Merek:        input.Merek,
		MadeIn:       input.MadeIn,
		GrupBarangID: input.GrupBarangID,
		HargaBeli:    input.HargaBeli,
		HargaJual:    input.HargaJual,
		Stok:         input.Stok,
		StokMinimal:  input.StokMinimal,
	}

	if err := config.DB.Model(&barang).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	config.DB.Preload("Gudang").Preload("GrupBarang").First(&barang, barang.ID)


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


// response ringkas untuk list barang di gudang
type BarangSimple struct {
	ID        uint    `json:"id"`
	Nama      string  `json:"nama"`
	HargaBeli float64 `json:"harga_beli"`
	HargaJual float64 `json:"harga_jual"`
	Stok      int     `json:"stok"`
	Satuan    string  `json:"satuan"`
	Kode      string  `json:"kode"`
}

// GET /gudang/:id/barang?q=...&page=1&limit=50
func BarangByGudang(c *gin.Context) {
	// --- parse gudang id ---
	gidStr := c.Param("id")
	gid, err := strconv.ParseUint(gidStr, 10, 64)
	if err != nil || gid == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "gudang_id tidak valid"})
		return
	}

	// --- optional query params ---
	q := strings.TrimSpace(c.Query("q"))

	page := 1
	limit := 50
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	offset := (page - 1) * limit

	// --- build query ---
	db := config.DB.Model(&models.Barang{}).
		Where("gudang_id = ?", gid)

	if q != "" {
		like := "%" + q + "%"
		// Postgres -> ILIKE (case-insensitive)
		db = db.Where("(nama ILIKE ? OR kode ILIKE ?)", like, like)
	}

	var rows []BarangSimple
	if err := db.
		Select("id, nama, harga_beli, harga_jual, stok, satuan, kode").
		Order("nama ASC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal mengambil daftar barang",
			"error":   err.Error(),
		})
		return
	}

	// (opsional) total untuk kebutuhan pagination front-end
	var total int64
	if err := config.DB.Model(&models.Barang{}).
		Where("gudang_id = ?", gid).
		Scopes(func(tx *gorm.DB) *gorm.DB {
			if q == "" {
				return tx
			}
			like := "%" + q + "%"
			return tx.Where("(nama ILIKE ? OR kode ILIKE ?)", like, like)
		}).Count(&total).Error; err != nil {
		// kalau count gagal, tetap kirim data tanpa total
		total = int64(len(rows))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  rows,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

