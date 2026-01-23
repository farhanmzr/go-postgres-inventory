// models/hutang_payment.go
package models

import "time"

type HutangPayment struct {
    ID       uint      `gorm:"primaryKey" json:"id"`
    HutangID uint      `gorm:"index;not null" json:"hutang_id"`

    Amount   int64     `gorm:"not null" json:"amount"`
    WalletID      uint   `gorm:"index;not null" json:"wallet_id"`
    PaymentMethod string `gorm:"size:20;not null" json:"payment_method"` // CASH / BANK / ...
    PaidAt   time.Time `gorm:"not null" json:"paid_at"`

    PaidByID uint      `gorm:"index;not null" json:"paid_by_id"` // actor user id
    Note     string    `gorm:"size:255" json:"note,omitempty"`

    CreatedAt time.Time `json:"created_at"`
}
