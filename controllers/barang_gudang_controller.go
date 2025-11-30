package controllers

import (
	"errors"
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GudangBarangCreateInput struct {
	BarangID uint `json:"barang_id" binding:"required"`
}

// POST /gudang/:gudang_id/barang
func TambahBarangKeGudang(c *gin.Context) {
	// ambil gudang_id dari path
	gudangIDStr := c.Param("gudang_id")
	if gudangIDStr == "" {
		gudangIDStr = c.Param("id") // fallback kalau route pakai :id
	}
	gudangID64, err := strconv.ParseUint(gudangIDStr, 10, 64)
	if err != nil || gudangID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gudang_id tidak valid"})
		return
	}

	gudangID := uint(gudangID64)

	// body: barang_id
	var in GudangBarangCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Payload tidak valid",
			"detail": err.Error(),
		})
		return
	}

	// cek gudang
	var g models.Gudang
	if err := config.DB.First(&g, gudangID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gudang tidak ditemukan"})
		return
	}

	// cek barang master
	var b models.Barang
	if err := config.DB.First(&b, in.BarangID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Barang master tidak ditemukan"})
		return
	}

	// cek apakah sudah ada gudang-barang ini
	var exist models.GudangBarang
	if err := config.DB.
		Where("gudang_id = ? AND barang_id = ?", gudangID, in.BarangID).
		First(&exist).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Barang sudah ada di gudang ini"})
		return
	}

	// buat dengan nilai awal 0
	gb := models.GudangBarang{
		GudangID:    gudangID,
		BarangID:    in.BarangID,
		LokasiSusun: "",
		HargaBeli:   0,
		HargaJual:   0,
		Stok:        0,
	}

	if err := config.DB.Create(&gb).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// preload untuk response
	if err := config.DB.
		Preload("Gudang").
		Preload("Barang").
		Preload("Barang.GrupBarang").
		First(&gb, gb.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Barang berhasil ditambahkan ke gudang",
		"data":    gb,
	})
}

func GetGudangBarangList(c *gin.Context) {
	db := config.DB

	gudangIDStr := c.Param("gudang_id")
	if gudangIDStr == "" {
		gudangIDStr = c.Param("id") // fallback kalau route pakai :id
	}
	gudangID64, err := strconv.ParseUint(gudangIDStr, 10, 64)
	if err != nil || gudangID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gudang_id tidak valid"})
		return
	}

	gudangID := uint(gudangID64)

	// opsional: cek gudang ada
	var cnt int64
	if err := db.Model(&models.Gudang{}).
		Where("id = ?", gudangID).
		Count(&cnt).Error; err != nil || cnt == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gudang tidak ditemukan"})
		return
	}

	page := getInt(c, "page", 1)
	size := getInt(c, "page_size", 100)
	offset := (page - 1) * size

	qstr := strings.TrimSpace(c.Query("q")) // search nama/kode barang

	q := db.Model(&models.GudangBarang{}).
		Where("gudang_id = ?", gudangID).
		Preload("Gudang").
		Preload("Barang").
		Preload("Barang.GrupBarang")

	if qstr != "" {
		like := "%" + qstr + "%"
		q = q.
			Joins("JOIN barangs ON barangs.id = gudang_barangs.barang_id").
			Where("barangs.nama ILIKE ? OR barangs.kode ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var rows []models.GudangBarang
	if err := q.
		Order("id DESC").
		Offset(offset).
		Limit(size).
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rows,
		"pagination": gin.H{
			"page":      page,
			"page_size": size,
			"total":     total,
		},
	})
}

func GetGudangBarangByID(c *gin.Context) {
	gudangIDStr := c.Param("gudang_id")
	if gudangIDStr == "" {
		gudangIDStr = c.Param("id") // fallback kalau route pakai :id
	}
	gudangID64, err := strconv.ParseUint(gudangIDStr, 10, 64)
	if err != nil || gudangID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gudang_id tidak valid"})
		return
	}
	id := uint(gudangID64)

	var gb models.GudangBarang
	if err := config.DB.
		Preload("Gudang").
		Preload("Barang").
		First(&gb, uint(id)).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gb})
}

type GudangBarangUpdateInput struct {
	LokasiSusun *string  `json:"lokasi_susun"`
	HargaBeli   *float64 `json:"harga_beli"`
	HargaJual   *float64 `json:"harga_jual"`
}

func UpdateGudangBarang(c *gin.Context) {
	gudangIDStr := c.Param("gudang_id")
	if gudangIDStr == "" {
		gudangIDStr = c.Param("id") // fallback kalau route pakai :id
	}
	gudangID64, err := strconv.ParseUint(gudangIDStr, 10, 64)
	if err != nil || gudangID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gudang_id tidak valid"})
		return
	}
	id := uint(gudangID64)

	var in GudangBarangUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid", "detail": err.Error()})
		return
	}

	updates := map[string]any{}
	if in.LokasiSusun != nil {
		updates["lokasi_susun"] = *in.LokasiSusun
	}
	if in.HargaBeli != nil {
		updates["harga_beli"] = *in.HargaBeli
	}
	if in.HargaJual != nil {
		updates["harga_jual"] = *in.HargaJual
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada field yang diupdate"})
		return
	}

	if err := config.DB.Model(&models.GudangBarang{}).
		Where("id = ?", uint(id)).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var gb models.GudangBarang
	if err := config.DB.
		Preload("Gudang").
		Preload("Barang").
		First(&gb, uint(id)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data berhasil diupdate",
		"data":    gb,
	})
}

func UpdateStokBarang(c *gin.Context) {
	gudangIDStr := c.Param("gudang_id")
	if gudangIDStr == "" {
		gudangIDStr = c.Param("id") // fallback kalau route pakai :id
	}
	gudangID64, err := strconv.ParseUint(gudangIDStr, 10, 64)
	if err != nil || gudangID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gudang_id tidak valid"})
		return
	}
	id := uint(gudangID64)

	// ambil data gudang_barang
	var gb models.GudangBarang
	if err := config.DB.First(&gb, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Barang di gudang tidak ditemukan"})
		return
	}

	var input struct {
		Stok   int    `json:"stok" binding:"required"`
		Alasan string `json:"alasan" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	uid, err := currentAdminID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized",
			"error":   err.Error(),
		})
		return
	}

	oldStok := gb.Stok
	newStok := input.Stok

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		// update stok di gudang_barang
		if err := tx.Model(&models.GudangBarang{}).
			Where("id = ?", gb.ID).
			Update("stok", newStok).Error; err != nil {
			return err
		}

		// simpan history stok
		history := models.StockHistory{
			GudangBarangID: gb.ID,
			OldStok:        oldStok,
			NewStok:        newStok,
			Selisih:        newStok - oldStok,
			Alasan:         input.Alasan,
			CreatedByID:    uid,
		}

		if err := tx.Create(&history).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// reload gudang_barang + relasi biar data lengkap untuk response
	if err := config.DB.
		Preload("Gudang").
		Preload("Barang").
		Preload("Barang.GrupBarang").
		First(&gb, gb.ID).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Stok barang di gudang berhasil diupdate",
		"data":    gb,
	})
}

func GetStockHistoryByBarang(c *gin.Context) {
	gudangIDStr := c.Param("gudang_id")
	if gudangIDStr == "" {
		gudangIDStr = c.Param("id") // fallback kalau route pakai :id
	}
	gudangID64, err := strconv.ParseUint(gudangIDStr, 10, 64)
	if err != nil || gudangID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gudang_id tidak valid"})
		return
	}
	gudangBarangID := uint(gudangID64)

	// Optional: pagination via ?page=1&limit=20
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var histories []models.StockHistory
	var total int64

	baseQuery := config.DB.Model(&models.StockHistory{}).
		Where("gudang_barang_id = ?", gudangBarangID)

	// Hitung total data
	if err := baseQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Ambil data history
	if err := baseQuery.
		Preload("GudangBarang").
		Preload("GudangBarang.Barang").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&histories).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "History stok barang per gudang",
		"data":    histories,
		"page":    page,
		"limit":   limit,
		"total":   total,
	})
}

// DELETE /gudang-barang/:id
func DeleteGudangBarang(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak boleh kosong"})
		return
	}

	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	id := uint(id64)

	var gb models.GudangBarang
	if err := config.DB.First(&gb, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data gudang-barang tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// OPTIONAL: larang hapus kalau stok masih ada
	// if gb.Stok != 0 {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error":   "Tidak bisa menghapus karena stok masih ada",
	// 		"message": fmt.Sprintf("Stok saat ini: %d", gb.Stok),
	// 	})
	// 	return
	// }

	if err := config.DB.Delete(&gb).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Barang di gudang berhasil dihapus",
		"id":      id,
	})
}

