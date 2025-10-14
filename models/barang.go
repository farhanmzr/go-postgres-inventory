package models

import "gorm.io/gorm"

type Barang struct {
	gorm.Model
	NamaBarang string `json:"nama_barang"`
	Stok       int    `json:"stok"`
	Lokasi     string `json:"lokasi"`
}
