package config

import "go-postgres-inventory/models"

func SeedPermissions() {
	codes := []models.Permission{
		{Code: "PURCHASE", Name: "Pembelian"},
		{Code: "SALES", Name: "Penjualan"},
		{Code: "CONSUMPTION", Name: "Pemakaian"},
		{Code: "CREATE_ITEM", Name: "Tambah Barang Baru"},
		{Code: "ACCESS_LOCATIONS", Name: "Akses Lokasi Barang & Susunan"},
		{Code: "CREATE_ITEM_GROUP", Name: "Tambah Grup Barang"},
		{Code: "CREATE_GUDANG", Name: "Tambah Gudang"},
		{Code: "CREATE_SUPPLIER", Name: "Tambah Supplier"},
		{Code: "REPORT_VIEW", Name: "Akses Menu Laporan"},
		{Code: "REPORT_STOCK_VIEW", Name: "Laporan Stok Barang"},
		//

		{Code: "HARGA_BELI_JUAL", Name: "Manage Harga Beli dan Jual"},
		{Code: "PERMINTAAN", Name: "Permintaan"},
		{Code: "CUSTOMER", Name: "Customer"},
		{Code: "EDIT_STOCK", Name: "Manage Stock"},
		{Code: "APPROVE_REJECT_PEMAKAIAN", Name: "Approve & Reject Pemakaian"},
		{Code: "APPROVE_REJECT_PENJUALAN", Name: "Approve & Reject Penjualan"},

		//EDIT
		{Code: "EDIT_PEMAKAIAN", Name: "Edit Pemakaian"},

		//DELETE
		{Code: "DELETE_PEMBELIAN", Name: "Hapus Pembelian"},
		{Code: "DELETE_PENJUALAN", Name: "Hapus Penjualan"},
		{Code: "DELETE_PEMAKAIAN", Name: "Hapus Pemakaian"},
		{Code: "DELETE_PERMINTAAN", Name: "Hapus Permintaan"},
		
	}
	for _, p := range codes {
		var cnt int64
		DB.Model(&models.Permission{}).Where("code = ?", p.Code).Count(&cnt)
		if cnt == 0 { DB.Create(&p) }
	}
}
