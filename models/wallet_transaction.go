package models

import "time"

type WalletTxType string

const (
	WalletTxPurchasePaid   WalletTxType = "PURCHASE_PAID"    // beli cash/bank -> OUT
	WalletTxSalesPaid      WalletTxType = "SALES_PAID"       // jual cash/bank -> IN
	WalletTxHutangPay      WalletTxType = "HUTANG_PAY"       // bayar hutang -> OUT
	WalletTxPiutangReceive WalletTxType = "PIUTANG_RECEIVE"  // terima piutang -> IN
	WalletTxAdjust         WalletTxType = "ADJUST"           // koreksi manual
)

type WalletTransaction struct {
	ID       uint       `gorm:"primaryKey" json:"id"`
	WalletID uint       `gorm:"index;not null" json:"wallet_id"`
	GudangID uint       `gorm:"index;not null" json:"gudang_id"`

	Type      WalletTxType `gorm:"type:text;not null" json:"type"`
	Direction string       `gorm:"size:3;not null" json:"direction"` // IN/OUT
	Amount    int64        `gorm:"not null" json:"amount"`

	RefType string `gorm:"size:40;not null" json:"ref_type"` // "purchase_request", "sales_request", "hutang_payment", ...
	RefID   uint   `gorm:"not null" json:"ref_id"`

	ActorID uint   `gorm:"index;not null" json:"actor_id"`
	Note    string `gorm:"size:255" json:"note,omitempty"`
	
	TxDate time.Time `gorm:"not null" json:"tx_date"`


	CreatedAt time.Time `json:"created_at"`
}
