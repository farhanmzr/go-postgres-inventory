package models

import "time"

type UsageStatus string

const (
	UsageBelumDiproses UsageStatus = "BELUM_DIPROSES"
	UsageSudahDiproses UsageStatus = "SUDAH_DIPROSES"
)

type UsageItemStatus string

const (
	ItemPending  UsageItemStatus = "PENDING"
	ItemApproved UsageItemStatus = "APPROVED"
	ItemRejected UsageItemStatus = "REJECTED"
)

type UsageRequest struct {
	ID            uint        `gorm:"primaryKey" json:"id"`
	TransCode     string      `gorm:"uniqueIndex;not null" json:"trans_code"`
	ManualCode    *string     `json:"manual_code"`
	UsageDate     time.Time   `gorm:"not null" json:"usage_date"`
	RequesterName string      `gorm:"not null" json:"requester_name"`
	CreatedByID   uint        `gorm:"not null" json:"created_by_id"`
	Status        UsageStatus `gorm:"type:usage_status;default:BELUM_DIPROSES" json:"status"`

	Items []UsageItem `json:"items"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UsageItem struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	UsageRequestID uint            `gorm:"index;not null" json:"usage_request_id"`
	BarangID       uint            `gorm:"not null" json:"barang_id"`
	CustomerID     uint            `gorm:"not null" json:"customer_id"`
	Qty            int64           `gorm:"not null" json:"qty"`
	ItemStatus     UsageItemStatus `gorm:"type:usage_item_status;default:PENDING" json:"item_status"`
	StockApplied   bool            `gorm:"not null;default:false" json:"stock_applied"`
	Note           *string         `json:"note"`

	Barang   *Barang   `json:"barang,omitempty"`
	Customer *Customer `json:"customer,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
