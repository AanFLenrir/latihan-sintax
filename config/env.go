package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv memuat variabel dari berkas .env ke dalam sistem
func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("Peringatan: Berkas .env tidak ditemukan, membaca langsung dari OS")
	}
}

// GetEnv mengambil nilai environment atau fallback jika tidak ada
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}