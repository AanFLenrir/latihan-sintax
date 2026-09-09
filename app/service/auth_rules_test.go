package service

import (
	"testing"
)

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "Password Valid Sempurna",
			password: "PasswordAman123",
			wantErr:  false,
		},
		{
			name:     "Password Terlalu Pendek",
			password: "Pass1",
			wantErr:  true,
		},
		{
			name:     "Password Tanpa Angka",
			password: "PasswordTanpaAngka",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}