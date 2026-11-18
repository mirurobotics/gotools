package docfile

import (
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	long := "// Package foo.\n" + strings.Repeat("//\n// more.\n", 3) + "package foo\n"
	spaced := "// Package foo.\n\n\npackage foo\n"
	tests := []struct {
		name     string
		filename string
		src      string
		maxLines int
		wantMsg  string
	}{
		{"long doc.go flagged", "pkg/doc.go", long, 5, "doc.go is 8 lines (limit 5)"},
		{"blank lines not counted", "doc.go", spaced, 2, ""},
		{"at limit allowed", "doc.go", long, 8, ""},
		{"other files skipped", "foo.go", long, 5, ""},
		{"disabled when limit is zero", "doc.go", long, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := Check(tt.filename, []byte(tt.src), tt.maxLines)
			if tt.wantMsg == "" {
				if len(diags) != 0 {
					t.Fatalf("got %v, want none", diags)
				}
				return
			}
			if len(diags) != 1 || diags[0].Message != tt.wantMsg || diags[0].Line != 1 {
				t.Fatalf("got %v, want one diagnostic %q at line 1", diags, tt.wantMsg)
			}
		})
	}
}
