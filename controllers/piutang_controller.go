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
	"gorm.io/gorm/clause"
)

// ===== ADMIN: list semua piutang (opsional: ?status=UNPAID/PAY_REQUESTED/PAID)
func PiutangListAdmin(c *gin.Context) {
	var rows []models.Piutang
	q := config.DB.Preload("Items").Order("due_date ASC, id DESC")

	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil piutang", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

// ===== USER: list piutang miliknya
func PiutangListUser(c *gin.Context) {
	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	var rows []models.Piutang
	q := config.DB.Preload("Items").Where("user_id = ?", uid).Order("due_date ASC, id DESC")
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal ambil piutang", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

// ===== USER: klik "Bayar" → ubah UNPAID -> PAY_REQUESTED
func PiutangRequestPay(c *gin.Context) {
	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var p models.Piutang
		if err := tx.Clauses(clauseUpdateLock()).First(&p, id).Error; err != nil {
			return err
		}
		if p.UserID != uid {
			return errors.New("forbidden")
		}
		if p.Status != models.CreditUnpaid {
			return errors.New("status tidak valid (harus UNPAID)")
		}
		now := time.Now().UTC()
		res := tx.Model(&models.Piutang{}).
			Where("id = ? AND status = ?", p.ID, models.CreditUnpaid).
			Updates(map[string]any{
				"status":           models.CreditPayRequested,
				"pay_requested_at": &now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("gagal update status")
		}
		return nil
	})

	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			code = http.StatusNotFound
		}
		c.JSON(code, gin.H{"message": "Gagal mengajukan bayar", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pengajuan pembayaran terkirim"})
}

// ===== ADMIN: approve pembayaran → PAY_REQUESTED -> PAID
func PiutangApprovePayment(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var p models.Piutang
		if err := tx.Clauses(clauseUpdateLock()).First(&p, id).Error; err != nil {
			return err
		}
		if p.Status != models.CreditPayRequested {
			return errors.New("status tidak valid (harus PAY_REQUESTED)")
		}
		now := time.Now().UTC()
		res := tx.Model(&models.Piutang{}).
			Where("id = ? AND status = ?", p.ID, models.CreditPayRequested).
			Updates(map[string]any{
				"status":  models.CreditPaid,
				"paid_at": &now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("gagal update status")
		}
		return nil
	})

	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			code = http.StatusNotFound
		}
		c.JSON(code, gin.H{"message": "Gagal approve pembayaran", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pembayaran disetujui"})
}

// kecil: untuk SELECT ... FOR UPDATE
func clauseUpdateLock() clause.Locking {
    return clause.Locking{Strength: "UPDATE"}
}
