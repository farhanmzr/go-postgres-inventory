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

// ===== ADMIN: list semua piutang (filter ?is_paid=true/false)
func PiutangListAdmin(c *gin.Context) {
    var rows []models.Piutang
    q := config.DB.Preload("Items").Order("due_date ASC, id DESC")

    if paid := c.Query("is_paid"); paid != "" {
        q = q.Where("is_paid = ?", paid)
    }

    if err := q.Find(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil piutang", "error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": rows})
}

// ===== USER: list piutang miliknya (filter ?is_paid=true/false)
func PiutangListUser(c *gin.Context) {
    uid, err := currentUserID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
        return
    }

    var rows []models.Piutang
    q := config.DB.Preload("Items").
        Where("user_id = ?", uid).
        Order("due_date ASC, id DESC")

    if paid := c.Query("is_paid"); paid != "" {
        q = q.Where("is_paid = ?", paid)
    }

    if err := q.Find(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil piutang", "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": rows})
}

type PiutangReceiveInput struct {
    Amount        int64  `json:"amount" binding:"required"`
    WalletID      uint   `json:"wallet_id" binding:"required"`
    PaymentMethod string `json:"payment_method" binding:"required"` // CASH/BANK (radio)
    Note          string `json:"note"`
}

// USER: terima piutang (bisa cicil/parsial) -> saldo wallet IN
func PiutangReceive(c *gin.Context) {
    uid, err := currentUserID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
        return
    }

    id, _ := strconv.Atoi(c.Param("id"))

    var in PiutangReceiveInput
    if err := c.ShouldBindJSON(&in); err != nil || in.Amount <= 0 || in.WalletID == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"message": "payload tidak valid"})
        return
    }
    if in.PaymentMethod != "CASH" && in.PaymentMethod != "BANK" {
        c.JSON(http.StatusBadRequest, gin.H{"message": "payment_method tidak valid (CASH/BANK)"})
        return
    }

    err = config.DB.Transaction(func(tx *gorm.DB) error {
        // 1) lock piutang
        var p models.Piutang
        if err := tx.Clauses(clauseUpdateLock()).First(&p, id).Error; err != nil {
            return err
        }
        if p.UserID != uid {
            return errors.New("forbidden")
        }
        if p.IsPaid {
            return errors.New("piutang sudah lunas")
        }

        remaining := p.Total - p.TotalPaid
        if remaining <= 0 {
            // safety mark
            return tx.Model(&models.Piutang{}).
                Where("id = ?", p.ID).
                Update("is_paid", true).Error
        }

        receive := in.Amount
        if receive > remaining {
            receive = remaining
        }

        // 2) ambil warehouse_id dari SalesRequest (untuk validasi wallet gudang)
        var sr models.SalesRequest
        if err := tx.Select("id", "warehouse_id").First(&sr, p.SalesRequestID).Error; err != nil {
            return err
        }

        // 3) lock wallet + cek gudang cocok + aktif
        var w models.WarehouseWallet
        if err := tx.Clauses(clauseUpdateLock()).First(&w, in.WalletID).Error; err != nil {
            return err
        }
        if w.GudangID != sr.WarehouseID {
            return errors.New("wallet bukan milik gudang penjualan ini")
        }
        if !w.IsActive {
            return errors.New("wallet tidak aktif")
        }

        // optional: cocokkan radio dengan type wallet
        if in.PaymentMethod == "CASH" && w.Type != models.WalletCash {
            return errors.New("payment_method CASH harus pilih wallet type CASH (laci)")
        }
        if in.PaymentMethod == "BANK" && w.Type != models.WalletBank {
            return errors.New("payment_method BANK harus pilih wallet type BANK")
        }

        now := time.Now().UTC()

        // 4) update wallet balance (IN)
        if err := tx.Model(&models.WarehouseWallet{}).
            Where("id = ?", w.ID).
            Update("balance", gorm.Expr("balance + ?", receive)).Error; err != nil {
            return err
        }

        // 5) insert history penerimaan piutang
        rc := models.PiutangReceipt{
            PiutangID:      p.ID,
            Amount:         receive,
            WalletID:       w.ID,
            PaymentMethod:  in.PaymentMethod,
            ReceivedAt:     now,
            ReceivedByID:   uid,
            Note:           in.Note,
        }
        if err := tx.Create(&rc).Error; err != nil {
            return err
        }

        // 6) update agregat piutang
        newPaid := p.TotalPaid + receive
        isPaid := newPaid >= p.Total

        res := tx.Model(&models.Piutang{}).
            Where("id = ? AND is_paid = false", p.ID).
            Updates(map[string]any{
                "total_paid": gorm.Expr("total_paid + ?", receive),
                "is_paid":    isPaid,
            })
        if res.Error != nil {
            return res.Error
        }
        if res.RowsAffected == 0 {
            return errors.New("gagal update penerimaan")
        }

        // 7) insert wallet tx log (recommended)
        wt := models.WalletTransaction{
            WalletID:  w.ID,
            GudangID:  sr.WarehouseID,
            Type:      models.WalletTxPiutangReceive,
            Direction: "IN",
            Amount:    receive,
            RefType:   "piutang",
            RefID:     p.ID,
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
        c.JSON(code, gin.H{"message": "Gagal terima piutang", "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Penerimaan piutang berhasil"})
}

// USER: history penerimaan piutang (untuk Android)
func PiutangReceiptHistory(c *gin.Context) {
    uid, err := currentUserID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
        return
    }

    id, _ := strconv.Atoi(c.Param("id"))

    // pastikan piutang milik user
    var p models.Piutang
    if err := config.DB.Select("id", "user_id").First(&p, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"message": "Piutang tidak ditemukan"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil piutang", "error": err.Error()})
        return
    }
    if p.UserID != uid {
        c.JSON(http.StatusForbidden, gin.H{"message": "forbidden"})
        return
    }

    var rows []models.PiutangReceipt
    if err := config.DB.
        Where("piutang_id = ?", p.ID).
        Order("received_at ASC, id ASC").
        Find(&rows).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil history", "error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": rows})
}
