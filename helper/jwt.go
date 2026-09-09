package helper

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims dipusatkan di sini
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Digunakan oleh auth_service.go
func GenerateAccessToken(username, role string) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	claims := Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			Issuer:    os.Getenv("JWT_ISSUER"),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// Digunakan oleh auth_service.go
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Digunakan oleh middleware/auth.go
func ParseToken(tokenStr string) (*Claims, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	claims := &Claims{}
	
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	
	return claims, nil
}