package config

import (
	"github.com/gofiber/fiber/v2"
	"latihan-sintaks/helper"
)

// NewFiberApp membuat instance Fiber dengan custom error handler.
func NewFiberApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return helper.Fail(c, code, err.Error())
		},
	})
}

// HandleNotFound adalah penampung terakhir jika URL tidak ada.
func HandleNotFound(app *fiber.App) {
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "rute tidak ditemukan")
	})
}