package models

import "gorm.io/gorm"

type StockHistory struct {
	gorm.Model
	BarangID uint   `json:"barang_id"`
	Barang   Barang `gorm:"foreignKey:BarangID" json:"barang"`

	OldStok     int    `json:"old_stok"`
	NewStok     int    `json:"new_stok"`
	Selisih     int    `json:"selisih"` // new - old
	Alasan      string `json:"alasan"`
	CreatedByID uint   `json:"created_by_id"`
}
