package models

import "gorm.io/gorm"

type Barang struct {
	gorm.Model
	Nama         string     `json:"nama"`
	Kode         string     `json:"kode"`
	LokasiSusun  string     `json:"lokasi_susun"`
	Satuan       string     `json:"satuan"`
	Merek        string     `json:"merek"`
	MadeIn       string     `json:"made_in"`
	GrupBarangID uint       `json:"grup_barang_id"`                             // foreign key ke GrupBarang
	GrupBarang   GrupBarang `gorm:"foreignKey:GrupBarangID" json:"grup_barang"` // preload
	StokMinimal  int        `json:"stok_minimal"`
}
