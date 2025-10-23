package models

import (
	"time"

	"gorm.io/gorm"
)

type Permintaan struct {
	gorm.Model
	Keterangan        string    `json:"keterangan"`
	NamaPeminta       string    `json:"nama_peminta"`
	KodePeminta       string    `json:"kode_peminta"`
	TanggalPermintaan time.Time `json:"tanggal_permintaan"`
	CreatedByID uint   `json:"created_by_id" gorm:"index"`
}
