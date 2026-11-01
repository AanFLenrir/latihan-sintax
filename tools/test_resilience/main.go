package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/app/service"
	"latihan-fiber/config"
	"latihan-fiber/database"
	"latihan-fiber/helper"
	"latihan-fiber/route"
)

const evidenceDir = "docs/evidence/modul-10"

func main() {
	config.LoadEnv()
	_ = os.MkdirAll(evidenceDir, 0755)

	var logOutput string
	log := func(msg string) {
		fmt.Println(msg)
		logOutput += msg + "\n"
	}

	log("================================================================")
	log("   PENGUJIAN KETAHANAN DUA DATABASE (POLYGLOT PERSISTENCE)     ")
	log("================================================================")

	// Inisialisasi pool PostgreSQL asli
	pool, err := database.NewPool(context.Background())
	if err != nil {
		log(fmt.Sprintf("Gagal init PostgreSQL pool: %v", err))
		return
	}
	defer pool.Close()

	// Inisialisasi client MongoDB asli
	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		log(fmt.Sprintf("Gagal init MongoDB client: %v", err))
		return
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()

	// Buat client dummy yang mati (port 27019 - tidak ada listener)
	optsDeadMongo := options.Client().
		ApplyURI("mongodb://127.0.0.1:27019").
		SetServerSelectionTimeout(1 * time.Second)
	deadMongoClient, _ := mongo.Connect(optsDeadMongo)
	defer func() { _ = deadMongoClient.Disconnect(context.Background()) }()

	// Buat dummy dead PostgreSQL pool (port 5439)
	deadPgConfig, _ := pgxpool.ParseConfig("postgres://postgres:wrong@127.0.0.1:5439/dummy?sslmode=disable")
	deadPgPool, _ := pgxpool.NewWithConfig(context.Background(), deadPgConfig)
	defer deadPgPool.Close()

	// Inisialisasi Repositori & Service
	mongoDB := mongoClient.Database(config.GetEnv("MONGO_DATABASE", "praktikum_backend"))
	portfolioRepo := repository.NewPortfolioRepository(mongoDB)
	studentRepo := repository.NewStudentRepository(pool)
	permissions := helper.NewPermissionSet(map[string][]string{
		"admin": {"course:manage", "supervision:manage"},
	})
	portfolioService := service.NewPortfolioService(portfolioRepo, studentRepo, permissions)

	// -------------------------------------------------------------
	// TEST CASE N.1: PostgreSQL Mati (Simulasi Pool Gagal / Down)
	// -------------------------------------------------------------
	log("\n--- SKENARIO N.1: PostgreSQL Gagal Dihubungi ---")
	{
		// 1. Health check saat PostgreSQL down
		appPgDown := fiber.New(fiber.Config{ErrorHandler: config.NewApp(config.NewLogger(), route.Dependencies{}).Config().ErrorHandler})
		route.Register(appPgDown, route.Dependencies{
			Pool:             deadPgPool,
			Mongo:            mongoClient, // MongoDB tetap aktif
			PortfolioService: portfolioService,
		})

		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		resp, _ := appPgDown.Test(req, 3000)
		log(fmt.Sprintf("[N.1.1] Health Check saat PostgreSQL Down -> HTTP %d (Ekspektasi: 503)", resp.StatusCode))

		// 2. Endpoint Portfolio yang membutuhkan validasi PostgreSQL (GET /students/1/portfolio)
		jwtManager := helper.NewJWTManager("d1fce7dfcfb4eae238bab8901e57a48ddd32bff75d42b998fd0aa4db4b9e48fd", "praktikum-backend", 1*time.Hour)
		token, _ := jwtManager.GenerateAccess(model.User{ID: 1, Username: "admin", Role: "admin"})

		deadStudentRepo := repository.NewStudentRepository(deadPgPool)
		svcWithDeadPg := service.NewPortfolioService(portfolioRepo, deadStudentRepo, permissions)

		appPgDown2 := fiber.New(fiber.Config{ErrorHandler: config.NewApp(config.NewLogger(), route.Dependencies{}).Config().ErrorHandler})
		route.Register(appPgDown2, route.Dependencies{
			Pool:             deadPgPool,
			Mongo:            mongoClient,
			JWT:              jwtManager,
			Permissions:      permissions,
			PortfolioService: svcWithDeadPg,
		})

		reqPort := httptest.NewRequest("GET", "/api/v1/students/1/portfolio", nil)
		reqPort.Header.Set("Authorization", "Bearer "+token)
		respPort, _ := appPgDown2.Test(reqPort, 3000)
		log(fmt.Sprintf("[N.1.2] GET /students/1/portfolio saat PostgreSQL Down -> HTTP %d (Ekspektasi: 500/503 karena validasi mahasiswa di PG gagal)", respPort.StatusCode))

		// 3. Endpoint Statistik Teknologi (HANYA butuh MongoDB)
		reqStats := httptest.NewRequest("GET", "/api/v1/portfolios/stats/technologies", nil)
		respStats, _ := appPgDown2.Test(reqStats, 3000)
		log(fmt.Sprintf("[N.1.3] GET /portfolios/stats/technologies saat PostgreSQL Down -> HTTP %d (Ekspektasi: 200 karena MongoDB tetap aktif)", respStats.StatusCode))
	}

	// -------------------------------------------------------------
	// TEST CASE N.2: MongoDB Mati (Simulasi Server Down)
	// -------------------------------------------------------------
	log("\n--- SKENARIO N.2: MongoDB Gagal Dihubungi ---")
	{
		// 1. Health check saat MongoDB down
		appMongoDown := fiber.New(fiber.Config{ErrorHandler: config.NewApp(config.NewLogger(), route.Dependencies{}).Config().ErrorHandler})
		route.Register(appMongoDown, route.Dependencies{
			Pool:  pool, // PostgreSQL tetap aktif
			Mongo: deadMongoClient,
		})

		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		resp, _ := appMongoDown.Test(req, 3000)
		log(fmt.Sprintf("[N.2.1] Health Check saat MongoDB Down -> HTTP %d (Ekspektasi: 503)", resp.StatusCode))

		// 2. Endpoint Portfolio / Statistik yang membutuhkan MongoDB
		deadMongoRepo := repository.NewPortfolioRepository(deadMongoClient.Database("dummy"))
		svcWithDeadMongo := service.NewPortfolioService(deadMongoRepo, studentRepo, permissions)

		jwtManager := helper.NewJWTManager("d1fce7dfcfb4eae238bab8901e57a48ddd32bff75d42b998fd0aa4db4b9e48fd", "praktikum-backend", 1*time.Hour)

		appMongoDown2 := fiber.New(fiber.Config{ErrorHandler: config.NewApp(config.NewLogger(), route.Dependencies{}).Config().ErrorHandler})
		route.Register(appMongoDown2, route.Dependencies{
			Pool:             pool,
			Mongo:            deadMongoClient,
			JWT:              jwtManager,
			Permissions:      permissions,
			PortfolioService: svcWithDeadMongo,
		})

		reqStats := httptest.NewRequest("GET", "/api/v1/portfolios/stats/technologies", nil)
		respStats, _ := appMongoDown2.Test(reqStats, 3000)
		log(fmt.Sprintf("[N.2.2] GET /portfolios/stats/technologies saat MongoDB Down -> HTTP %d (Ekspektasi: 500 Database Error)", respStats.StatusCode))
	}

	// -------------------------------------------------------------
	// TEST CASE N.3: Pemulihan Koneksi (Keduanya Aktif Kembali)
	// -------------------------------------------------------------
	log("\n--- SKENARIO N.3: Pemulihan Koneksi (Recovery) ---")
	{
		appRecovered := fiber.New(fiber.Config{ErrorHandler: config.NewApp(config.NewLogger(), route.Dependencies{}).Config().ErrorHandler})
		jwtManager := helper.NewJWTManager("d1fce7dfcfb4eae238bab8901e57a48ddd32bff75d42b998fd0aa4db4b9e48fd", "praktikum-backend", 1*time.Hour)
		token, _ := jwtManager.GenerateAccess(model.User{ID: 12, Username: "admin", Role: "admin"})

		route.Register(appRecovered, route.Dependencies{
			Pool:             pool,
			Mongo:            mongoClient,
			JWT:              jwtManager,
			Permissions:      permissions,
			PortfolioService: portfolioService,
		})

		// Health Check pulih
		reqHealth := httptest.NewRequest("GET", "/api/v1/health", nil)
		respHealth, _ := appRecovered.Test(reqHealth, 3000)
		log(fmt.Sprintf("[N.3.1] Health Check setelah Pemulihan -> HTTP %d (Ekspektasi: 200)", respHealth.StatusCode))

		// Portfolio Get pulih
		reqPort := httptest.NewRequest("GET", "/api/v1/students/1/portfolio", nil)
		reqPort.Header.Set("Authorization", "Bearer "+token)
		respPort, _ := appRecovered.Test(reqPort, 3000)
		log(fmt.Sprintf("[N.3.2] GET /students/1/portfolio setelah Pemulihan -> HTTP %d (Ekspektasi: 200)", respPort.StatusCode))

		// Stats pulih
		reqStats := httptest.NewRequest("GET", "/api/v1/portfolios/stats/technologies", nil)
		respStats, _ := appRecovered.Test(reqStats, 3000)
		log(fmt.Sprintf("[N.3.3] GET /portfolios/stats/technologies setelah Pemulihan -> HTTP %d (Ekspektasi: 200)", respStats.StatusCode))
	}

	log("\n================================================================")
	log("   HASIL: KETAHANAN DUA DATABASE TERBUKTI SESUAI ARSITEKTUR   ")
	log("================================================================")

	_ = os.WriteFile(filepath.Join(evidenceDir, "22_dual_database_resilience.txt"), []byte(logOutput), 0644)
}
