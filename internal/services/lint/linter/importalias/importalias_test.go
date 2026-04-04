package importalias

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		filename string // defaults to "test.go"
		src      string
		wantN    int
		wantMsg  string
	}{
		{name: "no alias", src: `package foo; import "fmt"`, wantN: 0},
		{
			name:  "alias without underscore",
			src:   `package foo; import errs "errors"`,
			wantN: 0,
		},
		{
			name:  "underscore alias rejected",
			src:   `package foo; import my_errs "errors"`,
			wantN: 1,
			wantMsg: `import alias "my_errs" contains` +
				` underscore; use camelCase instead`,
		},
		{
			name:  "multiple underscores rejected",
			src:   `package foo; import my_cool_pkg "errors"`,
			wantN: 1,
			wantMsg: `import alias "my_cool_pkg" contains` +
				` underscore; use camelCase instead`,
		},
		{
			name:  "test_ prefix allowed",
			src:   `package foo; import test_helpers "errors"`,
			wantN: 0,
		},
		{
			name:  "mock_ prefix allowed",
			src:   `package foo; import mock_service "errors"`,
			wantN: 0,
		},
		{
			name:  "dot import ignored",
			src:   `package foo; import . "errors"`,
			wantN: 0,
		},
		{
			name:  "blank import ignored",
			src:   `package foo; import _ "errors"`,
			wantN: 0,
		},
		{
			name:    "underscore prefix alias rejected",
			src:     `package foo; import _foo "errors"`,
			wantN:   1,
			wantMsg: `import alias "_foo" contains underscore; use camelCase instead`,
		},
		{
			name: "mixed group",
			src: `package foo

import (
	"fmt"
	my_errs "errors"
	mock_db "database/sql"
)`,
			wantN: 1,
			wantMsg: `import alias "my_errs" contains` +
				` underscore; use camelCase instead`,
		},
		{
			name:     "test file exempt",
			filename: "foo_test.go",
			src:      `package foo; import my_errs "errors"`,
			wantN:    0,
		},
		{
			name: "multiple violations",
			src: `package foo

import (
	my_errs "errors"
	my_fmt "fmt"
)`,
			wantN: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.filename
			if filename == "" {
				filename = "test.go"
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, filename, tt.src, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, filename, f)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN > 0 && tt.wantMsg != "" &&
				diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}
