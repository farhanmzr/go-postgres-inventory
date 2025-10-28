package models

import "time"

// models/purchase_invoice.go
type PurchaseInvoice struct {

    PurchaseRequestID uint           `gorm:"primaryKey" json:"id"` // expose sebagai "id" di JSON
	InvoiceNo         string        `gorm:"index:idx_purchase_invoices_invoice_no,unique;not null" json:"invoice_no"`
	BuyerName         string        `gorm:"not null" json:"buyer_name"`
	Payment           PaymentMethod `gorm:"type:text;not null" json:"payment"`
	InvoiceDate       time.Time     `gorm:"not null" json:"invoice_date"`
	Subtotal          int64         `gorm:"not null" json:"subtotal"`
	Discount          int64         `gorm:"not null;default:0" json:"discount"`
	Tax               int64         `gorm:"not null;default:0" json:"tax"`
	GrandTotal        int64         `gorm:"not null" json:"grand_total"`

	// Penting: mapping foreignKey/references karena PK parent = PurchaseRequestID (bukan kolom "id")
    Items     []PurchaseInvoiceItem `gorm:"foreignKey:PurchaseInvoiceID;references:PurchaseRequestID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"`
	
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
