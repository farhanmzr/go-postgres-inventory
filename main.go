package main

import (
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"go-postgres-inventory/routes"
	"go-postgres-inventory/utils"

	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	config.ConnectDB()

	// Auto-migrate models (ADMIN & USER terpisah + PERMISSIONS)
	config.DB.AutoMigrate(
		&models.Admin{},
		&models.User{},
		&models.Permission{},
		&models.UserPermission{},
		&models.Barang{},
		&models.Gudang{},
		&models.GrupBarang{},
		&models.Supplier{},
		&models.Permintaan{},
		&models.Customer{},
		&models.PurchaseRequest{},
		&models.PurchaseReqItem{},
	)

	config.SeedPermissions()

	// override secret dari ENV (Render)
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
