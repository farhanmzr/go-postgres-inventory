package models

import "gorm.io/gorm"

type Supplier struct {
    gorm.Model
    Nama string `json:"nama"`
    Kode string `json:"kode"`
}