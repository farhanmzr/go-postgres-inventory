// models/hutang.go
package models

import "time"

type Hutang struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	UserID   uint   `gorm:"index;not null" json:"user_id"` // user pembuat pembelian / pemilik data
	UserName string `gorm:"size:180;not null" json:"user_name"`

	SupplierID   uint   `gorm:"index;not null" json:"supplier_id"`
	SupplierName string `gorm:"size:180;not null" json:"supplier_name"`

	PurchaseRequestID uint      `gorm:"not null;index" json:"purchase_request_id"`
	WarehouseID uint `gorm:"-" json:"warehouse_id"`
	InvoiceNo         string    `gorm:"size:64;not null;index" json:"invoice_no"`
	InvoiceDate       time.Time `gorm:"not null" json:"invoice_date"`
	DueDate           time.Time `gorm:"not null" json:"due_date"`

	Total     int64 `gorm:"not null" json:"total"`
	TotalPaid int64 `gorm:"not null;default:0" json:"total_paid"` // total diterima
	IsPaid    bool  `gorm:"not null;default:false" json:"is_paid"`

	Items []HutangItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HutangItem struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	HutangID uint `gorm:"index;not null" json:"hutang_id"`

	BarangID  uint   `gorm:"not null" json:"barang_id"`
	Nama      string `gorm:"size:200;not null" json:"nama"`
	Kode      string `gorm:"size:100;not null" json:"kode"`
	Qty       int64  `gorm:"not null" json:"qty"`
	Price     int64  `gorm:"not null" json:"price"` // harga beli
	LineTotal int64  `gorm:"not null" json:"line_total"`
}
