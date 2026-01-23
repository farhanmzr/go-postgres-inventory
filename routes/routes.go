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

			// Permintaan
			adminAuth.GET("/permintaan", controllers.AdminGetAllPermintaan)
			adminAuth.DELETE("/permintaan/:id", controllers.DeletePermintaan)

			adminPemakaian := adminAuth.Group("/pemakaian")
			{
				adminPemakaian.GET("/:id", controllers.UsageDetail)              // detail header+items
				adminPemakaian.POST("/item/decide", controllers.UsageItemDecide) // approve/reject per item
				adminPemakaian.GET("/", controllers.AdminGetAllPemakaian)
			}

			customer := adminAuth.Group("/customer")
			{
				customer.GET("/", controllers.GetAllCustomer)
				customer.GET("/:id", controllers.GetCustomerByID)
				customer.POST("/", controllers.CreateCustomer)
				customer.PUT("/:id", controllers.UpdateCustomer)
				customer.DELETE("/:id", controllers.DeleteCustomer)
			}

			barang := adminAuth.Group("/barang")
			{
				barang.GET("/", controllers.GetAllBarang)
				barang.GET("/:id", controllers.GetBarangByID)
				barang.POST("/", controllers.CreateBarang)
				barang.PUT("/:id", controllers.UpdateBarang)
				barang.DELETE("/:id", controllers.DeleteBarang)
			}

			gudangBarang := adminAuth.Group("/gudang-barang")
			{
				gudangBarang.GET("/:id", controllers.GetGudangBarangByID)
				gudangBarang.PUT("/:id/stok", controllers.UpdateStokBarang)
				gudangBarang.GET("/:id/historyStok", controllers.GetStockHistoryByBarang)
				gudangBarang.PUT("/:id", controllers.UpdateGudangBarang)
				gudangBarang.DELETE("/:id", controllers.DeleteGudangBarang)
			}

			gudang := adminAuth.Group("/gudang")
			{
				gudang.GET("/", controllers.GetAllGudang)
				gudang.GET("/:id", controllers.GetGudangByID)
				gudang.POST("/", controllers.CreateGudang)
				gudang.PUT("/:id", controllers.UpdateGudang)
				gudang.DELETE("/:id", controllers.DeleteGudang)
				//barang
				gudang.GET("/:id/barang", controllers.GetGudangBarangList)
				gudang.POST("/:id/barang", controllers.TambahBarangKeGudang)
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
				pembelian.GET("/", controllers.PurchaseReqList)
				pembelian.GET("/invoice/:id", controllers.PurchaseInvoiceDetail)
				pembelian.DELETE("/:id", controllers.DeletePembelianAdmin)
			}
			penjualan := adminAuth.Group("/penjualan")
			{
				penjualan.GET("/", controllers.SalesReqAdminList)
				penjualan.POST("/:id/approve", controllers.SalesReqApprove)
				penjualan.POST("/:id/reject", controllers.SalesReqReject)
				penjualan.GET("/invoice/:id", controllers.SalesInvoiceDetail)
				penjualan.DELETE("/:id", controllers.DeletePenjualanAdmin)
			}
			reports := adminAuth.Group("/reports")
			{
				reports.GET("/barang", controllers.ReportBarang)
				reports.GET("/stock/grup/:id", controllers.ReportStockPerGrup)
				reports.GET("/stock/gudang/:id", controllers.ReportStockPerGudang)
				reports.GET("/purchases", controllers.ReportPurchasesAdmin)
				reports.GET("/sales", controllers.ReportSalesAdmin)
				reports.GET("/usage", controllers.ReportUsageAdmin)
				reports.GET("/permintaan", controllers.ReportPermintaanAdmin)
				reports.GET("/profit/barang", controllers.ReportProfitPerBarangAdmin)
			}
			piutangAdmin := adminAuth.Group("/piutang")
			{
				piutangAdmin.GET("/", controllers.PiutangListAdmin)
				piutangAdmin.GET("/:id/history", controllers.PiutangReceiptHistory)
			}

			hutangAdmin := adminAuth.Group("/hutang")
			{
				hutangAdmin.GET("/", controllers.HutangListAdmin)
				hutangAdmin.GET("/:id/history", controllers.HutangPaymentHistoryAdmin)
			}

			wallet := adminAuth.Group("/wallet")
			{
				// gudang wallets
				wallet.POST("/gudang/:gudang_id/cash", controllers.CreateCashWallet)
				wallet.POST("/gudang/:gudang_id/bank", controllers.CreateBankWallet)
				wallet.GET("/gudang/:gudang_id", controllers.ListWalletsByGudang)
				// mutasi
				wallet.GET("/:wallet_id/tx", controllers.ListWalletTransactions)
				wallet.POST("/:wallet_id/income", controllers.WalletManualIncome)
				wallet.POST("/:wallet_id/expense", controllers.WalletManualExpense)
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

				// contoh proteksi:
				// userAuth.GET("/purchase", middlewares.RequirePerm("PURCHASE"), controllers.PurchaseList)
				// userAuth.GET("/sales", middlewares.RequirePerm("SALES"), controllers.SalesList)
				// userAuth.GET("/reports", middlewares.RequirePerm("REPORT_VIEW"), controllers.ReportList)
				pemakaian := userAuth.Group("/pemakaian", middlewares.RequirePerm("CONSUMPTION"))
				{
					pemakaian.GET("/", controllers.UsageMyList)
					pemakaian.POST("/", controllers.UsageCreate)
				}

				permintaan := userAuth.Group("/permintaan", middlewares.RequirePerm("PERMINTAAN"))
				{
					permintaan.GET("/", controllers.GetMyPermintaan)
					permintaan.POST("/", controllers.CreatePermintaan)
				}
				penjualan := userAuth.Group("/penjualan", middlewares.RequirePerm("SALES"))
				{
					penjualan.GET("/", controllers.SalesReqUserList)
					penjualan.POST("/", controllers.CreatePenjualan)
					penjualan.GET("/invoice/:id", controllers.SalesInvoiceDetail)
					penjualan.DELETE("/:id", middlewares.RequirePerm("DELETE_PENJUALAN"), controllers.DeletePenjualanUser)
				}
				pembelian := userAuth.Group("/pembelian", middlewares.RequirePerm("PURCHASE"))
				{
					pembelian.GET("/", controllers.PurchaseReqMyList)
					pembelian.POST("/", controllers.CreatePembelian)
					pembelian.GET("/invoice/:id", controllers.PurchaseInvoiceDetail)
					pembelian.DELETE("/:id", middlewares.RequirePerm("DELETE_PEMBELIAN"), controllers.DeletePembelianUser)
				}
				customer := userAuth.Group("/customer")
				{
					customer.GET("/", controllers.GetAllCustomer)
					customer.GET("/:id", controllers.GetCustomerByID)
					customer.POST("/", middlewares.RequirePerm("CUSTOMER"), controllers.CreateCustomer)
					// barang.PUT("/:id", controllers.UpdateBarang)
					// barang.DELETE("/:id", controllers.DeleteBarang)
				}
				barang := userAuth.Group("/barang")
				{
					barang.GET("/", controllers.GetAllBarang)
					barang.GET("/:id", controllers.GetBarangByID)
					barang.POST("/", middlewares.RequirePerm("CREATE_ITEM"), controllers.CreateBarang)
					// barang.PUT("/:id", controllers.UpdateBarang)
					// barang.DELETE("/:id", controllers.DeleteBarang)
				}

				gudangBarang := userAuth.Group("/gudang-barang")
				{
					gudangBarang.GET("/:id", controllers.GetGudangBarangByID)
					gudangBarang.PUT("/:id/stok", middlewares.RequirePerm("EDIT_STOCK"), controllers.UpdateStokBarang)
					gudangBarang.GET("/:id/historyStok", middlewares.RequirePerm("EDIT_STOCK"), controllers.GetStockHistoryByBarang)
					gudangBarang.PUT("/:id", controllers.UpdateGudangBarang)
					// gudangBarang.DELETE("/:id", controllers.DeleteGudangBarang)
				}

				gudang := userAuth.Group("/gudang")
				{
					gudang.GET("/", controllers.GetAllGudang)
					gudang.GET("/:id", controllers.GetGudangByID)
					gudang.POST("/", middlewares.RequirePerm("CREATE_GUDANG"), controllers.CreateGudang)
					// gudang.PUT("/:id", controllers.UpdateGudang)
					// gudang.DELETE("/:id", controllers.DeleteGudang)
					//barang
					gudang.GET("/:id/barang", controllers.GetGudangBarangList)
					gudang.POST("/:id/barang", middlewares.RequirePerm("CREATE_ITEM"), controllers.TambahBarangKeGudang)
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

				reports := userAuth.Group("/reports", middlewares.RequirePerm("REPORT_STOCK_VIEW"))
				{
					reports.GET("/barang", controllers.ReportBarang)
					reports.GET("/stock/grup/:id", controllers.ReportStockPerGrup)
					reports.GET("/stock/gudang/:id", controllers.ReportStockPerGudang)
					reports.GET("/purchases", controllers.ReportPurchasesUser)
					reports.GET("/sales", controllers.ReportSalesUser)
					reports.GET("/usage", controllers.ReportUsageUser)
					reports.GET("/permintaan", controllers.ReportPermintaanUser)
					reports.GET("/profit/barang", controllers.ReportProfitPerBarangUser)
				}
				piutangUser := userAuth.Group("/piutang")
				{
					piutangUser.GET("/", controllers.PiutangListUser)
					piutangUser.POST("/:id/receive", controllers.PiutangReceive)
					piutangUser.GET("/:id/history", controllers.PiutangReceiptHistory)
				}
				hutangUser := userAuth.Group("/hutang")
				{
					hutangUser.GET("/", controllers.HutangListUser)
					hutangUser.POST("/:id/pay", controllers.HutangPay)
					hutangUser.GET("/:id/history", controllers.HutangPaymentHistory)
				}
				wallet := userAuth.Group("/wallet")
				{
					// gudang wallets
					wallet.POST("/gudang/:gudang_id/cash", middlewares.RequirePerm("ADD_WALLET"), controllers.CreateCashWallet)
					wallet.POST("/gudang/:gudang_id/bank", middlewares.RequirePerm("ADD_WALLET"), controllers.CreateBankWallet)
					wallet.GET("/gudang/:gudang_id", controllers.ListWalletsByGudang)
					// mutasi
					wallet.GET("/:wallet_id/tx", middlewares.RequirePerm("TRANSACTION_WALLET"), controllers.ListWalletTransactions)
					wallet.POST("/:wallet_id/income", middlewares.RequirePerm("TRANSACTION_WALLET"), controllers.WalletManualIncome)
					wallet.POST("/:wallet_id/expense", middlewares.RequirePerm("TRANSACTION_WALLET"), controllers.WalletManualExpense)
				}
			}
		}

	}
}
