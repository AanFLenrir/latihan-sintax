package helper

import (
	"github.com/gofiber/fiber/v2"
	"latihan-sintaks/app/model"
)

// CurrentUser digunakan oleh student_service.go untuk mengambil data user terautentikasi
func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	// Cek dari locals "userInfo" atau "user"
	userVal := c.Locals("userInfo")
	if userVal == nil {
		userVal = c.Locals("user")
	}

	if userVal == nil {
		return model.AuthUser{}, false
	}

	// Jika yang tersimpan bertipe *Claims
	if claims, ok := userVal.(*Claims); ok {
		return model.AuthUser{
			UserID:   0, // Disesuaikan jika claims memuat ID
			Username: claims.Username,
			Role:     claims.Role,
		}, true
	}

	// Jika yang tersimpan sudah berupa model.AuthUser
	if user, ok := userVal.(model.AuthUser); ok {
		return user, true
	}

	return model.AuthUser{}, false
}

// GetUserClaims digunakan oleh auth_route.go untuk mengambil klaim JWT langsung
func GetUserClaims(c *fiber.Ctx) (*Claims, bool) {
	userClaims, ok := c.Locals("userInfo").(*Claims)
	if !ok {
		// Fallback jika disimpan dengan kunci "user"
		if altClaims, altOk := c.Locals("user").(*Claims); altOk {
			return altClaims, true
		}
		return nil, false
	}
	return userClaims, true
}