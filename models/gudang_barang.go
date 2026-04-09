package models

import "gorm.io/gorm"

type GudangBarang struct {
	gorm.Model
	GudangID uint   `json:"gudang_id"`
	Gudang   Gudang `gorm:"foreignKey:GudangID" json:"gudang"`

	BarangID uint   `json:"barang_id"`
	Barang   Barang `gorm:"foreignKey:BarangID" json:"barang"`

	LokasiSusun string  `json:"lokasi_susun"`
	HargaBeli   int64 `json:"harga_beli"`
	HargaJual   int64 `json:"harga_jual"`
	Stok        int     `json:"stok"`
}
