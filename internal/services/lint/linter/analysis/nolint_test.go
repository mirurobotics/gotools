package analysis

import (
	"go/parser"
	"go/token"
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

func TestHasNolintNear(t *testing.T) {
	src := "//nolint:filelen\n\npackage foo //nolint:funclen\n\n//nolint:filelen\nvar x = 1\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		line int
		rule string
		want bool
	}{
		{1, "filelen", true},
		{2, "filelen", true},
		{3, "filelen", false},
		{3, "funclen", true},
		{6, "filelen", true},
		{7, "filelen", false},
	}
	for _, tt := range tests {
		if got := HasNolintNear(fset, f, tt.line, tt.rule); got != tt.want {
			t.Errorf("HasNolintNear(line %d, %s) = %v, want %v",
				tt.line, tt.rule, got, tt.want)
		}
	}
}
