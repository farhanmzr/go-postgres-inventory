package models

import "gorm.io/gorm"

type Barang struct {
	gorm.Model
	Nama         string     `json:"nama"`
	Kode         string     `json:"kode" gorm:"unique"`
	GudangID     uint       `json:"gudang_id"` // foreign key ke Gudang
	Gudang       Gudang     `json:"gudang"`    // preload
	LokasiSusun  string     `json:"lokasi_susun"`
	Satuan       string     `json:"satuan"`
	Merek        string     `json:"merek"`
	MadeIn       string     `json:"made_in"`
	GrupBarangID uint       `json:"grup_barang_id"` // foreign key ke KodeGrupBarang
	GrupBarang   GrupBarang `json:"grup_barang"`    // preload
	HargaBeli    float64    `json:"harga_beli"`
	HargaJual    float64    `json:"harga_jual"`
	Stok         int        `json:"stok"`
	StokMinimal  int        `json:"stok_minimal"`
}
