package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"go-postgres-inventory/config"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ================= DTO =================

type barangReportRow struct {
	ID           uint    `json:"id"`
	Nama         string  `json:"nama"`
	Kode         string  `json:"kode"`
	Satuan       string  `json:"satuan"`
	Merek        string  `json:"merek"`
	MadeIn       string  `json:"made_in"`
	GrupID       uint    `json:"grup_id"`
	GrupNama     string  `json:"grup_nama"`
	GudangID     uint    `json:"gudang_id"`
	GudangNama   string  `json:"gudang_nama"`
	LokasiSusun  string  `json:"lokasi_susun"`
	HargaBeli    float64 `json:"harga_beli"`
	HargaJual    float64 `json:"harga_jual"`
	Stok         int     `json:"stok"`
	StokMinimal  int     `json:"stok_minimal"`
	NilaiBeli    float64 `json:"nilai_beli"`
	NilaiJual    float64 `json:"nilai_jual"`
	StatusStok   string  `json:"status_stok"`
}

type stockBarangRow struct {
	BarangID uint   `json:"barang_id"`
	Nama     string `json:"nama"`
	Kode     string `json:"kode"`
	Satuan   string `json:"satuan"`
	Stok     int    `json:"stok"`
}

type stockGrupSummary struct {
	GrupID     uint   `json:"grup_id"`
	GrupNama   string `json:"grup_nama"`
	TotalStok  int    `json:"total_stok"`
	JumlahItem int64  `json:"jumlah_item"`
}

type stockGudangSummary struct {
	GudangID   uint   `json:"gudang_id"`
	GudangNama string `json:"gudang_nama"`
	TotalStok  int    `json:"total_stok"`
	JumlahItem int64  `json:"jumlah_item"`
}

// =============== Helpers ===============

func qSort(q *gorm.DB, sortBy string, fields map[string]string) *gorm.DB {
	switch sortBy {
	case "nama":
		return q.Order(fields["nama"] + " ASC")
	case "-nama":
		return q.Order(fields["nama"] + " DESC")
	case "kode":
		return q.Order(fields["kode"] + " ASC")
	case "-kode":
		return q.Order(fields["kode"] + " DESC")
	case "stok":
		return q.Order(fields["stok"] + " ASC")
	case "-stok":
		return q.Order(fields["stok"] + " DESC")
	default:
		return q.Order(fields["default"] + " DESC")
	}
}

func getInt(c *gin.Context, key string, def int) int {
	v, _ := strconv.Atoi(c.DefaultQuery(key, strconv.Itoa(def)))
	if v <= 0 {
		return def
	}
	return v
}

// =======================================
// ==========   CONTROLLERS   ============
// =======================================

// GET .../reports/barang?q=&merek=&min_stok=&max_stok=&sort=&page=&page_size=
func ReportBarang(c *gin.Context) {
	db := config.DB

	page := getInt(c, "page", 1)
	size := getInt(c, "page_size", 50)
	sortBy := c.DefaultQuery("sort", "")

	var minStokPtr *int
	if v := c.Query("min_stok"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			minStokPtr = &n
		}
	}
	var maxStokPtr *int
	if v := c.Query("max_stok"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxStokPtr = &n
		}
	}

	q := db.
		Table("barangs").
		Select(`
			barangs.id,
			barangs.nama,
			barangs.kode,
			barangs.satuan,
			barangs.merek,
			barangs.made_in,
			barangs.grup_barang_id AS grup_id,
			gb.nama AS grup_nama,
			barangs.gudang_id AS gudang_id,
			gd.nama AS gudang_nama,
			barangs.lokasi_susun,
			barangs.harga_beli,
			barangs.harga_jual,
			barangs.stok,
			barangs.stok_minimal,
			(barangs.harga_beli * barangs.stok) AS nilai_beli,
			(barangs.harga_jual * barangs.stok) AS nilai_jual,
			CASE WHEN barangs.stok < barangs.stok_minimal THEN 'LOW' ELSE 'OK' END AS status_stok
		`).
		Joins("INNER JOIN grup_barangs gb ON gb.id = barangs.grup_barang_id").
		Joins("INNER JOIN gudangs gd ON gd.id = barangs.gudang_id")

	if qstr := strings.TrimSpace(c.Query("q")); qstr != "" {
		like := "%" + qstr + "%"
		q = q.Where(`barangs.nama ILIKE ? OR barangs.kode ILIKE ? OR barangs.merek ILIKE ?`, like, like, like)
	}
	if merek := strings.TrimSpace(c.Query("merek")); merek != "" {
		q = q.Where("barangs.merek ILIKE ?", "%"+merek+"%")
	}
	if minStokPtr != nil {
		q = q.Where("barangs.stok >= ?", *minStokPtr)
	}
	if maxStokPtr != nil {
		q = q.Where("barangs.stok <= ?", *maxStokPtr)
	}

	// total (pakai subquery biar aman dari LIMIT/OFFSET)
	var total int64
	if err := db.Table("(?) as sub", q.Session(&gorm.Session{})).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	q = qSort(q, sortBy, map[string]string{
		"nama":    "barangs.nama",
		"kode":    "barangs.kode",
		"stok":    "barangs.stok",
		"default": "barangs.id",
	})

	offset := (page - 1) * size
	var rows []barangReportRow
	if err := q.Offset(offset).Limit(size).Scan(&rows).Error; err != nil {
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

// GET .../reports/stock/grup/:id?sort=&page=&page_size=
func ReportStockPerGrup(c *gin.Context) {
	db := config.DB

	grupID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || grupID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "grup_id tidak valid"})
		return
	}
	grupID := uint(grupID64)
	page := getInt(c, "page", 1)
	size := getInt(c, "page_size", 200)
	sortBy := c.DefaultQuery("sort", "")

	// Summary
	var sum stockGrupSummary
	if err := db.
		Table("barangs").
		Select(`
			gb.id   AS grup_id,
			gb.nama AS grup_nama,
			SUM(barangs.stok) AS total_stok,
			COUNT(barangs.id) AS jumlah_item
		`).
		Joins("INNER JOIN grup_barangs gb ON gb.id = barangs.grup_barang_id").
		Where("barangs.grup_barang_id = ?", grupID).
		Group("gb.id, gb.nama").
		Scan(&sum).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sum.GrupID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "grup_id tidak ditemukan atau tidak ada barang"})
		return
	}

	// Items
	itemsQ := db.
		Table("barangs").
		Select(`
			barangs.id   AS barang_id,
			barangs.nama AS nama,
			barangs.kode AS kode,
			barangs.satuan AS satuan,
			barangs.stok AS stok
		`).
		Where("barangs.grup_barang_id = ?", grupID)

	itemsQ = qSort(itemsQ, sortBy, map[string]string{
		"nama":    "barangs.nama",
		"kode":    "barangs.kode",
		"stok":    "barangs.stok",
		"default": "barangs.id",
	})

	offset := (page - 1) * size
	var items []stockBarangRow
	if err := itemsQ.Offset(offset).Limit(size).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": sum,
		"items":   items,
	})
}

// GET .../reports/stock/gudang/:id?sort=&page=&page_size=
func ReportStockPerGudang(c *gin.Context) {
	db := config.DB

	gudangID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || gudangID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gudang_id tidak valid"})
		return
	}
	gudangID := uint(gudangID64)
	page := getInt(c, "page", 1)
	size := getInt(c, "page_size", 200)
	sortBy := c.DefaultQuery("sort", "")

	// Summary
	var sum stockGudangSummary
	if err := db.
		Table("barangs").
		Select(`
			gd.id   AS gudang_id,
			gd.nama AS gudang_nama,
			SUM(barangs.stok) AS total_stok,
			COUNT(barangs.id) AS jumlah_item
		`).
		Joins("INNER JOIN gudangs gd ON gd.id = barangs.gudang_id").
		Where("barangs.gudang_id = ?", gudangID).
		Group("gd.id, gd.nama").
		Scan(&sum).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sum.GudangID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "gudang_id tidak ditemukan atau tidak ada barang"})
		return
	}

	// Items
	itemsQ := db.
		Table("barangs").
		Select(`
			barangs.id   AS barang_id,
			barangs.nama AS nama,
			barangs.kode AS kode,
			barangs.satuan AS satuan,
			barangs.stok AS stok
		`).
		Where("barangs.gudang_id = ?", gudangID)

	itemsQ = qSort(itemsQ, sortBy, map[string]string{
		"nama":    "barangs.nama",
		"kode":    "barangs.kode",
		"stok":    "barangs.stok",
		"default": "barangs.id",
	})

	offset := (page - 1) * size
	var items []stockBarangRow
	if err := itemsQ.Offset(offset).Limit(size).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": sum,
		"items":   items,
	})
}
