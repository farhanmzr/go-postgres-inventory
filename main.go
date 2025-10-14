package main

import (
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"go-postgres-inventory/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	config.ConnectDB()

	// Auto migrate semua model
	config.DB.AutoMigrate(&models.User{}, &models.Barang{})

	r := gin.Default()
	routes.SetupRoutes(r)

	r.Run(":8080")
}
