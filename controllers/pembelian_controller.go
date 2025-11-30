// controllers/purchase_request_user.go
package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PurchaseRequestInput struct {
	TransCode    string         `json:"trans_code"`    // dari UI boleh isi nomor transaksi (opsional), kalau kosong server generate
	ManualCode   *string        `json:"manual_code"`   // biarkan null; admin yang isi nanti
	PurchaseDate time.Time      `json:"purchase_date"` // wajib <= today
	BuyerName    string         `json:"buyer_name"`    // auto nama user
	WarehouseID  uint           `json:"warehouse_id" binding:"required"`
	SupplierID   uint           `json:"supplier_id" binding:"required"`
	Payment      string         `json:"payment" binding:"required"` // "CASH" | "CREDIT"
	Items        []PurchaseItem `json:"items" binding:"required,min=1"`
}

type PurchaseItem struct {
	BarangID uint  `json:"barang_id" binding:"required"`
	Qty      int64 `json:"qty" binding:"required,gt=0"`
	BuyPrice int64 `json:"buy_price" binding:"required,gt=0"`
}

func CreatePembelian(c *gin.Context) {
	var in PurchaseRequestInput
	var pembelianData models.PurchaseRequest
	var inv models.PurchaseInvoice

	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Payload tidak valid", "error": err.Error()})
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
		if err := config.DB.Model(&models.GudangBarang{}).
			Where("barang_id = ? AND gudang_id = ?", it.BarangID, in.WarehouseID).
			Count(&exist).Error; err != nil || exist == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": fmt.Sprintf("Barang %d tidak ditemukan di gudang %d", it.BarangID, in.WarehouseID),
			})
			return
		}
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {

		// 1) Siapkan items untuk PurchaseRequest
		items := make([]models.PurchaseReqItem, 0, len(in.Items))
		for _, it := range in.Items {
			items = append(items, models.PurchaseReqItem{
				BarangID:  it.BarangID,
				Qty:       it.Qty,
				BuyPrice:  it.BuyPrice,
				LineTotal: it.Qty * it.BuyPrice,
			})
		}

		// 2) Insert PurchaseRequest (header)
		pembelianData = models.PurchaseRequest{
			TransCode:    in.TransCode,
			ManualCode:   in.ManualCode,
			BuyerName:    in.BuyerName,
			PurchaseDate: in.PurchaseDate,
			WarehouseID:  in.WarehouseID,
			SupplierID:   in.SupplierID,
			Payment:      models.PaymentMethod(in.Payment),
			Items:        items,
			CreatedByID:  userID,
		}
		if err := tx.Create(&pembelianData).Error; err != nil {
			return err
		}

		// 4) Tambah stok & update harga_beli (hanya jika berubah)
		for _, it := range in.Items {
			// tambah stok di GudangBarang
			res := tx.Model(&models.GudangBarang{}).
				Where("barang_id = ? AND gudang_id = ?", it.BarangID, pembelianData.WarehouseID).
				UpdateColumn("stok", gorm.Expr("stok + ?", it.Qty))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return fmt.Errorf("barang %d tidak ditemukan di gudang %d", it.BarangID, pembelianData.WarehouseID)
			}

			// update harga beli terakhir di GudangBarang
			if err := tx.Model(&models.GudangBarang{}).
				Where("barang_id = ? AND gudang_id = ? AND harga_beli <> ?", it.BarangID, pembelianData.WarehouseID, float64(it.BuyPrice)).
				Update("harga_beli", float64(it.BuyPrice)).Error; err != nil {
				return err
			}
		}

		// 5) Buat Invoice (header + items) dari data pembelian
		var subtotal int64 = 0
		invItems := make([]models.PurchaseInvoiceItem, 0, len(in.Items))
		for _, it := range in.Items {
			line := it.Qty * it.BuyPrice
			subtotal += line
			invItems = append(invItems, models.PurchaseInvoiceItem{
				BarangID:  it.BarangID,
				Qty:       it.Qty,
				Price:     it.BuyPrice,
				LineTotal: line,
			})
		}
		discount := int64(0)
		tax := int64(0)
		grand := subtotal - discount + tax

		inv = models.PurchaseInvoice{
			PurchaseRequestID: pembelianData.ID,
			InvoiceNo:         pembelianData.TransCode, // nomor transaksi = transcode pembelian
			BuyerName:         pembelianData.BuyerName,
			Payment:           pembelianData.Payment,
			InvoiceDate:       pembelianData.PurchaseDate, // tanggal invoice = tanggal pembelian
			Subtotal:          subtotal,
			Discount:          discount,
			Tax:               tax,
			GrandTotal:        grand,
			Items:             invItems,
		}
		if err := tx.Create(&inv).Error; err != nil {
			return err
		}

		// 6) Jika payment CREDIT -> buat Piutang
		if pembelianData.Payment == models.PaymentCredit {
			due := inv.InvoiceDate.AddDate(0, 0, 7)

			// siapkan items snapshot dari invoice
			piuItems := make([]models.PiutangItem, 0, len(invItems))
			for _, iv := range invItems {
				// ambil nama & kode barang untuk snapshot
				var b models.Barang
				if err := tx.Select("id, nama, kode").First(&b, iv.BarangID).Error; err != nil {
					return err
				}
				piuItems = append(piuItems, models.PiutangItem{
					BarangID:  iv.BarangID,
					Nama:      b.Nama,
					Kode:      b.Kode,
					Qty:       iv.Qty,
					Price:     iv.Price,
					LineTotal: iv.LineTotal,
				})
			}

			piu := models.Piutang{
				UserID:      userID,
				UserName:    pembelianData.BuyerName, // display
				Source:      models.CreditFromPurchase,
				SourceID:    inv.PurchaseRequestID, // invoice PK = PurchaseRequestID
				InvoiceNo:   inv.InvoiceNo,
				InvoiceDate: inv.InvoiceDate,
				DueDate:     due,
				Total:       inv.GrandTotal,
				Status:      models.CreditUnpaid,
				Items:       piuItems,
			}
			if err := tx.Create(&piu).Error; err != nil {
				return err
			}
		}

		return nil

	})

	if err != nil {
		// log error ke stdout juga biar ketahuan
		fmt.Printf("PurchaseReqCreate error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat permintaan pembelian", "error": err.Error()})
		return
	}

	// (Opsional) balikin info ini, tapi tidak wajib karena ID sama
	c.JSON(http.StatusCreated, gin.H{
		"message":             "Berhasil melakukan Pembelian",
		"purchase_request_id": pembelianData.ID,
		"invoice_id":          inv.PurchaseRequestID, // == purchase_request_id
		"invoice_no":          inv.InvoiceNo,
	})
}

func PurchaseReqMyList(c *gin.Context) {
	// --- normalize user_id from context (hindari panic) ---
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
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user_id tidak valid (tipe tidak dikenal)"})
		return
	}

	var rows []models.PurchaseRequest
	if err := config.DB.
		Where("created_by_id = ?", userID).
		Preload("Supplier").
		Preload("Warehouse").
		Preload("Items.Barang").
		Order("id DESC").
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal mengambil data",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil semua data Pembelian", "data": rows})
}

func PurchaseInvoiceDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id tidak valid"})
		return
	}
	var inv models.PurchaseInvoice
	if err := config.DB.
		Preload("Items.Barang").
		First(&inv, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "invoice tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "gagal mengambil data", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil mengambil data Invoice", "data": inv})
}

func DeletePembelian(c *gin.Context) {
	// ambil id purchase_request dari path param
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id tidak valid"})
		return
	}

	var pr models.PurchaseRequest

	// load header + items
	if err := config.DB.
		Preload("Items").
		First(&pr, uint(id)).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Pembelian tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil data pembelian", "error": err.Error()})
		return
	}

	// (opsional) cek otorisasi: hanya pembuat / admin yang boleh hapus
	// rawID, _ := c.Get("user_id")
	// ...

	err = config.DB.Transaction(func(tx *gorm.DB) error {

		// 1) Revert stok gudang (kurangi lagi sesuai qty pembelian)
		for _, it := range pr.Items {
			var gb models.GudangBarang
			if err := tx.
				Where("barang_id = ? AND gudang_id = ?", it.BarangID, pr.WarehouseID).
				First(&gb).Error; err != nil {

				return fmt.Errorf("data stok gudang untuk barang %d tidak ditemukan: %w", it.BarangID, err)
			}

			if err := tx.Model(&models.GudangBarang{}).
				Where("barang_id = ? AND gudang_id = ?", it.BarangID, pr.WarehouseID).
				UpdateColumn("stok", gorm.Expr("stok - ?", it.Qty)).Error; err != nil {
				return err
			}
		}

		// 2) Kalau payment CREDIT, hapus piutang yang berasal dari pembelian ini
		if pr.Payment == models.PaymentCredit {
			// waktu create: Source = PURCHASE, SourceID = inv.PurchaseRequestID (== pr.ID)
			if err := tx.
				Where("source = ? AND source_id = ?", models.CreditFromPurchase, pr.ID).
				Delete(&models.Piutang{}).Error; err != nil {
				return err
			}
			// PiutangItem ikut kehapus karena sudah OnDelete:CASCADE
		}

		// 3) Hapus invoice (dan otomatis detailnya via OnDelete:CASCADE)
		// PK invoice = PurchaseRequestID, jadi cukup where di situ
		if err := tx.
			Where("purchase_request_id = ?", pr.ID).
			Delete(&models.PurchaseInvoice{}).Error; err != nil {
			return err
		}

		// 4) Hapus detail purchase request
		if err := tx.
			Where("purchase_request_id = ?", pr.ID).
			Delete(&models.PurchaseReqItem{}).Error; err != nil {
			return err
		}

		// 5) Terakhir, hapus header purchase request
		if err := tx.Delete(&pr).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		fmt.Printf("DeletePembelian error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Gagal menghapus Pembelian",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil menghapus Pembelian (termasuk invoice, piutang jika ada, & penyesuaian stok)",
	})
}
