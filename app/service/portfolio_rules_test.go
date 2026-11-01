package service_test

import (
	"reflect"
	"testing"

	"latihan-fiber/app/service"
)

func TestNormalizeTechnologies(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "clean array",
			input:    []string{"Go", "PostgreSQL", "MongoDB"},
			expected: []string{"Go", "PostgreSQL", "MongoDB"},
		},
		{
			name:     "trims spaces and removes empty",
			input:    []string{"  Go  ", "", "  ", "PostgreSQL"},
			expected: []string{"Go", "PostgreSQL"},
		},
		{
			name:     "deduplicates case-insensitively",
			input:    []string{"Go", "go", "GO", "Docker", "docker"},
			expected: []string{"Go", "Docker"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := service.NormalizeTechnologies(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("NormalizeTechnologies(%v) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestCanAddProject(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected bool
	}{
		{"zero projects", 0, true},
		{"19 projects", 19, true},
		{"20 projects (limit reached)", 20, false},
		{"21 projects (exceeded)", 21, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := service.CanAddProject(tc.count)
			if got != tc.expected {
				t.Errorf("CanAddProject(%d) = %v, want %v", tc.count, got, tc.expected)
			}
		})
	}
}
