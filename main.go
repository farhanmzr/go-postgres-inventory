package main

import (
	"log"
	"os"

	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"go-postgres-inventory/routes"
	"go-postgres-inventory/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	config.ConnectDB()

	// 🧱 Auto-migrate SEMUA tabel yang kamu butuhkan
	if err := config.DB.AutoMigrate(
		&models.Admin{},
		&models.User{},
		&models.Permission{},
		&models.UserPermission{},

		&models.Gudang{},
		&models.GudangBarang{},
		&models.GrupBarang{},
		&models.Barang{},
		&models.StockHistory{},
		&models.Supplier{},
		&models.Customer{},

		&models.PurchaseRequest{},
		&models.PurchaseReqItem{},
		&models.PurchaseInvoice{},
		&models.PurchaseInvoiceItem{},

		&models.UsageRequest{},
		&models.UsageItem{},

		&models.SalesRequest{},
		&models.SalesReqItem{},
		&models.SalesInvoice{},
		&models.SalesInvoiceItem{},

		&models.Piutang{},
		&models.PiutangItem{},
	); err != nil {
		log.Fatalf("❌ AutoMigrate error: %v", err)
	}
	log.Println("✅ AutoMigrate done")

	config.SeedPermissions()

	// Secrets dari ENV (Render)
	if s := os.Getenv("ADMIN_JWT_SECRET"); s != "" {
		utils.AdminSecret = []byte(s)
	}
	if s := os.Getenv("USER_JWT_SECRET"); s != "" {
		utils.UserSecret = []byte(s)
	}

	r := gin.Default()
	routes.SetupRoutes(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "🚀 Inventory API is running"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	_ = r.Run(":" + port)
}
