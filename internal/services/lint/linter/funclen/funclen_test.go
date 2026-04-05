package funclen

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		src      string
		maxLen   int
		wantN    int
		wantMsg  string
	}{
		{
			name:     "disabled when limit zero",
			filename: "",
			src: "package foo\n\nfunc f() {\n" +
				strings.Repeat("\tx := 1\n", 100) + "}\n",
			maxLen:  0,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "under limit",
			filename: "",
			src:      "package foo\n\nfunc f() {\n\tx := 1\n\t_ = x\n}\n",
			maxLen:   50,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name:     "at exactly limit",
			filename: "",
			src: "package foo\n\nfunc f() {\n" +
				strings.Repeat("\tx := 1\n", 5) + "}\n",
			maxLen:  5,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "over limit",
			filename: "",
			src: "package foo\n\nfunc f() {\n" +
				strings.Repeat("\tx := 1\n", 6) + "}\n",
			maxLen:  5,
			wantMsg: "function f is 6 lines (limit 5)",
			wantN:   1,
		},
		{
			name:     "blank lines not counted",
			filename: "",
			src:      "package foo\n\nfunc f() {\n\tx := 1\n\n\n\ty := 2\n}\n",
			maxLen:   2,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name:     "comment lines not counted",
			filename: "",
			src: "package foo\n\nfunc f() {\n" +
				"\t// comment\n\tx := 1\n" +
				"\t// another comment\n\ty := 2\n}\n",
			maxLen:  2,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "function literal checked",
			filename: "",
			src: "package foo\n\nvar f = func() {\n" +
				strings.Repeat("\tx := 1\n", 4) + "}\n",
			maxLen:  3,
			wantN:   1,
			wantMsg: "anonymous function is 4 lines (limit 3)",
		},
		{
			name:     "empty function",
			filename: "",
			src:      "package foo\n\nfunc f() {\n}\n",
			maxLen:   50,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name:     "nil body func decl",
			filename: "",
			src:      "package foo\n\nfunc Foo()\n",
			maxLen:   1,
			wantN:    0,
			wantMsg:  "",
		},
		{
			name:     "func literal under limit",
			filename: "",
			src: "package foo\n\nvar f = func() {\n" +
				strings.Repeat("\tx := 1\n", 2) + "}\n",
			maxLen:  5,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "test helper exempt",
			filename: "tests/helpers/foo.go",
			src: "package helpers\n\nfunc f() {\n" +
				strings.Repeat("\tx := 1\n", 100) +
				"}\n",
			maxLen:  5,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "test file exempt",
			filename: "foo_test.go",
			src: "package foo\n\nfunc f() {\n" +
				strings.Repeat("\tx := 1\n", 100) +
				"}\n",
			maxLen:  5,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint:funclen on func line exempts FuncDecl",
			filename: "foo.go",
			src: "package foo\n\n" +
				"//nolint:funclen\n" +
				"func f() {\n" +
				strings.Repeat("\tx := 1\n", 6) + "}\n",
			maxLen:  5,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint:funclen inline on func line exempts FuncDecl",
			filename: "foo.go",
			src: "package foo\n\n" +
				"func f() { //nolint:funclen\n" +
				strings.Repeat("\tx := 1\n", 6) + "}\n",
			maxLen:  5,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint:funclen on func literal line exempts FuncLit",
			filename: "foo.go",
			src: "package foo\n\n" +
				"//nolint:funclen\n" +
				"var f = func() {\n" +
				strings.Repeat("\tx := 1\n", 6) + "}\n",
			maxLen:  5,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint other rule does not exempt FuncDecl",
			filename: "foo.go",
			src: "package foo\n\n" +
				"//nolint:mutableglobal\n" +
				"func f() {\n" +
				strings.Repeat("\tx := 1\n", 6) + "}\n",
			maxLen:  5,
			wantN:   1,
			wantMsg: "function f is 6 lines (limit 5)",
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
			if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}
