package service


import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// ===== DTO umum =====

type BarangReportRow struct {
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
	NilaiBeli    float64 `json:"nilai_beli"` // HargaBeli * Stok
	NilaiJual    float64 `json:"nilai_jual"` // HargaJual * Stok
	StatusStok   string  `json:"status_stok"`
}

// ===== Filter laporan per barang (semua barang) =====

type BarangReportFilter struct {
	Query    string // cari di nama/kode/merek
	Merek    string
	MinStok  *int
	MaxStok  *int
	Page     int  // 1-based
	PageSize int  // default 50
	SortBy   string // "nama","-nama","kode","-kode","stok","-stok"
}

// ===== DTO laporan stok per grup =====

type StockBarangRow struct {
	BarangID uint   `json:"barang_id"`
	Nama     string `json:"nama"`
	Kode     string `json:"kode"`
	Satuan   string `json:"satuan"`
	Stok     int    `json:"stok"`
}

type StockGrupSummary struct {
	GrupID     uint   `json:"grup_id"`
	GrupNama   string `json:"grup_nama"`
	TotalStok  int    `json:"total_stok"`
	JumlahItem int64  `json:"jumlah_item"`
	// Optional: nilai persediaan (kalau mau)
	// NilaiBeli int64 `json:"nilai_beli"`
}

type StockGrupReport struct {
	Summary StockGrupSummary `json:"summary"`
	Items   []StockBarangRow `json:"items"`
}

// ===== DTO laporan stok per gudang =====

type StockGudangSummary struct {
	GudangID   uint   `json:"gudang_id"`
	GudangNama string `json:"gudang_nama"`
	TotalStok  int    `json:"total_stok"`
	JumlahItem int64  `json:"jumlah_item"`
}

type StockGudangReport struct {
	Summary StockGudangSummary `json:"summary"`
	Items   []StockBarangRow   `json:"items"`
}


// ===== Service =====

type Service interface {
	// 1) Semua barang
	LaporanPerBarang(ctx context.Context, f BarangReportFilter) ([]BarangReportRow, int64, error)

	// 2) Stok per grup barang — wajib grupID
	LaporanStockPerGrup(ctx context.Context, grupID uint, page, pageSize int, sortBy string) (StockGrupReport, error)

	// 3) Stok per gudang — wajib gudangID
	LaporanStockPerGudang(ctx context.Context, gudangID uint, page, pageSize int, sortBy string) (StockGudangReport, error)
}

type service struct{ db *gorm.DB }

func NewService(db *gorm.DB) Service { return &service{db: db} }

// ===== Implementations =====

// 1) Semua barang
func (s *service) LaporanPerBarang(ctx context.Context, f BarangReportFilter) ([]BarangReportRow, int64, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 500 {
		f.PageSize = 50
	}

	q := s.db.WithContext(ctx).
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
		// inner join karena FK wajib ada
		Joins("INNER JOIN grup_barangs gb ON gb.id = barangs.grup_barang_id").
		Joins("INNER JOIN gudangs gd ON gd.id = barangs.gudang_id")

	// Filters
	if f.Query != "" {
		like := "%" + f.Query + "%"
		q = q.Where(`barangs.nama ILIKE ? OR barangs.kode ILIKE ? OR barangs.merek ILIKE ?`, like, like, like)
	}
	if f.Merek != "" {
		q = q.Where("barangs.merek ILIKE ?", "%"+f.Merek+"%")
	}
	if f.MinStok != nil {
		q = q.Where("barangs.stok >= ?", *f.MinStok)
	}
	if f.MaxStok != nil {
		q = q.Where("barangs.stok <= ?", *f.MaxStok)
	}

	// Count
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sorting
	switch f.SortBy {
	case "nama":
		q = q.Order("barangs.nama ASC")
	case "-nama":
		q = q.Order("barangs.nama DESC")
	case "kode":
		q = q.Order("barangs.kode ASC")
	case "-kode":
		q = q.Order("barangs.kode DESC")
	case "stok":
		q = q.Order("barangs.stok ASC")
	case "-stok":
		q = q.Order("barangs.stok DESC")
	default:
		q = q.Order("barangs.id DESC")
	}

	// Pagination
	offset := (f.Page - 1) * f.PageSize
	var rows []BarangReportRow
	if err := q.Offset(offset).Limit(f.PageSize).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// 2) Stok per grup (WAJIB grupID) — ringkasan + breakdown per barang
func (s *service) LaporanStockPerGrup(ctx context.Context, grupID uint, page, pageSize int, sortBy string) (StockGrupReport, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 1000 {
		pageSize = 200
	}
	// Summary
	var sum StockGrupSummary
	if err := s.db.WithContext(ctx).
		Table("barangs").
		Select(`
			gb.id   AS grup_id,
			gb.nama AS grup_nama,
			SUM(barangs.stok)     AS total_stok,
			COUNT(barangs.id)     AS jumlah_item
		`).
		Joins("INNER JOIN grup_barangs gb ON gb.id = barangs.grup_barang_id").
		Where("barangs.grup_barang_id = ?", grupID).
		Group("gb.id, gb.nama").
		Scan(&sum).Error; err != nil {
		return StockGrupReport{}, err
	}
	if sum.GrupID == 0 {
		return StockGrupReport{}, fmt.Errorf("grup_id %d tidak ditemukan atau tidak ada barang", grupID)
	}

	// Items (breakdown)
	itemsQ := s.db.WithContext(ctx).
		Table("barangs").
		Select(`
			barangs.id   AS barang_id,
			barangs.nama AS nama,
			barangs.kode AS kode,
			barangs.satuan AS satuan,
			barangs.stok AS stok
		`).
		Where("barangs.grup_barang_id = ?", grupID)

	switch sortBy {
	case "nama":
		itemsQ = itemsQ.Order("barangs.nama ASC")
	case "-nama":
		itemsQ = itemsQ.Order("barangs.nama DESC")
	case "kode":
		itemsQ = itemsQ.Order("barangs.kode ASC")
	case "-kode":
		itemsQ = itemsQ.Order("barangs.kode DESC")
	case "stok":
		itemsQ = itemsQ.Order("barangs.stok ASC")
	case "-stok":
		itemsQ = itemsQ.Order("barangs.stok DESC")
	default:
		itemsQ = itemsQ.Order("barangs.id DESC")
	}

	offset := (page - 1) * pageSize
	var items []StockBarangRow
	if err := itemsQ.Offset(offset).Limit(pageSize).Scan(&items).Error; err != nil {
		return StockGrupReport{}, err
	}

	return StockGrupReport{
		Summary: sum,
		Items:   items,
	}, nil
}

// 3) Stok per gudang (WAJIB gudangID) — ringkasan + breakdown per barang
func (s *service) LaporanStockPerGudang(ctx context.Context, gudangID uint, page, pageSize int, sortBy string) (StockGudangReport, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 1000 {
		pageSize = 200
	}

	// Summary
	var sum StockGudangSummary
	if err := s.db.WithContext(ctx).
		Table("barangs").
		Select(`
			gd.id   AS gudang_id,
			gd.nama AS gudang_nama,
			SUM(barangs.stok)     AS total_stok,
			COUNT(barangs.id)     AS jumlah_item
		`).
		Joins("INNER JOIN gudangs gd ON gd.id = barangs.gudang_id").
		Where("barangs.gudang_id = ?", gudangID).
		Group("gd.id, gd.nama").
		Scan(&sum).Error; err != nil {
		return StockGudangReport{}, err
	}
	if sum.GudangID == 0 {
		return StockGudangReport{}, fmt.Errorf("gudang_id %d tidak ditemukan atau tidak ada barang", gudangID)
	}

	// Items
	itemsQ := s.db.WithContext(ctx).
		Table("barangs").
		Select(`
			barangs.id   AS barang_id,
			barangs.nama AS nama,
			barangs.kode AS kode,
			barangs.satuan AS satuan,
			barangs.stok AS stok
		`).
		Where("barangs.gudang_id = ?", gudangID)

	switch sortBy {
	case "nama":
		itemsQ = itemsQ.Order("barangs.nama ASC")
	case "-nama":
		itemsQ = itemsQ.Order("barangs.nama DESC")
	case "kode":
		itemsQ = itemsQ.Order("barangs.kode ASC")
	case "-kode":
		itemsQ = itemsQ.Order("barangs.kode DESC")
	case "stok":
		itemsQ = itemsQ.Order("barangs.stok ASC")
	case "-stok":
		itemsQ = itemsQ.Order("barangs.stok DESC")
	default:
		itemsQ = itemsQ.Order("barangs.id DESC")
	}

	offset := (page - 1) * pageSize
	var items []StockBarangRow
	if err := itemsQ.Offset(offset).Limit(pageSize).Scan(&items).Error; err != nil {
		return StockGudangReport{}, err
	}

	return StockGudangReport{
		Summary: sum,
		Items:   items,
	}, nil
}