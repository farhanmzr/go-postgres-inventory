// models/purchase_request.go
package models

import "time"

type PaymentMethod string
const (
	PaymentCash   PaymentMethod = "CASH"
	PaymentCredit PaymentMethod = "CREDIT"
)

type PurchaseRequest struct {
	ID              uint            `gorm:"primaryKey" json:"id"`
	TransCode       string          `gorm:"uniqueIndex;size:40" json:"trans_code"` // e.g. TR-2025-000123 (generate di server)
	ManualCode      *string         `gorm:"size:40" json:"manual_code"`            // opsional, admin isi
	BuyerName       string          `gorm:"size:180;not null" json:"buyer_name"`
	PurchaseDate    time.Time       `json:"purchase_date"` // tanggal (<= today)
	WarehouseID     uint            `json:"warehouse_id"`
	Warehouse       Gudang          `json:"warehouse"`
	SupplierID      uint            `json:"supplier_id"`
	Supplier        Supplier        `json:"supplier"`
	Payment         PaymentMethod   `gorm:"size:10" json:"payment"`

	Items           []PurchaseReqItem `json:"items"`

	CreatedByID     uint            `json:"created_by_id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type PurchaseReqItem struct {
	ID                uint    `gorm:"primaryKey" json:"id"`
	PurchaseRequestID uint    `gorm:"index" json:"purchase_request_id"`
	BarangID          uint    `json:"barang_id"`
	Barang            Barang  `json:"barang"`
	Qty               int64   `json:"qty"`
	BuyPrice          int64   `json:"buy_price"` // harga beli saat request
	LineTotal         int64   `json:"line_total"`
}
