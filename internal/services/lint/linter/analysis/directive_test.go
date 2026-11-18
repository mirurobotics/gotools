package analysis

import (
	"go/parser"
	"go/token"
	"testing"
)

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
	for _, tt := range tests {
		if got := IsDirective(tt.text); got != tt.want {
			t.Errorf("IsDirective(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestProseLines(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"line comments", "// a\n// b\n", 2},
		{"directive and blank lines dropped", "//\n// a\n//\n//nolint:x\n", 1},
		{"block comment trimmed", "/*\na\n*/\n", 1},
		{"empty block comment", "/* */\n", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			src := tt.src + "package foo\n"
			f, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			if got := ProseLines(f.Doc); got != tt.want {
				t.Errorf("ProseLines = %d, want %d", got, tt.want)
			}
		})
	}
}
