package route

import (
	"github.com/gofiber/fiber/v2"
	"latihan-sintaks/app/model"
	"latihan-sintaks/app/service"
	"latihan-sintaks/helper"
	"latihan-sintaks/middleware"
)

func SetupAuthRoutes(app *fiber.App, authService service.AuthService) {
	api := app.Group("/api/v1/auth")

	// 1. Register: Validasi, hashing, role ditentukan server (di service), return 201
	api.Post("/register", func(c *fiber.Ctx) error {
		var req model.RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "format request tidak valid"})
		}
		
		// Validasi password murni tanpa fiber.Ctx
		if err := service.ValidatePasswordStrength(req.Password); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
		}

		user, err := authService.Register(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "registrasi berhasil", "data": user})
	})

	// 2. Login: Dilindungi LoginRateLimiter untuk mencegah brute force (Nomor 5)
	api.Post("/login", middleware.LoginRateLimiter(), func(c *fiber.Ctx) error {
		var req model.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "format request tidak valid"})
		}

		tokenRes, err := authService.Login(c.Context(), req)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": err.Error()}) // 401 Unauthorized
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "login berhasil", "data": tokenRes})
	})

	// 3. Refresh: Rotasi token, 401 bila token tidak aktif
	api.Post("/refresh", func(c *fiber.Ctx) error {
		type RefreshReq struct {
			RefreshToken string `json:"refresh_token"`
		}
		var req RefreshReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "format request tidak valid"})
		}

		tokenRes, err := authService.Refresh(c.Context(), req.RefreshToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": err.Error()}) // 401 Unauthorized
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "token berhasil diperbarui", "data": tokenRes})
	})

	// Grup rute yang di-protect (wajib bawa token)
	protected := api.Group("/", middleware.RequireAuth())

	// 4. Logout: Mencabut refresh token
	protected.Post("/logout", func(c *fiber.Ctx) error {
		type LogoutReq struct {
			RefreshToken string `json:"refresh_token"`
		}
		var req LogoutReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "format request tidak valid"})
		}

		err := authService.Logout(c.Context(), req.RefreshToken)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "berhasil logout, token dicabut"})
	})

	// 5. Me: Profil pemakai yang sedang login
	protected.Get("/me", func(c *fiber.Ctx) error {
		claims, ok := helper.GetUserClaims(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "akses ditolak"})
		}

		user, err := authService.GetMe(c.Context(), claims.Username)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"data": user})
	})
}