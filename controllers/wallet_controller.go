package controllers

import (
	"errors"
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CreateCashWalletInput struct {
	Name string `json:"name" binding:"required"`
}

func CreateCashWallet(c *gin.Context) {
	// auth (admin/user permitted sesuai route kamu)
	_, err := currentUserID(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		return
	}

	gid64, _ := strconv.ParseUint(c.Param("gudang_id"), 10, 64)
	gudangID := uint(gid64)

	var in CreateCashWalletInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		c.JSON(400, gin.H{"message": "payload tidak valid"})
		return
	}

	w := models.WarehouseWallet{
		GudangID: gudangID,
		Type:     models.WalletCash,
		Name:     strings.TrimSpace(in.Name),
		Balance:  0,
		IsActive: true,
	}

	if err := config.DB.Create(&w).Error; err != nil {
		c.JSON(500, gin.H{"message": "gagal buat laci", "error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"data": w})
}

// CREATE BANK WALLET
type CreateBankWalletInput struct {
	Name        string `json:"name" binding:"required"`         // label
	AccountName string `json:"account_name" binding:"required"` // pemilik
	AccountNo   string `json:"account_no" binding:"required"`
	BankName    string `json:"bank_name" binding:"required"`
}

func CreateBankWallet(c *gin.Context) {
	_, err := currentUserID(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		return
	}

	gid64, _ := strconv.ParseUint(c.Param("gudang_id"), 10, 64)
	gudangID := uint(gid64)

	var in CreateBankWalletInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"message": "payload tidak valid", "error": err.Error()})
		return
	}

	w := models.WarehouseWallet{
		GudangID:    gudangID,
		Type:        models.WalletBank,
		Name:        strings.TrimSpace(in.Name),
		AccountName: strings.TrimSpace(in.AccountName),
		AccountNo:   strings.TrimSpace(in.AccountNo),
		BankName:    strings.TrimSpace(in.BankName),
		Balance:     0,
		IsActive:    true,
	}

	if err := config.DB.Create(&w).Error; err != nil {
		c.JSON(500, gin.H{"message": "gagal buat bank", "error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"data": w})
}

func ListWalletsByGudang(c *gin.Context) {
	_, err := currentUserID(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		return
	}

	gid64, _ := strconv.ParseUint(c.Param("gudang_id"), 10, 64)
	gudangID := uint(gid64)

	var rows []models.WarehouseWallet
	q := config.DB.Where("gudang_id = ? AND is_active = true", gudangID).Order("type ASC, id ASC")
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"message": "gagal ambil wallet", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": rows})
}

func ListWalletTransactions(c *gin.Context) {
	_, err := currentUserID(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		return
	}

	wid64, _ := strconv.ParseUint(c.Param("wallet_id"), 10, 64)
	walletID := uint(wid64)

	var rows []models.WalletTransaction
	if err := config.DB.
		Where("wallet_id = ?", walletID).
		Order("id DESC").
		Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"message": "gagal ambil mutasi", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": rows})
}

type WalletAdjustInput struct {
	Amount int64     `json:"amount" binding:"required"`
	Date   time.Time `json:"date" binding:"required"`
	Note   string    `json:"note"`
}

func WalletManualIncome(c *gin.Context) {
	actorID, err := currentUserID(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		return
	}

	wid64, _ := strconv.ParseUint(c.Param("wallet_id"), 10, 64)
	walletID := uint(wid64)

	var in WalletAdjustInput
	if err := c.ShouldBindJSON(&in); err != nil || in.Amount <= 0 {
		c.JSON(400, gin.H{"message": "payload tidak valid"})
		return
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		// ambil wallet untuk dapat gudang_id
		var w models.WarehouseWallet
		if err := tx.Clauses(clauseUpdateLock()).First(&w, walletID).Error; err != nil {
			return err
		}

		note := in.Note
		if note == "" {
			note = "Manual income"
		}

		// pakai helper: delta positif
		if err := applyWalletDelta(
			tx, walletID, w.GudangID, +in.Amount,
			models.WalletTxAdjust,
			"manual_income",
			walletID, // ref id bebas; bisa pakai walletID
			actorID,
			note,
			in.Date,
		); err != nil {
			return err
		}

		// kalau kamu mau pakai tanggal custom, perlu update created_at log (opsional).
		// paling gampang: abaikan Date dan gunakan server time.

		return nil
	})

	if err != nil {
		c.JSON(400, gin.H{"message": "gagal manual income", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "manual income berhasil"})
}

func WalletManualExpense(c *gin.Context) {
	actorID, err := currentUserID(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		return
	}

	wid64, _ := strconv.ParseUint(c.Param("wallet_id"), 10, 64)
	walletID := uint(wid64)

	var in WalletAdjustInput
	if err := c.ShouldBindJSON(&in); err != nil || in.Amount <= 0 {
		c.JSON(400, gin.H{"message": "payload tidak valid"})
		return
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var w models.WarehouseWallet
		if err := tx.Clauses(clauseUpdateLock()).First(&w, walletID).Error; err != nil {
			return err
		}

		note := in.Note
		if note == "" {
			note = "Manual expense"
		}

		if err := applyWalletDelta(
			tx, walletID, w.GudangID, -in.Amount,
			models.WalletTxAdjust,
			"manual_expense",
			walletID,
			actorID,
			note,
			in.Date,
		); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		c.JSON(400, gin.H{"message": "gagal manual expense", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "manual expense berhasil"})
}

func walletHasTransactions(tx *gorm.DB, walletID uint) (bool, error) {
	var total int64
	if err := tx.Model(&models.WalletTransaction{}).
		Where("wallet_id = ?", walletID).
		Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func DeleteWallet(c *gin.Context) {
	_, err := currentUserID(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		return
	}

	gid64, err := strconv.ParseUint(c.Param("gudang_id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"message": "gudang_id tidak valid"})
		return
	}
	gudangID := uint(gid64)

	wid64, err := strconv.ParseUint(c.Param("wallet_id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"message": "wallet_id tidak valid"})
		return
	}
	walletID := uint(wid64)

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var wallet models.WarehouseWallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND gudang_id = ?", walletID, gudangID).
			First(&wallet).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("wallet tidak ditemukan")
			}
			return err
		}

		hasTx, err := walletHasTransactions(tx, wallet.ID)
		if err != nil {
			return err
		}
		if hasTx {
			return errors.New("sudah ada transaksi, silakan hapus transaksi terlebih dahulu")
		}

		if wallet.Balance != 0 {
			return errors.New("wallet masih memiliki saldo, kosongkan saldo terlebih dahulu")
		}

		return tx.Delete(&wallet).Error
	})

	if err != nil {
		switch err.Error() {
		case "Wallet tidak ditemukan":
			c.JSON(404, gin.H{"message": err.Error()})
			return
		case "Sudah ada transaksi, silakan hapus transaksi terlebih dahulu":
			c.JSON(400, gin.H{"message": err.Error()})
			return
		case "Wallet masih memiliki saldo, kosongkan saldo terlebih dahulu":
			c.JSON(400, gin.H{"message": err.Error()})
			return
		default:
			c.JSON(500, gin.H{"message": "Gagal menghapus wallet", "error": err.Error()})
			return
		}
	}

	c.JSON(200, gin.H{"message": "Wallet berhasil dihapus"})
}

func DeleteWalletTransaction(c *gin.Context) {
	_, err := currentUserID(c)
	if err != nil {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		return
	}

	wid64, err := strconv.ParseUint(c.Param("wallet_id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"message": "wallet_id tidak valid"})
		return
	}
	walletID := uint(wid64)

	txid64, err := strconv.ParseUint(c.Param("transaction_id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"message": "transaction_id tidak valid"})
		return
	}
	transactionID := uint(txid64)

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var wallet models.WarehouseWallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", walletID).
			First(&wallet).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("wallet tidak ditemukan")
			}
			return err
		}

		var wt models.WalletTransaction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND wallet_id = ?", transactionID, walletID).
			First(&wt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("transaksi wallet tidak ditemukan")
			}
			return err
		}

		if wt.Type != models.WalletTxAdjust {
			return errors.New("hanya transaksi manual yang bisa dihapus dari mutasi wallet")
		}

		if wt.RefType != "manual_income" && wt.RefType != "manual_expense" {
			return errors.New("transaksi ini bukan manual income/expense")
		}

		switch wt.Direction {
		case "IN":
			if wallet.Balance < wt.Amount {
				return errors.New("saldo wallet tidak cukup untuk menghapus transaksi ini")
			}
			wallet.Balance -= wt.Amount
		case "OUT":
			wallet.Balance += wt.Amount
		default:
			return errors.New("direction transaksi tidak valid")
		}

		if err := tx.Model(&models.WarehouseWallet{}).
			Where("id = ?", wallet.ID).
			Update("balance", wallet.Balance).Error; err != nil {
			return err
		}

		if err := tx.Delete(&wt).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		switch err.Error() {
		case "wallet tidak ditemukan":
			c.JSON(404, gin.H{"message": err.Error()})
			return
		case "transaksi wallet tidak ditemukan":
			c.JSON(404, gin.H{"message": err.Error()})
			return
		case "hanya transaksi manual yang bisa dihapus dari mutasi wallet":
			c.JSON(400, gin.H{"message": err.Error()})
			return
		case "transaksi ini bukan manual income/expense":
			c.JSON(400, gin.H{"message": err.Error()})
			return
		case "saldo wallet tidak cukup untuk menghapus transaksi ini":
			c.JSON(400, gin.H{"message": err.Error()})
			return
		case "direction transaksi tidak valid":
			c.JSON(400, gin.H{"message": err.Error()})
			return
		default:
			c.JSON(500, gin.H{"message": "gagal menghapus transaksi wallet", "error": err.Error()})
			return
		}
	}

	c.JSON(200, gin.H{"message": "transaksi wallet berhasil dihapus"})
}
