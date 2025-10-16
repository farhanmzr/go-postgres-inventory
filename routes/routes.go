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

		gudang := api.Group("/gudang", middlewares.AuthMiddleware())
		{
			gudang.GET("/", controllers.GetAllGudang)
			gudang.GET("/:id", controllers.GetGudangByID)

			// Hanya admin yang boleh CRUD
			gudang.POST("/", middlewares.AdminOnly(), controllers.CreateGudang)
			gudang.PUT("/:id", middlewares.AdminOnly(), controllers.UpdateGudang)
			gudang.DELETE("/:id", middlewares.AdminOnly(), controllers.DeleteGudang)
		}

		grupBarang := api.Group("/grupbarang", middlewares.AuthMiddleware())
		{
			grupBarang.GET("/", controllers.GetAllGrupBarang)
			grupBarang.GET("/:id", controllers.GetGrupBarangByID)

			// Hanya admin yang boleh CRUD
			grupBarang.POST("/", middlewares.AdminOnly(), controllers.CreateGrupBarang)
			grupBarang.PUT("/:id", middlewares.AdminOnly(), controllers.UpdateGrupBarang)
			grupBarang.DELETE("/:id", middlewares.AdminOnly(), controllers.DeleteGrupBarang)
		}

		supplier := api.Group("/supplier", middlewares.AuthMiddleware())
		{
			supplier.GET("/", controllers.GetAllSupplier)
			supplier.GET("/:id", controllers.GetSupplierByID)

			// Hanya admin yang boleh CRUD
			supplier.POST("/", middlewares.AdminOnly(), controllers.CreateSupplier)
			supplier.PUT("/:id", middlewares.AdminOnly(), controllers.UpdateSupplier)
			supplier.DELETE("/:id", middlewares.AdminOnly(), controllers.DeleteSupplier)
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
