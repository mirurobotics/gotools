package analysis

import "testing"

func TestIsDirective(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"//go:build linux", true},
		{"//nolint:funclen // reason", true},
		{"//nolint", true},
		{"// nolint:gosec", true},
		{"// #nosec G304 -- reason", true},
		{"//lint:ignore SA1000 reason", true},
		{"// +build linux", true},
		{"// explain the code", false},
		{"// Note: a colon in prose", false},
		{"/* block */", false},
	}
	for _, tt := range tests {
		if got := IsDirective(tt.text); got != tt.want {
			t.Errorf("IsDirective(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}
