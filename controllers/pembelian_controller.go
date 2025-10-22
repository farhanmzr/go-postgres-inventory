// controllers/purchase_request_user.go
package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"go-postgres-inventory/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PurchaseRequestInput struct {
	TransCode    string         `json:"trans_code"`          // dari UI boleh isi nomor transaksi (opsional), kalau kosong server generate
	ManualCode   *string        `json:"manual_code"`         // biarkan null; admin yang isi nanti
	PurchaseDate time.Time      `json:"purchase_date"`       // wajib <= today
	BuyerName    string         `json:"buyer_name"`          // auto nama user
	WarehouseID  uint           `json:"warehouse_id" binding:"required"`
	SupplierID   uint           `json:"supplier_id" binding:"required"`
	Payment      string         `json:"payment" binding:"required"` // "CASH" | "CREDIT"
	Items        []PurchaseItem `json:"items" binding:"required,min=1"`
}
type PurchaseItem struct {
	BarangID  uint  `json:"barang_id" binding:"required"`
	Qty       int64 `json:"qty" binding:"required,gt=0"`
	BuyPrice  int64 `json:"buy_price" binding:"required,gt=0"`
}

func PurchaseReqCreate(c *gin.Context) {
    var in PurchaseRequestInput
    if err := c.ShouldBindJSON(&in); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Payload tidak valid", "error": err.Error()})
        return
    }

    // validasi tanggal tidak ke depan (gunakan UTC agar konsisten)
    today := time.Now().UTC().Truncate(24 * time.Hour)
    if in.PurchaseDate.After(today) {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Tanggal pembelian tidak boleh ke depan"})
        return
    }

    // validasi payment
    if in.Payment != "CASH" && in.Payment != "CREDIT" {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Metode pembayaran tidak valid"})
        return
    }

    // --- normalize user_id ---
    rawID, _ := c.Get("user_id")
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
        }
    default:
        c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid"})
        return
    }

    // --- cek FK gudang & supplier ---
    var cnt int64
    if err := config.DB.Model(&models.Gudang{}).Where("id = ?", in.WarehouseID).Count(&cnt).Error; err != nil || cnt == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Gudang tidak ditemukan"})
        return
    }
    if err := config.DB.Model(&models.Supplier{}).Where("id = ?", in.SupplierID).Count(&cnt).Error; err != nil || cnt == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Supplier tidak ditemukan"})
        return
    }

    // --- opsional: pastikan semua barang_id ada & memang milik gudang tsb ---
    for _, it := range in.Items {
        var exist int64
        if err := config.DB.Model(&models.Barang{}).
            Where("id = ? AND gudang_id = ?", it.BarangID, in.WarehouseID).
            Count(&exist).Error; err != nil || exist == 0 {
            c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("Barang %d tidak ditemukan di gudang %d", it.BarangID, in.WarehouseID)})
            return
        }
    }

    err := config.DB.Transaction(func(tx *gorm.DB) error {
        // siapkan items
        items := make([]models.PurchaseReqItem, 0, len(in.Items))
        for _, it := range in.Items {
            items = append(items, models.PurchaseReqItem{
                BarangID:  it.BarangID,
                Qty:       it.Qty,
                BuyPrice:  it.BuyPrice,
                LineTotal: it.Qty * it.BuyPrice,
            })
        }

        // generate trans code jika kosong
        code := in.TransCode
        if code == "" {
            var seq int64
            if err := tx.Raw("SELECT COALESCE(MAX(id),0)+1 FROM purchase_requests").Scan(&seq).Error; err != nil {
                return err
            }
            code = utils.GenTransCode(seq, time.Now())
        }

        p := models.PurchaseRequest{
            TransCode:    code,
            ManualCode:   in.ManualCode,
            BuyerName:    in.BuyerName,
            PurchaseDate: in.PurchaseDate,
            WarehouseID:  in.WarehouseID,
            SupplierID:   in.SupplierID,
            Payment:      models.PaymentMethod(in.Payment),
            Status:       models.StatusPending,
            Items:        items,
            CreatedByID:  userID,
        }
        return tx.Create(&p).Error
    })

    if err != nil {
        // log error ke stdout juga biar ketahuan
        fmt.Printf("PurchaseReqCreate error: %v\n", err)
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat permintaan pembelian", "error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"message": "Permintaan pembelian dibuat (PENDING)"})
}


func PurchaseReqMyList(c *gin.Context) {
	uid, _ := c.Get("user_id")
	var rows []models.PurchaseRequest
	if err := config.DB.
		Where("created_by_id = ?", uint(uid.(int))).
		Preload("Supplier").Preload("Warehouse").
		Preload("Items.Barang").
		Order("id DESC").Find(&rows).Error; err != nil {
		c.JSON(500, gin.H{"message": "Gagal mengambil data", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": rows})
}
