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
		{Code: "REPORT_VIEW", Name: "Akses Menu Laporan"},
		{Code: "REPORT_STOCK_VIEW", Name: "Laporan Stok Barang"},
	}
	for _, p := range codes {
		var cnt int64
		DB.Model(&models.Permission{}).Where("code = ?", p.Code).Count(&cnt)
		if cnt == 0 { DB.Create(&p) }
	}
}
