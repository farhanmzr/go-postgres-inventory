package controllers

import (
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
		GudangID:     gudangID,
		Type:         models.WalletBank,
		Name:         strings.TrimSpace(in.Name),
		AccountName:  strings.TrimSpace(in.AccountName),
		AccountNo:    strings.TrimSpace(in.AccountNo),
		BankName:     strings.TrimSpace(in.BankName),
		Balance:      0,
		IsActive:     true,
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
    Date   time.Time `json:"date"` // opsional
    Note   string    `json:"note"`
}

func WalletManualIncome(c *gin.Context) {
    actorID, err := currentUserID(c)
    if err != nil {
        c.JSON(401, gin.H{"message":"Unauthorized"})
        return
    }

    wid64, _ := strconv.ParseUint(c.Param("wallet_id"), 10, 64)
    walletID := uint(wid64)

    var in WalletAdjustInput
    if err := c.ShouldBindJSON(&in); err != nil || in.Amount <= 0 {
        c.JSON(400, gin.H{"message":"payload tidak valid"})
        return
    }

    err = config.DB.Transaction(func(tx *gorm.DB) error {
        // ambil wallet untuk dapat gudang_id
        var w models.WarehouseWallet
        if err := tx.Clauses(clauseUpdateLock()).First(&w, walletID).Error; err != nil {
            return err
        }

        note := in.Note
        if note == "" { note = "Manual income" }

        // pakai helper: delta positif
        if err := applyWalletDelta(
            tx, walletID, w.GudangID, +in.Amount,
            models.WalletTxAdjust,
            "manual_income",
            walletID, // ref id bebas; bisa pakai walletID
            actorID,
            note,
        ); err != nil {
            return err
        }

        // kalau kamu mau pakai tanggal custom, perlu update created_at log (opsional).
        // paling gampang: abaikan Date dan gunakan server time.

        return nil
    })

    if err != nil {
        c.JSON(400, gin.H{"message":"gagal manual income", "error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"message":"manual income berhasil"})
}

func WalletManualExpense(c *gin.Context) {
    actorID, err := currentUserID(c)
    if err != nil {
        c.JSON(401, gin.H{"message":"Unauthorized"})
        return
    }

    wid64, _ := strconv.ParseUint(c.Param("wallet_id"), 10, 64)
    walletID := uint(wid64)

    var in WalletAdjustInput
    if err := c.ShouldBindJSON(&in); err != nil || in.Amount <= 0 {
        c.JSON(400, gin.H{"message":"payload tidak valid"})
        return
    }

    err = config.DB.Transaction(func(tx *gorm.DB) error {
        var w models.WarehouseWallet
        if err := tx.Clauses(clauseUpdateLock()).First(&w, walletID).Error; err != nil {
            return err
        }

        note := in.Note
        if note == "" { note = "Manual expense" }

        if err := applyWalletDelta(
            tx, walletID, w.GudangID, -in.Amount,
            models.WalletTxAdjust,
            "manual_expense",
            walletID,
            actorID,
            note,
        ); err != nil {
            return err
        }
        return nil
    })

    if err != nil {
        c.JSON(400, gin.H{"message":"gagal manual expense", "error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"message":"manual expense berhasil"})
}





