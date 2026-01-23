// models/piutang.go
package models

import "time"

// Header piutang (1 piutang per invoice)
type Piutang struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	UserID   uint   `gorm:"index;not null" json:"user_id"`      // pemilik piutang (created_by_id)
	UserName string `gorm:"size:180;not null" json:"user_name"` // display

	SalesRequestID uint `gorm:"not null;index" json:"sales_request_id"`

	InvoiceNo   string    `gorm:"size:64;not null;index" json:"invoice_no"`
	InvoiceDate time.Time `gorm:"not null" json:"invoice_date"`
	DueDate     time.Time `gorm:"not null" json:"due_date"`

	Total     int64 `gorm:"not null" json:"total"`
	TotalPaid int64 `gorm:"not null;default:0" json:"total_paid"` // total diterima
	IsPaid    bool  `gorm:"not null;default:false" json:"is_paid"`

	Items []PiutangItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Detail barang pada piutang (snapshot dari invoice items)
type PiutangItem struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	PiutangID uint   `gorm:"index;not null" json:"piutang_id"`
	BarangID  uint   `gorm:"not null" json:"barang_id"`
	Nama      string `gorm:"size:200;not null" json:"nama"`
	Kode      string `gorm:"size:100;not null" json:"kode"`

	Qty       int64 `gorm:"not null" json:"qty"`
	Price     int64 `gorm:"not null" json:"price"` // harga per unit (beli/jual sesuai source)
	LineTotal int64 `gorm:"not null" json:"line_total"`
}
