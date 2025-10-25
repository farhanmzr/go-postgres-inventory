package controllers

import (
	"fmt"
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UsageCreateInput struct {
	TransCode  string           `json:"trans_code"`                    // opsional; jika kosong server generate
	ManualCode *string          `json:"manual_code"`                   // biarkan null; admin isi kalau perlu
	UsageDate  time.Time        `json:"usage_date" binding:"required"` // wajib <= today
	Requester  string           `json:"requester" binding:"required"`  // auto dari user ctx nama/username
	Items      []UsageItemInput `json:"items" binding:"required,min=1"`
}

type UsageItemInput struct {
	BarangID   uint    `json:"barang_id" binding:"required"`
	CustomerID uint    `json:"customer_id" binding:"required"`
	Qty        int64   `json:"qty" binding:"required,gt=0"`
	Note       *string `json:"note"`
}

func UsageCreate(c *gin.Context) {
	var in UsageCreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "payload tidak valid", "error": err.Error()})
		return
	}

	// validasi tanggal tidak ke depan
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if in.UsageDate.After(today) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "tanggal pemakaian tidak boleh ke depan"})
		return
	}

	// ambil user_id dari context (normalize)
	rawID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak ditemukan"})
		return
	}
	var userID uint
	switch v := rawID.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case int64:
		userID = uint(v)
	case float64:
		userID = uint(v)
	case string:
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			userID = uint(n)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid (tipe)"})
		return
	}

	// validasi FK barang & customer
	for i, it := range in.Items {
		var cnt int64
		if err := config.DB.Model(&models.Barang{}).Where("id = ?", it.BarangID).Count(&cnt).Error; err != nil || cnt == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("barang index %d tidak ditemukan", i)})
			return
		}
		if err := config.DB.Model(&models.Customer{}).Where("id = ?", it.CustomerID).Count(&cnt).Error; err != nil || cnt == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("customer index %d tidak ditemukan", i)})
			return
		}
	}

	// transaksi: insert header+items → set trans_code = ID (increment angka)
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// siapkan items
		items := make([]models.UsageItem, 0, len(in.Items))
		for _, it := range in.Items {
			items = append(items, models.UsageItem{
				BarangID:   it.BarangID,
				CustomerID: it.CustomerID,
				Qty:        it.Qty,
				ItemStatus: models.ItemPending,
				Note:       it.Note,
			})
		}

		// insert header dengan trans_code sementara (unik)
		u := models.UsageRequest{
			TransCode:     fmt.Sprintf("tmp-%d", time.Now().UnixNano()),
			ManualCode:    in.ManualCode,
			UsageDate:     in.UsageDate,
			RequesterName: in.Requester,
			CreatedByID:   userID,
			Status:        models.UsageBelumDiproses,
			Items:         items,
		}
		if err := tx.Create(&u).Error; err != nil {
			return err
		}

		// ganti ke increment angka = ID (aman, konsisten)
		code := fmt.Sprintf("%d", u.ID)
		if err := tx.Model(&models.UsageRequest{}).
			Where("id = ?", u.ID).
			Update("trans_code", code).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "gagal membuat pemakaian", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "pemakaian dibuat (BELUM_DIPROSES)"})
}

func UsageMyList(c *gin.Context) {
	rawID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak ditemukan"})
		return
	}

	var userID uint
	switch v := rawID.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case int64:
		userID = uint(v)
	case float64:
		userID = uint(v)
	case string:
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			userID = uint(n)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid (tipe)"})
		return
	}

	var rows []models.UsageRequest
	if err := config.DB.
		Where("created_by_id = ?", userID).
		Preload("Items.Barang").
		Preload("Items.Customer").
		Order("id DESC").
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "gagal mengambil data", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rows})
}
