// models/purchase_request.go
package models

import "time"

type SalesStatus string

const (
	StatusPending  SalesStatus = "PENDING"
	StatusApproved SalesStatus = "APPROVED"
	StatusRejected SalesStatus = "REJECTED"
)

type SalesRequest struct {
	ID           uint          `gorm:"primaryKey" json:"id"`
	TransCode    string        `gorm:"uniqueIndex;size:40" json:"trans_code"` // e.g. TR-2025-000123 (generate di server)
	ManualCode   *string       `gorm:"size:40" json:"manual_code"`            // opsional, admin isi
	Username     string        `gorm:"size:180;not null" json:"username"`
	SalesDate time.Time     `json:"sales_date"` // tanggal (<= today)
	WarehouseID  uint          `json:"warehouse_id"`
	Warehouse    Gudang        `json:"warehouse"`
	CustomerID   uint          `json:"customer_id"`
	Customer     Customer      `json:"customer"`
	Payment      PaymentMethod `gorm:"size:10" json:"payment"`

	Status       SalesStatus `gorm:"size:12;index" json:"status"`
	RejectReason *string     `gorm:"size:255" json:"reject_reason"`

	Items []SalesReqItem `json:"items"`

	CreatedByID uint      `json:"created_by_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SalesReqItem struct {
	ID                uint   `gorm:"primaryKey" json:"id"`
	PurchaseRequestID uint   `gorm:"index" json:"purchase_request_id"`
	BarangID          uint   `json:"barang_id"`
	Barang            Barang `json:"barang"`
	Qty               int64  `json:"qty"`
	SellPrice         int64  `json:"sell_price"` // harga jual saat request
	LineTotal         int64  `json:"line_total"`
}
