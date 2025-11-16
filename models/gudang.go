package models

import "gorm.io/gorm"

type Gudang struct {
	gorm.Model
	Nama     string `json:"nama"`
	Kode     string `json:"kode"`
	Lokasi   string `json:"lokasi"`
	Kas      string `json:"kas"`
	KodeKas  string `json:"kode_kas"`
	Bank     string `json:"bank"`
	KodeBank string `json:"kode_bank"`
}
