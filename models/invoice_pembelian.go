package models

import "time"

// models/purchase_invoice.go
type PurchaseInvoice struct {
	ID                uint          `gorm:"primaryKey" json:"id"`
	InvoiceNo         string        `gorm:"index:idx_purchase_invoices_invoice_no,unique;not null" json:"invoice_no"`
	PurchaseRequestID uint          `gorm:"not null" json:"purchase_request_id"`
	BuyerName         string        `gorm:"not null" json:"buyer_name"`
	Payment           PaymentMethod `gorm:"type:text;not null" json:"payment"`
	InvoiceDate       time.Time     `gorm:"not null" json:"invoice_date"`
	Subtotal          int64         `gorm:"not null" json:"subtotal"`
	Discount          int64         `gorm:"not null;default:0" json:"discount"`
	Tax               int64         `gorm:"not null;default:0" json:"tax"`
	GrandTotal        int64         `gorm:"not null" json:"grand_total"`

	// biar GORM tau relasi otomatis, cukup begini ketika child punya PurchaseInvoiceID
	Items     []PurchaseInvoiceItem `json:"items" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

type PurchaseInvoiceItem struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// ⬇️ ganti ke PurchaseInvoiceID (konvensi GORM)
	PurchaseInvoiceID uint    `gorm:"index;not null" json:"invoice_id"`
	BarangID          uint    `gorm:"not null" json:"barang_id"`
	Qty               int64   `gorm:"not null" json:"qty"`
	Price             int64   `gorm:"not null" json:"price"`
	LineTotal         int64   `gorm:"not null" json:"line_total"`
	Barang            *Barang `json:"barang,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 🔽 Tambahkan ini di file yang sama:
func (PurchaseInvoice) TableName() string     { return "purchase_invoices" }
func (PurchaseInvoiceItem) TableName() string { return "purchase_invoice_items" }
