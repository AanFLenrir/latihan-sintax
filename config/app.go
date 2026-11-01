package config

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model" // Pastikan import model ini ada untuk ErrorResponse
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
	"latihan-fiber/route"
)

func NewApp(logger *slog.Logger, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "Praktikum Backend Lanjut"),
		ErrorHandler: newErrorHandler(logger),
		// Membatasi ukuran body mencegah satu request besar menghabiskan memori server
		BodyLimit: 1 * 1024 * 1024, // 1 MB
	})

	// Panggil middleware.Register dengan allowedOrigins dari .env
	middleware.Register(app, logger, GetEnv("ALLOWED_ORIGINS", ""))

	route.Register(app, deps)

	// TAMBAHAN LANGKAH 2: Ubah pula penangan route yang tidak dikenal agar ikut
	// melewati jalur AppError yang sama
	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})

	return app
}

// newErrorHandler adalah SATU-SATUNYA tempat error berubah menjadi
// response HTTP di seluruh aplikasi.
func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// PERBAIKAN 1: Tambah tanda ':=' (Ditolak Compiler di modul aslinya)
		requestID, _ := c.Locals("requestid").(string)

		var appErr *helper.AppError
		switch {
		case errors.As(err, &appErr):
			// Kegagalan yang sudah kita rencanakan.
		case errors.Is(err, fiber.ErrRequestEntityTooLarge):
			// PERBAIKAN 2: Tambah tanda '='
			appErr = &helper.AppError{
				Status:  fiber.StatusRequestEntityTooLarge,
				Code:    "PAYLOAD_TOO_LARGE",
				Message: "ukuran body melebihi batas yang diizinkan",
			}
		default:
			// Kegagalan yang tidak kita duga.
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				// PERBAIKAN 3: Tambah tanda '='
				appErr = &helper.AppError{
					Status:  fiberErr.Code,
					Code:    "HTTP_ERROR",
					Message: fiberErr.Message,
				}
			} else {
				// PERBAIKAN 4: Tambah tanda '='
				appErr = helper.Internal(err)
			}
		}

		// PERBAIKAN 5 (Tingkat log tertukar): 4xx adalah kesalahan klien (WARN), 5xx adalah kesalahan server (ERROR).
		if appErr.Status < fiber.StatusInternalServerError {
			logger.Warn("request_rejected",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status))
		} else {
			errMsg := appErr.Message
			if appErr.Unwrap() != nil {
				errMsg = appErr.Unwrap().Error()
			}
			logger.Error("request_failed",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status),
				slog.String("error", errMsg))
		}

		return c.Status(appErr.Status).JSON(model.ErrorResponse{
			Success:   false,
			Code:      appErr.Code,
			Message:   appErr.Message,
			Fields:    appErr.Fields,
			RequestID: requestID,
		})
	}
}
