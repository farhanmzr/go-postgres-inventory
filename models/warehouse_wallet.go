package models

import "time"

type WalletType string

const (
	WalletCash WalletType = "CASH" // laci
	WalletBank WalletType = "BANK" // rekening
)

type WarehouseWallet struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	GudangID uint      `gorm:"index;not null" json:"gudang_id"`
	Type     WalletType `gorm:"type:text;not null" json:"type"` // CASH/BANK

	// label
	Name    string `gorm:"size:120;not null" json:"name"`
	Balance int64  `gorm:"not null;default:0" json:"balance"`

	// khusus bank (opsional untuk CASH)
	AccountName string `gorm:"size:120" json:"account_name,omitempty"`
	AccountNo   string `gorm:"size:64" json:"account_no,omitempty"`
	BankName    string `gorm:"size:80" json:"bank_name,omitempty"`

	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
