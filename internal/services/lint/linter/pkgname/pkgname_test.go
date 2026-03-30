package pkgname

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantN   int
		wantMsg string
	}{
		{name: "clean name", src: `package foo`, wantN: 0, wantMsg: ""},
		{name: "camelCase name", src: `package fooBar`, wantN: 0, wantMsg: ""},
		{
			name: "test_ prefix allowed",
			src:  `package test_helpers`, wantN: 0, wantMsg: "",
		},
		{
			name: "mock_ prefix allowed",
			src:  `package mock_service`, wantN: 0, wantMsg: "",
		},
		{name: "_test suffix allowed", src: `package foo_test`, wantN: 0, wantMsg: ""},
		{
			name:  "underscore in middle rejected",
			src:   `package my_package`,
			wantN: 1,
			wantMsg: "package name \"my_package\" contains" +
				" underscore; use camelCase instead",
		},
		{
			name:  "multiple underscores rejected",
			src:   `package my_cool_package`,
			wantN: 1,
			wantMsg: "package name \"my_cool_package\" contains" +
				" underscore; use camelCase instead",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.src, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, "test.go", f)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN > 0 && diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}
