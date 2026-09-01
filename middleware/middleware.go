package middleware

import (
	"os"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"latihan-sintaks/config"
)

// Setup memasang semua penengah (middleware) ke dalam aplikasi.
func Setup(app *fiber.App, logFile *os.File) {
	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(requestid.New())

	// Mencatat log ke file berformat JSON
	app.Use(logger.New(config.LoggerConfig(logFile)))

	// Mencatat log ke terminal/layar (agar gampang dipantau)
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${locals:requestid} ${status} - ${latency} ${method} ${path}\n",
	}))
}