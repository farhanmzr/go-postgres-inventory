// controllers/piutang_controller.go
package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ===== ADMIN: list semua hutang (opsional: ?status=UNPAID/PAY_REQUESTED/PAID)
func HutangListAdmin(c *gin.Context) {
	var rows []models.Hutang
	q := config.DB.Preload("Items").Order("due_date ASC, id DESC")

	if paid := c.Query("is_paid"); paid != "" {
		q = q.Where("is_paid = ?", paid)
	}

	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil hutang", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

// GET /hutang/admin/:id/history
func HutangPaymentHistoryAdmin(c *gin.Context) {
    if _, err := currentAdminID(c); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized", "error": err.Error()})
        return
    }

    id, _ := strconv.Atoi(c.Param("id"))

    // cek hutang exist
    var h models.Hutang
    if err := config.DB.Select("id").First(&h, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"message": "Hutang tidak ditemukan"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil hutang", "error": err.Error()})
        return
    }

    var rows []models.HutangPayment
    if err := config.DB.
        Where("hutang_id = ?", h.ID).
        Order("paid_at ASC, id ASC").
        Find(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil history", "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": rows})
}


// ===== USER: list hutang miliknya
func HutangListUser(c *gin.Context) {
	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	var rows []models.Hutang
	q := config.DB.Preload("Items").Where("user_id = ?", uid).Order("due_date ASC, id DESC")

	if paid := c.Query("is_paid"); paid != "" {
		q = q.Where("is_paid = ?", paid)
	}

	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil hutang", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

type HutangPayInput struct {
    Amount        int64  `json:"amount" binding:"required"`
    WalletID      uint   `json:"wallet_id" binding:"required"`
    PaymentMethod string `json:"payment_method" binding:"required"` // CASH/BANK
    Note          string `json:"note"`
}

func HutangPay(c *gin.Context) {
    uid, err := currentUserID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
        return
    }

    id, _ := strconv.Atoi(c.Param("id"))

    var in HutangPayInput
    if err := c.ShouldBindJSON(&in); err != nil || in.Amount <= 0 || in.WalletID == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"message": "payload tidak valid"})
        return
    }

    // validasi payment_method sederhana (opsional)
    if in.PaymentMethod != "CASH" && in.PaymentMethod != "BANK" {
        c.JSON(http.StatusBadRequest, gin.H{"message": "payment_method tidak valid (CASH/BANK)"})
        return
    }

    err = config.DB.Transaction(func(tx *gorm.DB) error {
        // 1) lock hutang
        var h models.Hutang
        if err := tx.Clauses(clauseUpdateLock()).First(&h, id).Error; err != nil {
            return err
        }
        if h.UserID != uid {
            return errors.New("forbidden")
        }
        if h.IsPaid {
            return errors.New("hutang sudah lunas")
        }

        remaining := h.Total - h.TotalPaid
        if remaining <= 0 {
            // safety mark
            return tx.Model(&models.Hutang{}).
                Where("id = ?", h.ID).
                Update("is_paid", true).Error
        }

        pay := in.Amount
        if pay > remaining {
            pay = remaining
        }

        // 2) ambil warehouse_id dari PurchaseRequest (untuk validasi wallet gudang)
        var pr models.PurchaseRequest
        if err := tx.Select("id", "warehouse_id").First(&pr, h.PurchaseRequestID).Error; err != nil {
            return err
        }

        // 3) lock wallet + cek gudang cocok + saldo cukup
        var w models.WarehouseWallet
        if err := tx.Clauses(clauseUpdateLock()).First(&w, in.WalletID).Error; err != nil {
            return err
        }
        if w.GudangID != pr.WarehouseID {
            return errors.New("wallet bukan milik gudang pembelian ini")
        }
        if !w.IsActive {
            return errors.New("wallet tidak aktif")
        }

        // optional: cocokkan radio button dengan wallet type
        if in.PaymentMethod == "CASH" && w.Type != models.WalletCash {
            return errors.New("payment_method CASH harus pilih wallet type CASH (laci)")
        }
        if in.PaymentMethod == "BANK" && w.Type != models.WalletBank {
            return errors.New("payment_method BANK harus pilih wallet type BANK")
        }

        if w.Balance < pay {
            return errors.New("saldo wallet tidak cukup")
        }

        // 4) update wallet balance (OUT)
        if err := tx.Model(&models.WarehouseWallet{}).
            Where("id = ?", w.ID).
            Update("balance", gorm.Expr("balance - ?", pay)).Error; err != nil {
            return err
        }

        now := time.Now().UTC()

        // 5) insert history pembayaran hutang
        hp := models.HutangPayment{
            HutangID:       h.ID,
            Amount:         pay,
            WalletID:       w.ID,
            PaymentMethod:  in.PaymentMethod,
            PaidAt:         now,
            PaidByID:       uid,
            Note:           in.Note,
        }
        if err := tx.Create(&hp).Error; err != nil {
            return err
        }

        // 6) update agregat hutang
        newPaid := h.TotalPaid + pay
        isPaid := newPaid >= h.Total

        res := tx.Model(&models.Hutang{}).
            Where("id = ? AND is_paid = false", h.ID).
            Updates(map[string]any{
                "total_paid": gorm.Expr("total_paid + ?", pay),
                "is_paid":    isPaid,
            })
        if res.Error != nil {
            return res.Error
        }
        if res.RowsAffected == 0 {
            return errors.New("gagal update pembayaran")
        }

        // (opsional tapi recommended) insert wallet transaction log
        wt := models.WalletTransaction{
            WalletID:  w.ID,
            GudangID:  pr.WarehouseID,
            Type:      models.WalletTxHutangPay,
            Direction: "OUT",
            Amount:    pay,
            RefType:   "hutang",
            RefID:     h.ID,
            ActorID:   uid,
            Note:      in.Note,
            CreatedAt: now,
        }
        if err := tx.Create(&wt).Error; err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        code := http.StatusBadRequest
        if errors.Is(err, gorm.ErrRecordNotFound) {
            code = http.StatusNotFound
        }
        c.JSON(code, gin.H{"message": "Gagal bayar hutang", "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Pembayaran hutang berhasil"})
}

// GET /hutang/:id/history
func HutangPaymentHistory(c *gin.Context) {
    uid, err := currentUserID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
        return
    }

    id, _ := strconv.Atoi(c.Param("id"))

    // Pastikan hutang ada dan milik user
    var h models.Hutang
    if err := config.DB.Select("id", "user_id").First(&h, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"message": "Hutang tidak ditemukan"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil hutang", "error": err.Error()})
        return
    }
    if h.UserID != uid {
        c.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
        return
    }

    // Ambil history pembayaran
    var rows []models.HutangPayment
    if err := config.DB.
        Where("hutang_id = ?", h.ID).
        Order("paid_at ASC, id ASC").
        Find(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil history", "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": rows})
}


