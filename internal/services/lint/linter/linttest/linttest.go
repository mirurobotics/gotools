// Package linttest holds assertions shared by the linter rule tests.
package linttest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Parse parses src as filename with comments, failing the test on error.
func Parse(t *testing.T, filename, src string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return fset, f
}

// AssertCount checks the diagnostic count and, when wantMsg is set, the first message.
func AssertCount(t *testing.T, diags []analysis.Diagnostic, wantN int, wantMsg string) {
	t.Helper()
	if len(diags) != wantN {
		t.Fatalf("got %d diagnostics, want %d: %v", len(diags), wantN, diags)
	}
	if wantN > 0 && wantMsg != "" && diags[0].Message != wantMsg {
		t.Errorf("message = %q, want %q", diags[0].Message, wantMsg)
	}
}

// AssertLines checks each diagnostic's line and, when wantMsg is set, its message.
func AssertLines(
	t *testing.T,
	diags []analysis.Diagnostic,
	wantLines []int,
	wantMsg string,
) {
	t.Helper()
	if len(diags) != len(wantLines) {
		t.Fatalf("got %d diagnostics, want %d: %v", len(diags), len(wantLines), diags)
	}
	for i, d := range diags {
		if d.Line != wantLines[i] {
			t.Errorf("diag %d line = %d, want %d", i, d.Line, wantLines[i])
		}
		if wantMsg != "" && d.Message != wantMsg {
			t.Errorf("diag %d message = %q, want %q", i, d.Message, wantMsg)
		}
	}
}
