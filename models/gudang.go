package models

import "gorm.io/gorm"

type Gudang struct {
    gorm.Model
    Nama   string `json:"nama"`
    Kode   string `json:"kode"`
    Lokasi string `json:"lokasi"`
}
