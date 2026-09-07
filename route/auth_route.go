package route

import (
	"latihan-sintaks/app/model"
	"latihan-sintaks/app/service"

	"github.com/gofiber/fiber/v2"
)

func SetupAuthRoutes(app *fiber.App, authService service.AuthService) {
	api := app.Group("/api/v1/auth")

	// Endpoint Register
	api.Post("/register", func(c *fiber.Ctx) error {
		var req model.RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "format request tidak valid",
			})
		}

		user, err := authService.Register(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"message": "registrasi berhasil",
			"data":    user,
		})
	})

	// Endpoint Login
	api.Post("/login", func(c *fiber.Ctx) error {
		var req model.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "format request tidak valid",
			})
		}

		tokenRes, err := authService.Login(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "login berhasil",
			"data":    tokenRes,
		})
	})
}