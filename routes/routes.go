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

			// Manajemen data profile admin
			adminAuth.GET("/profile", controllers.GetDataAdminProfile)
			adminAuth.PUT("/profile", controllers.AdminUpdateProfile)
			adminAuth.PUT("/profile/password", controllers.AdminChangePassword)

			// Manajemen user operasional
			adminAuth.GET("/users", controllers.AdminGetAllUsers)
			adminAuth.POST("/users", controllers.AdminCreateUser) // gabungan
			adminAuth.PUT("/users/:userID/permissions", controllers.AdminSetUserPermissions)
			adminAuth.GET("/permissions", controllers.AdminListPermissions)

			adminAuth.GET("/permintaan", controllers.AdminGetAllPermintaan)

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
			pembelian := adminAuth.Group("/pembelian")
			{
				pembelian.GET("/pending", controllers.PurchaseReqPendingList)
				pembelian.POST("/:id/approve", controllers.PurchaseReqApprove)
				pembelian.POST("/:id/reject", controllers.PurchaseReqReject)
			}
		}

		// ================= USER (customer) APP =================
		user := api.Group("/user")
		{
			user.POST("/login", controllers.UserLogin)

			userAuth := user.Group("/", middlewares.UserAuth())
			{
				userAuth.GET("/profile", controllers.UserProfile)
				userAuth.PUT("/profile", controllers.UserUpdateProfile)
				userAuth.PUT("/profile/password", controllers.UserChangePassword)
				userAuth.GET("/permissions", controllers.GetPermissions)
				userAuth.GET("/gudang/:id/barang", controllers.BarangByGudang)

				// contoh proteksi:
				// userAuth.GET("/purchase", middlewares.RequirePerm("PURCHASE"), controllers.PurchaseList)
				// userAuth.GET("/sales", middlewares.RequirePerm("SALES"), controllers.SalesList)
				// userAuth.GET("/reports", middlewares.RequirePerm("REPORT_VIEW"), controllers.ReportList)
				permintaan := userAuth.Group("/permintaan", middlewares.RequirePerm("PERMINTAAN"))
				{
					permintaan.GET("/", controllers.CreatePermintaan)
					permintaan.POST("/", controllers.GetMyPermintaan)
				}
				pembelian := userAuth.Group("/pembelian", middlewares.RequirePerm("PURCHASE"))
				{
					pembelian.GET("/", controllers.PurchaseReqMyList)
					pembelian.POST("/", controllers.PurchaseReqCreate)
				}
				barang := userAuth.Group("/barang")
				{
					barang.GET("/", controllers.GetAllBarang)
					barang.GET("/:id", controllers.GetBarangByID)
					barang.POST("/", middlewares.RequirePerm("CREATE_ITEM"), controllers.CreateBarang)
					// barang.PUT("/:id", controllers.UpdateBarang)
					// barang.DELETE("/:id", controllers.DeleteBarang)
				}
				gudang := userAuth.Group("/gudang")
				{
					gudang.GET("/", controllers.GetAllGudang)
					gudang.GET("/:id", controllers.GetGudangByID)
					gudang.POST("/", middlewares.RequirePerm("CREATE_GUDANG"), controllers.CreateGudang)
					// gudang.PUT("/:id", controllers.UpdateGudang)
					// gudang.DELETE("/:id", controllers.DeleteGudang)
				}

				grupBarang := userAuth.Group("/grupbarang")
				{
					grupBarang.GET("/", controllers.GetAllGrupBarang)
					grupBarang.GET("/:id", controllers.GetGrupBarangByID)
					grupBarang.POST("/", middlewares.RequirePerm("CREATE_ITEM_GROUP"), controllers.CreateGrupBarang)
					// grupBarang.PUT("/:id", controllers.UpdateGrupBarang)
					// grupBarang.DELETE("/:id", controllers.DeleteGrupBarang)
				}

				supplier := userAuth.Group("/supplier")
				{
					supplier.GET("/", controllers.GetAllSupplier)
					supplier.GET("/:id", controllers.GetSupplierByID)
					supplier.POST("/", middlewares.RequirePerm("CREATE_SUPPLIER"), controllers.CreateSupplier)
					// supplier.PUT("/:id", controllers.UpdateSupplier)
					// supplier.DELETE("/:id", controllers.DeleteSupplier)
				}

			}
		}

	}
}
