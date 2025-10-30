// models/usage.go

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
	UsageDate     time.Time   `gorm:"not null" json_json:"usage_date"`
	RequesterName string      `gorm:"not null" json:"requester_name"`
	PenggunaName  string      `gorm:"not null" json:"pengguna_name"`

	WarehouseID uint    `json:"warehouse_id"`
	Warehouse   Gudang  `gorm:"foreignKey:WarehouseID;references:ID" json:"warehouse"`
	CustomerID  uint    `json:"customer_id"`
	Customer    Customer `gorm:"foreignKey:CustomerID;references:ID" json:"customer"`

	CreatedByID uint        `gorm:"not null" json:"created_by_id"`
	Status      UsageStatus `gorm:"type:text;not null;default:BELUM_DIPROSES" json:"status"`

	// penting: definisikan relasi & constraint ke items
	Items []UsageItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UsageItem struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	UsageRequestID uint            `gorm:"index;not null" json:"usage_request_id"`
	// opsional: backlink ke header (tidak perlu di-JSON)
	UsageRequest *UsageRequest `gorm:"foreignKey:UsageRequestID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	BarangID uint    `gorm:"not null" json:"barang_id"`
	// ⬇⬇⬇ Tambahkan field relasi yang diminta Preload("Items.Barang")
	Barang   *Barang `gorm:"foreignKey:BarangID;references:ID" json:"barang,omitempty"`

	CustomerID uint       `gorm:"not null" json:"customer_id"`
	// kalau mau bisa preload Items.Customer juga:
	Customer  *Customer   `gorm:"foreignKey:CustomerID;references:ID" json:"customer,omitempty"`

	Qty          int64           `gorm:"not null" json:"qty"`
	ItemStatus   UsageItemStatus `gorm:"type:text;not null;default:PENDING" json:"item_status"`
	StockApplied bool            `gorm:"not null;default:false" json:"stock_applied"`
	Note         *string         `json:"note"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
