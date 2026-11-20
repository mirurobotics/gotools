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
		{"//export F", true},
		{"//extern f", true},
		{"//line foo.go:10", true},
		{"// explain the code", false},
		{"// Note: a colon in prose", false},
		{"/* block */", false},
	}
	for _, tc := range tests {
		if got := IsDirective(tc.text); got != tc.want {
			t.Errorf("IsDirective(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}
