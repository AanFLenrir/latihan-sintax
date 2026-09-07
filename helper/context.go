package helper

import (
	"github.com/gofiber/fiber/v2"
)

// GetUserClaims mengambil data user yang sedang login dari Fiber context
func GetUserClaims(c *fiber.Ctx) (*Claims, bool) {
	userClaims, ok := c.Locals("userInfo").(*Claims)
	if !ok {
		return nil, false
	}
	return userClaims, true
}