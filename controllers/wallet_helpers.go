package controllers

import (
	"errors"
	"fmt"

	"go-postgres-inventory/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func clauseUpdateLock() clause.Locking {
	return clause.Locking{Strength: "UPDATE"}
}

// delta: + untuk IN, - untuk OUT
func applyWalletDelta(
	tx *gorm.DB,
	walletID uint,
	gudangID uint,
	delta int64,
	txType models.WalletTxType,
	refType string,
	refID uint,
	actorID uint,
	note string,
) error {
	if delta == 0 {
		return nil
	}

	// lock wallet row
	var w models.WarehouseWallet
	if err := tx.Clauses(clauseUpdateLock()).
		First(&w, walletID).Error; err != nil {
		return err
	}

	if w.GudangID != gudangID {
		return errors.New("wallet tidak milik gudang ini")
	}
	if !w.IsActive {
		return errors.New("wallet tidak aktif")
	}

	// guard saldo tidak negatif
	newBal := w.Balance + delta
	if newBal < 0 {
		return fmt.Errorf("Saldo wallet tidak cukup (saldo=%d, butuh=%d)", w.Balance, -delta)
	}

	// update saldo
	if err := tx.Model(&models.WarehouseWallet{}).
		Where("id = ?", w.ID).
		Update("balance", newBal).Error; err != nil {
		return err
	}

	// insert mutasi
	dir := "IN"
	amt := delta
	if delta < 0 {
		dir = "OUT"
		amt = -delta
	}

	log := models.WalletTransaction{
		WalletID:  w.ID,
		GudangID:  gudangID,
		Type:      txType,
		Direction: dir,
		Amount:    amt,
		RefType:   refType,
		RefID:     refID,
		ActorID:   actorID,
		Note:      note,
	}
	return tx.Create(&log).Error
}
