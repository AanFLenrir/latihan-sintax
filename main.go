package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"latihan-sintaks/app/repository"
	"latihan-sintaks/app/service"
	"latihan-sintaks/config"
	"latihan-sintaks/database"
	"latihan-sintaks/middleware"
	"latihan-sintaks/route"
)

func main() {
	// 1. Muat variabel environment
	if err := godotenv.Load(); err != nil {
		log.Println("Peringatan: file .env tidak ditemukan, menggunakan variabel sistem")
	}

	// 2. Hubungkan ke Database
	db := database.ConnectPostgres()
	defer db.Close() // <-- Sudah diperbaiki: pgxpool.Close() tidak menerima context

	// 3. Inisialisasi Repository dan Service
	studentRepo := repository.NewStudentRepository(db)
	studentService := service.NewStudentService(studentRepo)

	// 4. Inisialisasi Logger
	logFile := config.SetupLogger()
	defer logFile.Close()

	// 5. Inisialisasi Fiber App
	app := config.NewFiberApp()

	// 6. Pasang Middleware dan Rute
	middleware.Setup(app, logFile)
	route.Setup(app, studentService)
	config.HandleNotFound(app)

	// 7. Jalankan Server dengan Graceful Shutdown
	go func() {
		if err := app.Listen(":3000"); err != nil {
			log.Panic(err)
		}
	}()

	// Menunggu sinyal interupsi (Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Mematikan server...")
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Shutdown(); err != nil {
		log.Fatal("Server dipaksa mati:", err)
	}
	log.Println("Server berhasil dimatikan")
}