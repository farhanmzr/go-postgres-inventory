package controllers

import (
	"errors"
	"fmt"
	"time"

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
	txDate time.Time,
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
		TxDate:    txDate,
	}
	return tx.Create(&log).Error
}

func refundAllHutangPayments(tx *gorm.DB, hutangID uint, gudangID uint, actorID uint) error {
    var pays []models.HutangPayment
    if err := tx.Where("hutang_id = ?", hutangID).Order("id ASC").Find(&pays).Error; err != nil {
        return err
    }

    for _, hp := range pays {
        if hp.Amount <= 0 { continue }

        // refund: uang balik ke wallet (IN)
        if err := applyWalletDelta(
            tx,
            hp.WalletID,
            gudangID,
            +hp.Amount,
            models.WalletTxHutangRefund,
            "hutang_refund",
            hutangID,
            actorID,
            "Refund cicilan hutang (hapus transaksi)",
            time.Now().UTC(),
        ); err != nil {
            return err
        }
    }

    // hapus history pembayaran
    return tx.Where("hutang_id = ?", hutangID).Delete(&models.HutangPayment{}).Error
}


func refundAllPiutangReceipts(tx *gorm.DB, piutangID uint, gudangID uint, actorID uint) error {
    var rows []models.PiutangReceipt
    if err := tx.Where("piutang_id = ?", piutangID).Order("id ASC").Find(&rows).Error; err != nil {
        return err
    }

    for _, rc := range rows {
        if rc.Amount <= 0 { continue }

        // reverse receipt: uang yang dulu masuk harus keluar (OUT)
        if err := applyWalletDelta(
            tx,
            rc.WalletID,
            gudangID,
            -rc.Amount,
            models.WalletTxPiutangRefund,
            "piutang_refund",
            piutangID,
            actorID,
            "Reverse receipt piutang (hapus transaksi)",
            time.Now().UTC(),
        ); err != nil {
            return err
        }
    }

    // hapus history receipt
    return tx.Where("piutang_id = ?", piutangID).Delete(&models.PiutangReceipt{}).Error
}


