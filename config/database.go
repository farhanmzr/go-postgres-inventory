package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	// 1️⃣ Coba ambil DB_URL dari environment (Render)
	dbURL := os.Getenv("DB_URL")

	// 2️⃣ Kalau kosong, berarti sedang dijalankan di lokal
	if dbURL == "" {
		_ = godotenv.Load() // optional, kalau kamu pakai file .env lokal

		host := "localhost"
		user := "postgres"
		password := "12345"
		dbname := "inventory"
		port := "5432"

		dbURL = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			host, user, password, dbname, port)
	}

	// 3️⃣ Coba konek ke database
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Gagal konek ke database:", err)
	}

	DB = db
	fmt.Println("✅ Database connected!")
}
