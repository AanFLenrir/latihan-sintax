package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"latihan-sintaks/helper"
)

func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "authorization header tidak ditemukan",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "format token harus Bearer token",
			})
		}

		tokenStr := parts[1]

		claims, err := helper.ParseToken(tokenStr)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "expired") || strings.Contains(errStr, "Token is expired") {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"message": "access token sudah kedaluwarsa",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "access token tidak valid",
			})
		}

		c.Locals("userInfo", claims)

		return c.Next()
	}
}

// LoginRateLimiter membatasi jumlah percobaan login dari satu alamat IP[cite: 1].
func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5, // Maksimal 5 kali percobaan[cite: 1]
		Expiration: 1 * time.Minute, // Dalam rentang waktu 1 menit[cite: 1]
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", "60")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"message": "terlalu banyak percobaan login, coba lagi dalam satu menit",
			})
		},
	})
}