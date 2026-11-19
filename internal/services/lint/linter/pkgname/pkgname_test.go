package pkgname

import (
	"go/parser"
	"go/token"
	"testing"
)

type checkCase struct {
	src     string
	wantN   int
	wantMsg string
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
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
}

func TestCheck_CleanName(t *testing.T) {
	runCheckCase(t, checkCase{src: `package foo`, wantN: 0, wantMsg: ""})
}

func TestCheck_CamelCaseName(t *testing.T) {
	runCheckCase(t, checkCase{src: `package fooBar`, wantN: 0, wantMsg: ""})
}

func TestCheck_TestPrefixAllowed(t *testing.T) {
	runCheckCase(t, checkCase{src: `package test_helpers`, wantN: 0, wantMsg: ""})
}

func TestCheck_MockPrefixAllowed(t *testing.T) {
	runCheckCase(t, checkCase{src: `package mock_service`, wantN: 0, wantMsg: ""})
}

func TestCheck_TestSuffixAllowed(t *testing.T) {
	runCheckCase(t, checkCase{src: `package foo_test`, wantN: 0, wantMsg: ""})
}

func TestCheck_UnderscoreInMiddleRejected(t *testing.T) {
	runCheckCase(t, checkCase{
		src:   `package my_package`,
		wantN: 1,
		wantMsg: "package name \"my_package\" contains" +
			" underscore; use camelCase instead",
	})
}

func TestCheck_MultipleUnderscoresRejected(t *testing.T) {
	runCheckCase(t, checkCase{
		src:   `package my_cool_package`,
		wantN: 1,
		wantMsg: "package name \"my_cool_package\" contains" +
			" underscore; use camelCase instead",
	})
}
