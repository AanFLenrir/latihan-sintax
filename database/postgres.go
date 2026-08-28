package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	
	// Sesuaikan "latihan-sintaks" ini dengan module di go.mod Anda
	"latihan-sintaks/config"
)

func ConnectPostgres() *pgxpool.Pool {
	// Merakit Data Source Name (DSN) dari environment
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s pool_max_conns=%s",
		config.GetEnv("DB_HOST", "127.0.0.1"),
		config.GetEnv("DB_PORT", "5432"),
		config.GetEnv("DB_USER", "postgres"),
		config.GetEnv("DB_PASSWORD", ""), // pastikan mengisi password di file .env
		config.GetEnv("DB_NAME", "praktikum_backend"),
		config.GetEnv("DB_SSLMODE", "disable"),
		config.GetEnv("DB_MAX_CONNS", "10"),
	)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Gagal membaca konfigurasi database: %v", err)
	}

	// Memberikan batas waktu 5 detik untuk mencoba terhubung
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("Gagal membuat koneksi ke PostgreSQL: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("PostgreSQL tidak merespons (Ping gagal): %v", err)
	}

	fmt.Println("Koneksi ke PostgreSQL berhasil!")
	return pool
}