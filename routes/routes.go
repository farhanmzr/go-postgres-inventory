package routes

import (
	"go-postgres-inventory/controllers"
	"go-postgres-inventory/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{

		// ================= ADMIN APP =================
		admin := api.Group("/admin")
		{
			admin.POST("/register", controllers.AdminRegister)
			admin.POST("/login", controllers.AdminLogin)
			

			// Semua di bawah butuh token admin
			adminAuth := admin.Group("/", middlewares.AdminAuth())

			// Manajemen user operasional
			adminAuth.GET("/users", controllers.AdminGetAllUsers)
			adminAuth.POST("/users", controllers.AdminCreateUser)
			adminAuth.PUT("/users/:userID/permissions", controllers.AdminSetUserPermissions)

			// Resource ADMIN lainnya (semua di bawah /api/admin/**)
			barang := adminAuth.Group("/barang")
			{
				barang.GET("/", controllers.GetAllBarang)
				barang.GET("/:id", controllers.GetBarangByID)
				barang.POST("/", controllers.CreateBarang)
				barang.PUT("/:id", controllers.UpdateBarang)
				barang.DELETE("/:id", controllers.DeleteBarang)
			}

			gudang := adminAuth.Group("/gudang")
			{
				gudang.GET("/", controllers.GetAllGudang)
				gudang.GET("/:id", controllers.GetGudangByID)
				gudang.POST("/", controllers.CreateGudang)
				gudang.PUT("/:id", controllers.UpdateGudang)
				gudang.DELETE("/:id", controllers.DeleteGudang)
			}

			grupBarang := adminAuth.Group("/grupbarang")
			{
				grupBarang.GET("/", controllers.GetAllGrupBarang)
				grupBarang.GET("/:id", controllers.GetGrupBarangByID)
				grupBarang.POST("/", controllers.CreateGrupBarang)
				grupBarang.PUT("/:id", controllers.UpdateGrupBarang)
				grupBarang.DELETE("/:id", controllers.DeleteGrupBarang)
			}

			supplier := adminAuth.Group("/supplier")
			{
				supplier.GET("/", controllers.GetAllSupplier)
				supplier.GET("/:id", controllers.GetSupplierByID)
				supplier.POST("/", controllers.CreateSupplier)
				supplier.PUT("/:id", controllers.UpdateSupplier)
				supplier.DELETE("/:id", controllers.DeleteSupplier)
			}
		}

		// ================= USER APP ==================
		app := api.Group("/app")
		{
			app.POST("/login", controllers.UserLogin)

			appAuth := app.Group("/", middlewares.UserAuth())
			appAuth.GET("/profile", controllers.UserProfile)

			// contoh endpoint yang butuh permission tertentu:
			// appAuth.POST("/items", middlewares.RequirePerm("CREATE_ITEM"), controllers.CreateBarangUser)
			// appAuth.POST("/stock/movements", middlewares.RequirePerm("CONSUMPTION"), controllers.CreatePemakaian)
			// appAuth.GET("/reports/stock", middlewares.RequirePerm("REPORT_STOCK_VIEW"), controllers.ReportStock)
		}


	}
}
