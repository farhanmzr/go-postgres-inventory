package models

import "time"

// Invoice penjualan (header)
type SalesInvoice struct {
	SalesRequestID uint               `gorm:"primaryKey" json:"id"` // expose sebagai "id" di JSON
	InvoiceNo         string        `gorm:"index:idx_sales_invoices_invoice_no,unique;not null" json:"invoice_no"`
	Username       string             `gorm:"not null" json:"username"`
	Payment        PaymentMethod      `gorm:"type:text;not null" json:"payment"`
	InvoiceDate    time.Time          `gorm:"not null" json:"invoice_date"`
	Subtotal       int64              `gorm:"not null" json:"subtotal"`
	Discount       int64              `gorm:"not null;default:0" json:"discount"`
	Tax            int64              `gorm:"not null;default:0" json:"tax"`
	GrandTotal     int64              `gorm:"not null" json:"grand_total"`

	Items          []SalesInvoiceItem `gorm:"foreignKey:SalesInvoiceID;references:SalesRequestID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

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
	// ⬇️ kolom baru untuk profit
    CostPrice       int64  `gorm:"not null"` // snapshot harga beli/unit saat penjualan dibuat
    ProfitPerUnit   int64  `gorm:"not null"` // = Price - CostPrice
    ProfitTotal     int64  `gorm:"not null"` // = ProfitPerUnit * Qty
	
	LineTotal      int64     `gorm:"not null" json:"line_total"` // qty * price
	Barang         *Barang   `json:"barang,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
