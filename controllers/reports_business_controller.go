package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-postgres-inventory/config"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

/*
ROUTE YANG DICOBAKAN (sesuai pola kamu sebelumnya):

ADMIN (lihat semua):
GET /api/admin/reports/purchases
GET /api/admin/reports/sales
GET /api/admin/reports/usage
GET /api/admin/reports/permintaan
GET /api/admin/reports/profit/barang

USER (auto filter by created_by_id):
GET /api/user/reports/purchases
GET /api/user/reports/sales
GET /api/user/reports/usage
GET /api/user/reports/permintaan
GET /api/user/reports/profit/barang

Query umum yang didukung:
- date_from=YYYY-MM-DD
- date_to=YYYY-MM-DD
- page=1, page_size=50
- sort=... (lihat masing2 controller)
- filter spesifik (warehouse_id, customer_id, supplier_id, payment, status)
*/

// ================= Common helpers =================

func getIntQ(c *gin.Context, key string, def int) int {
	v, _ := strconv.Atoi(c.DefaultQuery(key, strconv.Itoa(def)))
	if v <= 0 {
		return def
	}
	return v
}

func getUintQPtr(c *gin.Context, key string) *uint {
	if v := strings.TrimSpace(c.Query(key)); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil && n > 0 {
			u := uint(n)
			return &u
		}
	}
	return nil
}

func getDatePtr(c *gin.Context, key string) *time.Time {
	if s := strings.TrimSpace(c.Query(key)); s != "" {
		// format: 2006-01-02
		if t, err := time.Parse("2006-01-02", s); err == nil {
			// gunakan awal hari utk from, akhir hari utk to (nanti di controller)
			return &t
		}
	}
	return nil
}

func applyPagingSort(q *gorm.DB, page, size int, sortBy string, allowed map[string]string, defaultDesc string) *gorm.DB {
	// sorting
	if col, ok := allowed[sortBy]; ok {
		q = q.Order(col + " ASC")
	} else if _, ok := allowed["-"+sortBy]; ok && strings.HasPrefix(sortBy, "-") {
		// handled below, but keep pattern concise
	} else {
		// handle signed keys
		if strings.HasPrefix(sortBy, "-") {
			key := sortBy[1:]
			if col, ok := allowed[key]; ok {
				q = q.Order(col + " DESC")
			} else {
				q = q.Order(defaultDesc)
			}
		} else if sortBy != "" {
			if col, ok := allowed[sortBy]; ok {
				q = q.Order(col + " ASC")
			} else {
				q = q.Order(defaultDesc)
			}
		} else {
			q = q.Order(defaultDesc)
		}
	}

	offset := (page - 1) * size
	return q.Offset(offset).Limit(size)
}

// ================= Laporan Pembelian =================

type PurchaseRow struct {
	ID            uint      `json:"id"`
	TransCode     string    `json:"trans_code"`
	ManualCode    *string   `json:"manual_code"`
	PurchaseDate  time.Time `json:"purchase_date"`
	WarehouseID   uint      `json:"warehouse_id"`
	WarehouseName string    `json:"warehouse_name"`
	SupplierID    uint      `json:"supplier_id"`
	SupplierName  string    `json:"supplier_name"`
	Payment       string    `json:"payment"`
	ItemCount     int64     `json:"item_count"`
	TotalQty      int64     `json:"total_qty"`
	Subtotal      int64     `json:"subtotal"` // SUM(LineTotal)
	CreatedByID   uint      `json:"created_by_id"`
}

type PurchaseSummary struct {
	CountTx  int64 `json:"count_tx"`
	TotalQty int64 `json:"total_qty"`
	Subtotal int64 `json:"subtotal"`
}

func ReportPurchasesAdmin(c *gin.Context) { reportPurchases(c, nil) }
func ReportPurchasesUser(c *gin.Context) {
	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reportPurchases(c, &uid)
}

func reportPurchases(c *gin.Context, onlyUserID *uint) {
	db := config.DB

	page := getIntQ(c, "page", 1)
	size := getIntQ(c, "page_size", 50)
	sortBy := c.DefaultQuery("sort", "-purchase_date")
	dateFrom := getDatePtr(c, "date_from")
	dateTo := getDatePtr(c, "date_to")
	supplierID := getUintQPtr(c, "supplier_id")
	warehouseID := getUintQPtr(c, "warehouse_id")
	payment := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("payment", ""))) // CASH/CREDIT

	q := db.Table("purchase_requests pr").
		Select(`
			pr.id,
			pr.trans_code,
			pr.manual_code,
			pr.purchase_date,
			pr.warehouse_id,
			gd.nama AS warehouse_name,
			pr.supplier_id,
			sp.nama AS supplier_name,
			pr.payment,
			pr.created_by_id,
			COUNT(it.id) AS item_count,
			COALESCE(SUM(it.qty),0) AS total_qty,
			COALESCE(SUM(it.line_total),0) AS subtotal
		`).
		Joins("INNER JOIN gudangs gd ON gd.id = pr.warehouse_id").
		Joins("INNER JOIN suppliers sp ON sp.id = pr.supplier_id").
		Joins("LEFT JOIN purchase_req_items it ON it.purchase_request_id = pr.id").
		Group("pr.id, gd.nama, sp.nama")

	// filter tanggal
	if dateFrom != nil {
		q = q.Where("pr.purchase_date >= ?", dateFrom.Truncate(24*time.Hour))
	}
	if dateTo != nil {
		q = q.Where("pr.purchase_date < ?", dateTo.Truncate(24*time.Hour).Add(24*time.Hour))
	}
	if supplierID != nil {
		q = q.Where("pr.supplier_id = ?", *supplierID)
	}
	if warehouseID != nil {
		q = q.Where("pr.warehouse_id = ?", *warehouseID)
	}
	if payment != "" {
		q = q.Where("pr.payment = ?", payment)
	}
	if onlyUserID != nil {
		q = q.Where("pr.created_by_id = ?", *onlyUserID)
	}

	// total summary (pakai subquery agar LIMIT/OFFSET tidak mengganggu)
	var summary PurchaseSummary
	if err := db.Table("(?) as x", q.Session(&gorm.Session{})).
		Select("COUNT(*) AS count_tx, COALESCE(SUM(total_qty),0) AS total_qty, COALESCE(SUM(subtotal),0) AS subtotal").
		Scan(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// sorting + paging
	allowed := map[string]string{
		"id":            "pr.id",
		"purchase_date": "pr.purchase_date",
		"subtotal":      "subtotal",
		"total_qty":     "total_qty",
	}
	q = applyPagingSort(q, page, size, sortBy, allowed, "pr.purchase_date DESC")

	var rows []PurchaseRow
	if err := q.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":    summary,
		"data":       rows,
		"pagination": gin.H{"page": page, "page_size": size},
	})
}

// ================= Laporan Penjualan (berdasarkan INVOICE) =================

type SalesRow struct {
	InvoiceNo      string    `json:"invoice_no"`
	InvoiceDate    time.Time `json:"invoice_date"`
	Username       string    `json:"username"`
	Payment        string    `json:"payment"`
	Subtotal       int64     `json:"subtotal"`
	Discount       int64     `json:"discount"`
	Tax            int64     `json:"tax"`
	GrandTotal     int64     `json:"grand_total"`
	SalesRequestID uint      `json:"sales_request_id"`
	CreatedByID    uint      `json:"created_by_id"`
	CustomerID     uint      `json:"customer_id"`
	CustomerName   string    `json:"customer_name"`
	WarehouseID    uint      `json:"warehouse_id"`
	WarehouseName  string    `json:"warehouse_name"`
	ItemCount      int64     `json:"item_count"`
	TotalQty       int64     `json:"total_qty"`
}

type SalesSummary struct {
	CountTx  int64 `json:"count_tx"`
	TotalQty int64 `json:"total_qty"`
	GrandTot int64 `json:"grand_total"`
}

func ReportSalesAdmin(c *gin.Context) { reportSales(c, nil) }
func ReportSalesUser(c *gin.Context) {
	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reportSales(c, &uid)
}

func reportSales(c *gin.Context, onlyUserID *uint) {
	db := config.DB

	page := getIntQ(c, "page", 1)
	size := getIntQ(c, "page_size", 50)
	sortBy := c.DefaultQuery("sort", "-invoice_date")
	dateFrom := getDatePtr(c, "date_from")
	dateTo := getDatePtr(c, "date_to")
	warehouseID := getUintQPtr(c, "warehouse_id")
	customerID := getUintQPtr(c, "customer_id")
	payment := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("payment", "")))

	q := db.Table("sales_invoices si").
		Select(`
			si.invoice_no,
			si.invoice_date,
			si.username,
			si.payment,
			si.subtotal,
			si.discount,
			si.tax,
			si.grand_total,
			sr.id as sales_request_id,
			sr.created_by_id,
			sr.customer_id,
			cu.nama as customer_name,
			sr.warehouse_id,
			gd.nama as warehouse_name,
			COUNT(ii.id) as item_count,
			COALESCE(SUM(ii.qty),0) as total_qty
		`).
		Joins("INNER JOIN sales_requests sr ON sr.id = si.sales_request_id").
		Joins("INNER JOIN gudangs gd ON gd.id = sr.warehouse_id").
		Joins("INNER JOIN customers cu ON cu.id = sr.customer_id").
		Joins("LEFT JOIN sales_invoice_items ii ON ii.sales_invoice_id = si.sales_request_id").
		Group("si.invoice_no, si.invoice_date, si.username, si.payment, si.subtotal, si.discount, si.tax, si.grand_total, sr.id, sr.created_by_id, sr.customer_id, cu.nama, sr.warehouse_id, gd.nama")

	// filters
	if dateFrom != nil {
		q = q.Where("si.invoice_date >= ?", dateFrom.Truncate(24*time.Hour))
	}
	if dateTo != nil {
		q = q.Where("si.invoice_date < ?", dateTo.Truncate(24*time.Hour).Add(24*time.Hour))
	}
	if warehouseID != nil {
		q = q.Where("sr.warehouse_id = ?", *warehouseID)
	}
	if customerID != nil {
		q = q.Where("sr.customer_id = ?", *customerID)
	}
	if payment != "" {
		q = q.Where("si.payment = ?", payment)
	}
	if onlyUserID != nil {
		q = q.Where("sr.created_by_id = ?", *onlyUserID)
	}

	// summary
	var summary SalesSummary
	if err := db.Table("(?) as x", q.Session(&gorm.Session{})).
		Select("COUNT(*) as count_tx, COALESCE(SUM(total_qty),0) as total_qty, COALESCE(SUM(grand_total),0) as grand_total").
		Scan(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	allowed := map[string]string{
		"invoice_date": "si.invoice_date",
		"grand_total":  "si.grand_total",
		"total_qty":    "total_qty",
	}
	q = applyPagingSort(q, page, size, sortBy, allowed, "si.invoice_date DESC")

	var rows []SalesRow
	if err := q.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":    summary,
		"data":       rows,
		"pagination": gin.H{"page": page, "page_size": size},
	})
}

// ================= Laporan Pemakaian =================

type UsageRow struct {
	ID            uint      `json:"id"`
	TransCode     string    `json:"trans_code"`
	ManualCode    *string   `json:"manual_code"`
	UsageDate     time.Time `json:"usage_date"`
	RequesterName string    `json:"requester_name"`
	PenggunaName  string    `json:"pengguna_name"`
	Status        string    `json:"status"`
	WarehouseID   uint      `json:"warehouse_id"`
	WarehouseName string    `json:"warehouse_name"`
	CustomerID    uint      `json:"customer_id"`
	CustomerName  string    `json:"customer_name"`
	ItemCount     int64     `json:"item_count"`
	TotalQty      int64     `json:"total_qty"`
	CreatedByID   uint      `json:"created_by_id"`
}

type UsageSummary struct {
	CountTx  int64 `json:"count_tx"`
	TotalQty int64 `json:"total_qty"`
}

func ReportUsageAdmin(c *gin.Context) { reportUsage(c, nil) }
func ReportUsageUser(c *gin.Context) {
	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reportUsage(c, &uid)
}

func reportUsage(c *gin.Context, onlyUserID *uint) {
	db := config.DB

	page := getIntQ(c, "page", 1)
	size := getIntQ(c, "page_size", 50)
	sortBy := c.DefaultQuery("sort", "-usage_date")
	dateFrom := getDatePtr(c, "date_from")
	dateTo := getDatePtr(c, "date_to")
	warehouseID := getUintQPtr(c, "warehouse_id")
	customerID := getUintQPtr(c, "customer_id")
	status := strings.TrimSpace(c.DefaultQuery("status", "")) // BELUM_DIPROSES/SUDAH_DIPROSES

	q := db.Table("usage_requests ur").
		Select(`
			ur.id,
			ur.trans_code,
			ur.manual_code,
			ur.usage_date,
			ur.requester_name,
			ur.pengguna_name,
			ur.status,
			ur.warehouse_id,
			gd.nama AS warehouse_name,
			ur.customer_id,
			cu.nama AS customer_name,
			ur.created_by_id,
			COUNT(ui.id) AS item_count,
			COALESCE(SUM(ui.qty),0) AS total_qty
		`).
		Joins("INNER JOIN gudangs gd ON gd.id = ur.warehouse_id").
		Joins("INNER JOIN customers cu ON cu.id = ur.customer_id").
		Joins("LEFT JOIN usage_items ui ON ui.usage_request_id = ur.id").
		Group("ur.id, gd.nama, cu.nama")

	if dateFrom != nil {
		q = q.Where("ur.usage_date >= ?", dateFrom.Truncate(24*time.Hour))
	}
	if dateTo != nil {
		q = q.Where("ur.usage_date < ?", dateTo.Truncate(24*time.Hour).Add(24*time.Hour))
	}
	if warehouseID != nil {
		q = q.Where("ur.warehouse_id = ?", *warehouseID)
	}
	if customerID != nil {
		q = q.Where("ur.customer_id = ?", *customerID)
	}
	if status != "" {
		q = q.Where("ur.status = ?", status)
	}
	if onlyUserID != nil {
		q = q.Where("ur.created_by_id = ?", *onlyUserID)
	}

	var summary UsageSummary
	if err := db.Table("(?) as x", q.Session(&gorm.Session{})).
		Select("COUNT(*) as count_tx, COALESCE(SUM(total_qty),0) as total_qty").
		Scan(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	allowed := map[string]string{
		"usage_date": "ur.usage_date",
		"total_qty":  "total_qty",
		"id":         "ur.id",
	}
	q = applyPagingSort(q, page, size, sortBy, allowed, "ur.usage_date DESC")

	var rows []UsageRow
	if err := q.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":    summary,
		"data":       rows,
		"pagination": gin.H{"page": page, "page_size": size},
	})
}

// ================= Laporan Permintaan =================

type PermintaanRow struct {
	ID                uint      `json:"id"`
	TanggalPermintaan time.Time `json:"tanggal_permintaan"`
	NamaPeminta       string    `json:"nama_peminta"`
	KodePeminta       string    `json:"kode_peminta"`
	Keterangan        string    `json:"keterangan"`
	CreatedByID       uint      `json:"created_by_id"`
}

type PermintaanSummary struct {
	CountTx int64 `json:"count_tx"`
}

func ReportPermintaanAdmin(c *gin.Context) { reportPermintaan(c, nil) }
func ReportPermintaanUser(c *gin.Context) {
	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reportPermintaan(c, &uid)
}

func reportPermintaan(c *gin.Context, onlyUserID *uint) {
	db := config.DB

	page := getIntQ(c, "page", 1)
	size := getIntQ(c, "page_size", 50)
	sortBy := c.DefaultQuery("sort", "-tanggal")
	dateFrom := getDatePtr(c, "date_from")
	dateTo := getDatePtr(c, "date_to")

	q := db.Table("permintaans p").
		Select(`
        p.id,
        p.tanggal_permintaan,
        p.nama_peminta,
        p.kode_peminta,
        p.keterangan,
        p.created_by_id
    `).
		Where(`
        EXISTS (
            SELECT 1
            FROM permintaan_items pi
            WHERE pi.permintaan_id = p.id
        )
    `)

	if dateFrom != nil {
		q = q.Where("p.tanggal_permintaan >= ?", dateFrom.Truncate(24*time.Hour))
	}
	if dateTo != nil {
		q = q.Where("p.tanggal_permintaan < ?", dateTo.Truncate(24*time.Hour).Add(24*time.Hour))
	}
	if onlyUserID != nil {
		q = q.Where("p.created_by_id = ?", *onlyUserID)
	}

	var summary PermintaanSummary
	if err := db.Table("(?) as x", q.Session(&gorm.Session{})).
		Select("COUNT(*) as count_tx").
		Scan(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	allowed := map[string]string{
		"tanggal": "p.tanggal_permintaan",
		"id":      "p.id",
	}
	q = applyPagingSort(q, page, size, sortBy, allowed, "p.tanggal_permintaan DESC")

	var rows []PermintaanRow
	if err := q.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":    summary,
		"data":       rows,
		"pagination": gin.H{"page": page, "page_size": size},
	})
}

// ================= Laporan Keuntungan Per Barang =================

type ProfitPerBarangRow struct {
	BarangID   uint    `json:"barang_id"`
	Kode       string  `json:"kode"`
	Nama       string  `json:"nama"`
	Satuan     string  `json:"satuan"`
	QtySold    int64   `json:"qty_sold"`
	Revenue    int64   `json:"revenue"`   // SUM(price * qty)
	Cost       int64   `json:"cost"`      // SUM(cost_price * qty)
	Profit     int64   `json:"profit"`    // SUM(profit_total)
	AvgPrice   float64 `json:"avg_price"` // revenue / qty
	AvgCost    float64 `json:"avg_cost"`  // cost / qty
	ProfitPerU float64 `json:"profit_per_unit"`
}

type ProfitSummary struct {
	TotalQty int64 `json:"total_qty"`
	Revenue  int64 `json:"revenue"`
	Cost     int64 `json:"cost"`
	Profit   int64 `json:"profit"`
}

func ReportProfitPerBarangAdmin(c *gin.Context) { reportProfitPerBarang(c, nil) }
func ReportProfitPerBarangUser(c *gin.Context) {
	uid, err := currentUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reportProfitPerBarang(c, &uid)
}

// sumber data: sales_invoice_items (ii) + sales_invoices (si) + sales_requests (sr) untuk created_by_id
func reportProfitPerBarang(c *gin.Context, onlyUserID *uint) {
	db := config.DB

	page := getIntQ(c, "page", 1)
	size := getIntQ(c, "page_size", 50)
	sortBy := c.DefaultQuery("sort", "-profit")
	dateFrom := getDatePtr(c, "date_from")
	dateTo := getDatePtr(c, "date_to")
	barangID := getUintQPtr(c, "barang_id")
	warehouseID := getUintQPtr(c, "warehouse_id")
	customerID := getUintQPtr(c, "customer_id")

	base := db.Table("sales_invoice_items ii").
		Joins("INNER JOIN sales_invoices si ON si.sales_request_id = ii.sales_invoice_id").
		Joins("INNER JOIN sales_requests sr ON sr.id = si.sales_request_id").
		Joins("INNER JOIN barangs b ON b.id = ii.barang_id")

	if dateFrom != nil {
		base = base.Where("si.invoice_date >= ?", dateFrom.Truncate(24*time.Hour))
	}
	if dateTo != nil {
		base = base.Where("si.invoice_date < ?", dateTo.Truncate(24*time.Hour).Add(24*time.Hour))
	}
	if barangID != nil {
		base = base.Where("ii.barang_id = ?", *barangID)
	}
	if warehouseID != nil {
		base = base.Where("sr.warehouse_id = ?", *warehouseID)
	}
	if customerID != nil {
		base = base.Where("sr.customer_id = ?", *customerID)
	}
	if onlyUserID != nil {
		base = base.Where("sr.created_by_id = ?", *onlyUserID)
	}

	agg := base.Select(`
		ii.barang_id AS barang_id,
		b.kode AS kode,
		b.nama AS nama,
		b.satuan AS satuan,
		COALESCE(SUM(ii.qty),0) AS qty_sold,
		COALESCE(SUM(ii.price * ii.qty),0) AS revenue,
		COALESCE(SUM(ii.cost_price * ii.qty),0) AS cost,
		COALESCE(SUM(ii.profit_total),0) AS profit
	`).Group("ii.barang_id, b.kode, b.nama, b.satuan")

	// summary keseluruhan
	var summary ProfitSummary
	if err := db.Table("(?) as x", agg.Session(&gorm.Session{})).
		Select("COALESCE(SUM(qty_sold),0) AS total_qty, COALESCE(SUM(revenue),0) AS revenue, COALESCE(SUM(cost),0) AS cost, COALESCE(SUM(profit),0) AS profit").
		Scan(&summary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// paging + sorting
	allowed := map[string]string{
		"nama":    "nama",
		"kode":    "kode",
		"qty":     "qty_sold",
		"revenue": "revenue",
		"cost":    "cost",
		"profit":  "profit",
	}
	// terapkan sort dan paging ke query utama
	agg = applyPagingSort(agg, page, size, sortBy, allowed, "profit DESC")

	var raw []struct {
		BarangID uint
		Kode     string
		Nama     string
		Satuan   string
		QtySold  int64
		Revenue  int64
		Cost     int64
		Profit   int64
	}
	if err := agg.Scan(&raw).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// hitung avg price/cost di Go
	out := make([]ProfitPerBarangRow, 0, len(raw))
	for _, r := range raw {
		row := ProfitPerBarangRow{
			BarangID: r.BarangID,
			Kode:     r.Kode,
			Nama:     r.Nama,
			Satuan:   r.Satuan,
			QtySold:  r.QtySold,
			Revenue:  r.Revenue,
			Cost:     r.Cost,
			Profit:   r.Profit,
		}
		if r.QtySold > 0 {
			row.AvgPrice = float64(r.Revenue) / float64(r.QtySold)
			row.AvgCost = float64(r.Cost) / float64(r.QtySold)
			row.ProfitPerU = float64(r.Profit) / float64(r.QtySold)
		}
		out = append(out, row)
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":    summary,
		"data":       out,
		"pagination": gin.H{"page": page, "page_size": size},
	})
}
