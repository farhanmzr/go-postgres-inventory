package models

import "time"

// Invoice pembelian (header)
type PurchaseInvoice struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	InvoiceNo         string         `gorm:"uniqueIndex;not null" json:"invoice_no"` // sama dengan TransCode pembelian
	PurchaseRequestID uint           `gorm:"not null" json:"purchase_request_id"`
	BuyerName         string         `gorm:"not null" json:"buyer_name"`
	Payment           PaymentMethod  `gorm:"type:text;not null" json:"payment"`
	InvoiceDate       time.Time      `gorm:"not null" json:"invoice_date"`

	// ringkasan angka (pakai int64 biar aman, satuan minor currency)
	Subtotal   int64 `gorm:"not null" json:"subtotal"`
	Discount   int64 `gorm:"not null;default:0" json:"discount"`
	Tax        int64 `gorm:"not null;default:0" json:"tax"`
	GrandTotal int64 `gorm:"not null" json:"grand_total"`

	Items     []PurchaseInvoiceItem `json:"items"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// Invoice pembelian (item)
type PurchaseInvoiceItem struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	InvoiceID   uint   `gorm:"index;not null" json:"invoice_id"`
	BarangID    uint   `gorm:"not null" json:"barang_id"`
	Qty         int64  `gorm:"not null" json:"qty"`
	Price       int64  `gorm:"not null" json:"price"`       // harga beli per unit (minor currency)
	LineTotal   int64  `gorm:"not null" json:"line_total"`  // qty * price
	Barang      *Barang `json:"barang,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
