// models/piutang_receipt.go
package models

import "time"

type PiutangReceipt struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	PiutangID uint `gorm:"index;not null" json:"piutang_id"`

	Amount        int64     `gorm:"not null" json:"amount"`
	WalletID      uint      `gorm:"index;not null" json:"wallet_id"`
	PaymentMethod string    `gorm:"size:20;not null" json:"payment_method"` // CASH / BANK / ...
	ReceivedAt    time.Time `gorm:"not null" json:"received_at"`

	ReceivedByID uint   `gorm:"index;not null" json:"received_by_id"`
	Note         string `gorm:"size:255" json:"note,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}
