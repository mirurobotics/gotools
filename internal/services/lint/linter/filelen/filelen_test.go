package filelen

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func body(n int) string { return strings.Repeat("var _ = 1\n", n) }

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		src      string
		maxLen   int
		wantN    int
		wantLine int
		wantMsg  string
	}{
		{
			name:     "disabled when limit zero",
			filename: "",
			src:      "package foo\n" + body(100),
			maxLen:   0,
			wantN:    0,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "at exactly limit",
			filename: "",
			src:      "package foo\n" + body(4),
			maxLen:   5,
			wantN:    0,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "over limit",
			filename: "",
			src:      "package foo\n" + body(5),
			maxLen:   5,
			wantN:    1,
			wantLine: 1,
			wantMsg:  "file is 6 lines (limit 5)",
		},
		{
			name:     "blank and comment lines not counted",
			filename: "",
			src: "// Package foo does things.\npackage foo\n\n" +
				"// a comment\n \t \n" + body(1) + "\t// indented\n" +
				body(1) + "\n\n",
			maxLen:   3,
			wantN:    0,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "block comment lines not counted",
			filename: "",
			src: "/*\nLicense header\nspanning lines\n*/\n\npackage foo\n" +
				"/* inline */ var x = 1\nvar y = 2 // why\n",
			maxLen:   2,
			wantN:    1,
			wantLine: 6,
			wantMsg:  "file is 3 lines (limit 2)",
		},
		{
			name:     "comment markers inside strings counted as code",
			filename: "",
			src: "package foo\n\nconst s = `\n" +
				"// not a comment\n/* nor this */\n`\n",
			maxLen:   4,
			wantN:    1,
			wantLine: 1,
			wantMsg:  "file is 5 lines (limit 4)",
		},
		{
			name:     "CRLF block comment not counted",
			filename: "",
			src: "/*\r\nheader\r\n*/\r\n" +
				"package foo\r\nvar x = 1\r\n",
			maxLen:   1,
			wantN:    1,
			wantLine: 4,
			wantMsg:  "file is 2 lines (limit 1)",
		},
		{
			name:     "block comment opening with slash not counted",
			filename: "",
			src:      "/*/ note\nmore\n*/\npackage foo\nvar x = 1\n",
			maxLen:   1,
			wantN:    1,
			wantLine: 4,
			wantMsg:  "file is 2 lines (limit 1)",
		},
		{
			name:     "reported on package clause line",
			filename: "",
			src:      "// Package foo does things.\npackage foo\n" + body(5),
			maxLen:   5,
			wantN:    1,
			wantLine: 2,
			wantMsg:  "file is 6 lines (limit 5)",
		},
		{
			name:     "test file exempt",
			filename: "foo_test.go",
			src:      "package foo\n" + body(100),
			maxLen:   5,
			wantN:    0,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "test helper exempt",
			filename: "tests/helpers/foo.go",
			src:      "package helpers\n" + body(100),
			maxLen:   5,
			wantN:    0,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "nolint:filelen above package clause exempts file",
			filename: "foo.go",
			src:      "//nolint:filelen // big table\npackage foo\n" + body(100),
			maxLen:   5,
			wantN:    0,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "miru:nolint:filelen on package clause exempts file",
			filename: "foo.go",
			src:      "package foo //miru:nolint:filelen\n" + body(100),
			maxLen:   5,
			wantN:    0,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "filelen in a nolint list exempts file",
			filename: "foo.go",
			src:      "package foo //nolint:gosec,filelen // reason\n" + body(100),
			maxLen:   5,
			wantN:    0,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "rule name as a prefix does not exempt",
			filename: "foo.go",
			src:      "package foo //nolint:filelenx\n" + body(100),
			maxLen:   5,
			wantN:    1,
			wantLine: 1,
			wantMsg:  "file is 101 lines (limit 5)",
		},
		{
			name:     "nolint higher in package doc comment exempts file",
			filename: "foo.go",
			src: "//nolint:filelen // big table\n" +
				"//go:generate stringer -type=Kind\npackage foo\n" + body(100),
			maxLen:   5,
			wantN:    0,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "nolint two lines above package clause does not exempt",
			filename: "foo.go",
			src:      "//nolint:filelen\n\npackage foo\n" + body(100),
			maxLen:   5,
			wantN:    1,
			wantLine: 3,
			wantMsg:  "file is 101 lines (limit 5)",
		},
		{
			name:     "nolint elsewhere in file does not exempt",
			filename: "foo.go",
			src:      "package foo\n//nolint:filelen\n" + body(100),
			maxLen:   5,
			wantN:    1,
			wantLine: 1,
			wantMsg:  "file is 101 lines (limit 5)",
		},
		{
			name:     "nolint other rule does not exempt",
			filename: "foo.go",
			src:      "//nolint:funclen\npackage foo\n" + body(100),
			maxLen:   5,
			wantN:    1,
			wantLine: 2,
			wantMsg:  "file is 101 lines (limit 5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.filename
			if filename == "" {
				filename = "test.go"
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, filename, tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, filename, f, []byte(tt.src), tt.maxLen)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN == 0 {
				return
			}
			if diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
			if diags[0].Line != tt.wantLine {
				t.Errorf("line = %d, want %d", diags[0].Line, tt.wantLine)
			}
		})
	}
}
