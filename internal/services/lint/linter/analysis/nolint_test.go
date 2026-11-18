package analysis

import (
	"testing"
)

func TestNolintsRule(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"//nolint:filelen", true},
		{"//nolint:filelen // reason", true},
		{"//nolint:gosec,filelen // reason", true},
		{"//miru:nolint:filelen", true},
		{"//nolint:filelenx", false},
		{"//nolint:funclen", false},
		{"//nolint:gosec, filelen // reason", true},
		{"//nolint:filelen,gosec // reason", true},
		{"//nolint:filelen , gosec", true},
		{"//nolint:filelen// reason", true},
		{"// nolint:filelen", true},
		{"// remove nolint:filelen once split", false},
		{"// filelen", false},
	}
	for _, tt := range tests {
		if got := NolintsRule(tt.text, "filelen"); got != tt.want {
			t.Errorf("NolintsRule(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}
