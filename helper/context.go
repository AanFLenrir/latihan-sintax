package helper

import (
	"github.com/gofiber/fiber/v2"
)

func GetUserClaims(c *fiber.Ctx) (*Claims, bool) {
	// Pastikan middleware milikmu menyimpan data dengan nama "userInfo"
	userClaims, ok := c.Locals("userInfo").(*Claims)
	if !ok {
		return nil, false
	}
	return userClaims, true
}