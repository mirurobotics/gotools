package straycomment

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		wantLines []int
	}{
		{
			name: "section banner flagged once per group",
			src: `package foo

// ----------------
// Helpers
// ----------------

func f() {}
`,
			wantLines: []int{3},
		},
		{
			name: "loose note in struct and const block flagged",
			src: `package foo

type T struct {
	A int

	// group of fields below

	B int
}

const (
	A = 1

	// more constants

	B = 2
)
`,
			wantLines: []int{6, 14},
		},
		{
			name: "attached comments allowed",
			src: `// Package foo is a package.
package foo

import (
	// fmt is used for printing.
	"fmt" // trailing

	// #include <stdlib.h>
	// #include <string.h>
	"C"
)

// T is a type.
type T struct {
	// A is a field.
	A int // trailing
}

// Group doc.
const (
	// A is a constant.
	A = 1 // trailing
)

// F prints.
func F() { fmt.Println() }
`,
			wantLines: nil,
		},
		{
			name: "cgo preamble on a single import allowed",
			src: `package foo

// #include <stdio.h>
// #include <stdlib.h>
import "C"
`,
			wantLines: nil,
		},
		{
			name: "header, directives, and function comments skipped",
			src: `// Copyright 2026 Example.

//go:build linux

package foo

//go:generate echo hi

func f() {
	// funccomment handles this
}
`,
			wantLines: nil,
		},
		{
			name: "comment after last declaration flagged",
			src: `package foo

func f() {}

// TODO: more
`,
			wantLines: []int{5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "foo.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			diags := Check(fset, "foo.go", f)
			if len(diags) != len(tt.wantLines) {
				t.Fatalf(
					"got %d diagnostics, want %d: %v",
					len(diags), len(tt.wantLines), diags,
				)
			}
			for i, d := range diags {
				if d.Line != tt.wantLines[i] || d.Message != message {
					t.Errorf("diag %d = %v, want line %d", i, d, tt.wantLines[i])
				}
			}
		})
	}
}
