package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() {
	// 1) Ambil URL dari env (Render biasanya pakai DATABASE_URL)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL") // fallback kalau kamu set sendiri
	}

	// 2) Fallback lokal
	if dbURL == "" {
		host := "localhost"
		user := "postgres"
		password := "12345"
		dbname := "inventory"
		port := "5432"
		dbURL = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			host, user, password, dbname, port,
		)
	} else {
		// Render sering butuh sslmode=require; kalau belum ada, tambahkan
		if !strings.Contains(dbURL, "sslmode=") {
			sep := "?"
			if strings.Contains(dbURL, "?") {
				sep = "&"
			}
			dbURL = dbURL + sep + "sslmode=require"
		}
		// pastikan search_path public agar tabel dibuat di schema public
		if !strings.Contains(dbURL, "search_path=") {
			sep := "?"
			if strings.Contains(dbURL, "?") {
				sep = "&"
			}
			dbURL = dbURL + sep + "search_path=public"
		}
	}

	// 3) Buka koneksi dengan logger agar kelihatan errornya
	gormLogger := logger.New(
		log.New(os.Stdout, "[GORM] ", log.LstdFlags),
		logger.Config{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      logger.Warn, // bisa naikkan ke Info saat debug
			Colorful:      true,
		},
	)

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("❌ Gagal konek ke database: %v", err)
	}

	// 4) Set beberapa session (opsional tapi rapi)
	if err := db.Exec(`SET search_path TO public`).Error; err != nil {
		log.Printf("⚠️  Gagal set search_path public: %v", err)
	}
	if err := db.Exec(`SET TIME ZONE 'UTC'`).Error; err != nil {
		log.Printf("⚠️  Gagal set timezone UTC: %v", err)
	}

	// 5) Log info koneksi
	var dbName, currentUser, searchPath string
	_ = db.Raw("SELECT current_database()").Scan(&dbName)
	_ = db.Raw("SELECT current_user").Scan(&currentUser)
	_ = db.Raw("SHOW search_path").Scan(&searchPath)
	log.Printf("✅ DB connected: db=%s user=%s search_path=%s", dbName, currentUser, searchPath)

	DB = db
}
