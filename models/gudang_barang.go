package models

import "gorm.io/gorm"

type GudangBarang struct {
	gorm.Model
	GudangID uint   `json:"gudang_id"`
	Gudang   Gudang `gorm:"foreignKey:GudangID" json:"gudang"`

	BarangID uint   `json:"barang_id"`
	Barang   Barang `gorm:"foreignKey:BarangID" json:"barang"`

	LokasiSusun string  `json:"lokasi_susun"`
	HargaBeli   float64 `json:"harga_beli"`
	HargaJual   float64 `json:"harga_jual"`
	Stok        int     `json:"stok"`
}
