package main

import (
	"go-postgres-inventory/config"
	"go-postgres-inventory/models"
	"go-postgres-inventory/routes"

	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	config.ConnectDB()

	// Auto migrate semua model
	config.DB.AutoMigrate(&models.User{}, &models.Barang{})

	r := gin.Default()
	routes.SetupRoutes(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "🚀 Go API with PostgreSQL is running successfully!",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)

}
