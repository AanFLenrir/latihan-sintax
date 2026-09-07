package service

import (
	"errors"
	"strings"
)

func ValidateRegisterInput(username, email, password string) error {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(email) == "" {
		return errors.New("username dan email tidak boleh kosong")
	}
	if len(password) < 8 {
		return errors.New("password minimal harus 8 karakter")
	}
	// Daftar sederhana contoh password lemah yang ditolak
	weakPasswords := []string{"password123", "12345678", "qwertyuiop", "rahasia"}
	for _, wp := range weakPasswords {
		if password == wp {
			return errors.New("password terlalu umum dan mudah ditebak")
		}
	}
	return nil
}