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
	ID          uint          `gorm:"primaryKey" json:"id"`
	TransCode   string        `gorm:"uniqueIndex:idx_sales_trans_code;size:64" json:"trans_code"`
	TransSeq    uint          `gorm:"index:idx_sales_user_seq,unique" json:"trans_seq"` // <— angka urut
	ManualCode  *string       `gorm:"size:40" json:"manual_code"`
	Username    string        `gorm:"size:180;not null" json:"username"`
	SalesDate   time.Time     `json:"sales_date"`
	WarehouseID uint          `json:"warehouse_id"`
	Warehouse   Gudang        `json:"warehouse"`
	CustomerID  uint          `json:"customer_id"`
	Customer    Customer      `json:"customer"`
	Payment     PaymentMethod `gorm:"size:10" json:"payment"`

	WalletID *uint `gorm:"index" json:"wallet_id,omitempty"`

	Status       SalesStatus `gorm:"size:12;index" json:"status"`
	RejectReason *string     `gorm:"size:255" json:"reject_reason"`

	Items       []SalesReqItem `json:"items"`
	CreatedByID uint           `gorm:"index:idx_sales_user_seq,unique" json:"created_by_id"` // <— ikut composite unique
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type SalesReqItem struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	SalesRequestID uint   `gorm:"index" json:"sales_request_id"`
	BarangID       uint   `json:"barang_id"`
	Barang         Barang `json:"barang"`
	Qty            int64  `json:"qty"`
	SellPrice      int64  `json:"sell_price"` // harga jual saat request
	LineTotal      int64  `json:"line_total"`
}
