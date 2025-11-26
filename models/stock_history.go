package models

import (
	"gorm.io/gorm"
)

type StockHistory struct {
	gorm.Model
	GudangBarangID uint         `json:"gudang_barang_id"`
	GudangBarang   GudangBarang `gorm:"foreignKey:GudangBarangID" json:"gudang_barang"`

	OldStok int    `json:"old_stok"`
	NewStok int    `json:"new_stok"`
	Selisih int    `json:"selisih"`
	Alasan  string `json:"alasan"`

	CreatedByID uint `json:"created_by_id"`
	// optional: relasi ke user/admin
}
