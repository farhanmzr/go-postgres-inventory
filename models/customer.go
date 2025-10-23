package models

import "gorm.io/gorm"

type Customer struct {
    gorm.Model
    Nama   string `json:"nama"`
    Kode   string `json:"kode"`
    Seri string `json:"seri"`
}
