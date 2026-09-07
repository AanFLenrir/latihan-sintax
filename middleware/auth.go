package middleware

import (
	"strings"

	"latihan-sintaks/helper"

	"github.com/gofiber/fiber/v2"
)

func AuthProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "autentikasi diperlukan",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "format token tidak valid",
			})
		}

		tokenStr := parts[1]
		claims, err := helper.ParseToken(tokenStr)
		if err != nil {
			msg := "access token tidak valid"
			if strings.Contains(err.Error(), "expired") {
				msg = "access token kedaluwarsa"
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": msg,
			})
		}

		// Simpan informasi user ke context Fiber agar bisa diakses di handler
		c.Locals("userInfo", claims)
		return c.Next()
	}
}