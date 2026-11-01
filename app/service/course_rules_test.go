package service_test

import (
	"testing"

	"latihan-fiber/app/service"
)

func TestWithinCreditLimit(t *testing.T) {
	tests := []struct {
		name     string
		taken    int
		adding   int
		expected bool
	}{
		{"below limit", 18, 3, true},
		{"exact limit 24", 21, 3, true},
		{"exceeds limit 25", 22, 3, false},
		{"zero taken", 0, 3, true},
		{"already 24 adding 1", 24, 1, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := service.WithinCreditLimit(tc.taken, tc.adding)
			if got != tc.expected {
				t.Errorf("WithinCreditLimit(%d, %d) = %v; want %v", tc.taken, tc.adding, got, tc.expected)
			}
		})
	}
}

func TestCreditsAfterSwap(t *testing.T) {
	tests := []struct {
		name     string
		taken    int
		leaving  int
		joining  int
		expected int
	}{
		{"swap same credits", 21, 3, 3, 21},
		{"swap higher credits", 21, 2, 4, 23},
		{"swap lower credits", 21, 4, 2, 19},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := service.CreditsAfterSwap(tc.taken, tc.leaving, tc.joining)
			if got != tc.expected {
				t.Errorf("CreditsAfterSwap(%d, %d, %d) = %d; want %d", tc.taken, tc.leaving, tc.joining, got, tc.expected)
			}
		})
	}
}
