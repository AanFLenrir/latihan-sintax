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
	"latihan-sintaks/helper"
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
	defer db.Close()

	// 3. Inisialisasi RoleRepository dan memuat PermissionSet (Wajib untuk Modul 6 RBAC)
	roleRepo := repository.NewRoleRepository(db)
	rawPermissions, err := roleRepo.LoadPermissions(context.Background())
	if err != nil {
		log.Fatalf("gagal memuat permission: %v", err)
	}
	permissions := helper.NewPermissionSet(rawPermissions)

	// 4. Inisialisasi Repository dan Service (Student & Auth)
	studentRepo := repository.NewStudentRepository(db)
	studentService := service.NewStudentService(studentRepo, permissions)

	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	
	// AuthService menggunakan 2 parameter sesuai konstruktor aslimu
	authService := service.NewAuthService(userRepo, tokenRepo)

	// 5. Inisialisasi Logger
	logFile := config.SetupLogger()
	defer logFile.Close()

	// 6. Inisialisasi Fiber App
	app := config.NewFiberApp()

	// 7. Pasang Middleware dan Rute
	middleware.Setup(app, logFile)
	
	// Daftarkan Rute Auth & Rute Student
	route.SetupAuthRoutes(app, authService)
	route.Setup(app, studentService)

	config.HandleNotFound(app)

	// 8. Jalankan Server dengan Graceful Shutdown
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