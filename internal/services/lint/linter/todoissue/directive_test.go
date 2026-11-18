package todoissue

import "testing"

func TestIsDirective(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"//go:build linux", true},
		{"//lint:ignore SA1000 reason", true},
		{"//export Foo", true},
		{"//extern foo", true},
		{"//line foo.go:1", true},
		{"//nolint:gosec // reason", true},
		{"// #nosec G304 -- reason", true},
		{"// +build linux", true},
		{"/* go:build */", false},
		{"// plain comment", false},
		{"//Note: prose", false},
		{"//go:", false},
		{"//:value", false},
		{"//go: build", false},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := isDirective(tt.text); got != tt.want {
				t.Errorf("isDirective(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}
