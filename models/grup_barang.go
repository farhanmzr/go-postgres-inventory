package models

import "gorm.io/gorm"

type GrupBarang struct {
    gorm.Model
    Nama string `json:"nama"`
    Kode string `json:"kode"`
}