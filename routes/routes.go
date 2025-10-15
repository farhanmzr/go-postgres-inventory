package routes

import (
	"go-postgres-inventory/controllers"
	"go-postgres-inventory/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/register", controllers.Register)
		api.POST("/login", controllers.Login)
		api.GET("/profile", controllers.Profile)

		api.GET("/users", middlewares.AuthMiddleware(), middlewares.AdminOnly(), controllers.GetAllUsers)

		barang := api.Group("/barang", middlewares.AuthMiddleware())
		{
			barang.GET("/", controllers.GetAllBarang)
			barang.GET("/:id", controllers.GetBarangByID)

			// Hanya admin yang boleh CRUD
			barang.POST("/", middlewares.AdminOnly(), controllers.CreateBarang)
			barang.PUT("/:id", middlewares.AdminOnly(), controllers.UpdateBarang)
			barang.DELETE("/:id", middlewares.AdminOnly(), controllers.DeleteBarang)
		}

		pembelian := api.Group("/pembelian")
		{
			// user membuat pembelian
			pembelian.POST("/", middlewares.AuthMiddleware(), controllers.CreatePembelian)

			// admin melihat semua pembelian
			pembelian.GET("/", middlewares.AuthMiddleware(), middlewares.AdminOnly(), controllers.GetAllPembelian)

			// admin mengubah status
			pembelian.PUT("/:id/status", middlewares.AuthMiddleware(), middlewares.AdminOnly(), controllers.UpdatePembelianStatus)
		}

	}
}
