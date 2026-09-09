package service

import (
	"errors"
	"unicode"
)

// ValidateRegisterInput memastikan input dasar tidak kosong dan memanggil aturan password
func ValidateRegisterInput(username, email, password string) error {
	if username == "" || email == "" {
		return errors.New("username dan email tidak boleh kosong")
	}
	return ValidatePasswordStrength(password)
}

// ValidatePasswordStrength adalah pure function untuk mengecek kekuatan password
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("password minimal harus 8 karakter")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("password harus mengandung huruf besar, huruf kecil, dan angka")
	}

	return nil
}