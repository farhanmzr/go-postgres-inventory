package models

import "time"

// Invoice penjualan (header)
type SalesInvoice struct {
	ID             uint               `gorm:"primaryKey" json:"id"`
	InvoiceNo      string             `gorm:"uniqueIndex;not null" json:"invoice_no"`
	SalesRequestID uint               `gorm:"not null" json:"sales_request_id"`
	CustomerName   string             `gorm:"not null" json:"customer_name"`
	Payment        PaymentMethod      `gorm:"type:text;not null" json:"payment"`
	InvoiceDate    time.Time          `gorm:"not null" json:"invoice_date"`
	Subtotal       int64              `gorm:"not null" json:"subtotal"`
	Discount       int64              `gorm:"not null;default:0" json:"discount"`
	Tax            int64              `gorm:"not null;default:0" json:"tax"`
	GrandTotal     int64              `gorm:"not null" json:"grand_total"`
	Items          []SalesInvoiceItem `json:"items" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// Invoice penjualan (item)
type SalesInvoiceItem struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SalesInvoiceID uint      `gorm:"index;not null" json:"invoice_id"`
	BarangID       uint      `gorm:"not null" json:"barang_id"`
	Qty            int64     `gorm:"not null" json:"qty"`
	Price          int64     `gorm:"not null" json:"price"`      // harga jual per unit
	LineTotal      int64     `gorm:"not null" json:"line_total"` // qty * price
	Barang         *Barang   `json:"barang,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
