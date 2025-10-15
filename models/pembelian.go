package models

import "time"

type Pembelian struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	BarangID  uint      `json:"barang_id"`
	Barang    Barang    `json:"barang" gorm:"foreignKey:BarangID"`
	Jumlah    int       `json:"jumlah"`
	Status    string    `json:"status"` // pending, approved, rejected
	Tanggal   time.Time `json:"tanggal"`
}
