package doclen

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		src       string
		lim       Limits
		wantLines []int
		wantMsg   string
	}{
		{
			name:     "multi-line func doc flagged",
			filename: "foo.go",
			src: `package foo

// F does a thing.
// And explains it at length.
func F() {}
`,
			lim:       Limits{Doc: 1, DocGoPkg: 20},
			wantLines: []int{3},
			wantMsg:   "doc comment is 2 lines (limit 1)",
		},
		{
			name:     "one-line docs with directives allowed",
			filename: "foo.go",
			src: `package foo

// F does a thing.
//
//nolint:funclen // reason
func F() {}

// T is a type.
//
//go:generate echo hi
type T struct {
	// A is a field.
	A int // trailing
}
`,
			lim:       Limits{Doc: 1, DocGoPkg: 20},
			wantLines: nil,
			wantMsg:   "",
		},
		{
			name:     "type, value, and field docs flagged",
			filename: "foo.go",
			src: `package foo

// T is a type
// with two lines.
type T struct {
	// A is a field
	// with two lines.
	A int
}

const (
	// C is a const
	// with two lines.
	C = 1
)
`,
			lim:       Limits{Doc: 1, DocGoPkg: 20},
			wantLines: []int{3, 6, 12},
			wantMsg:   "",
		},
		{
			name:     "block comment counted by lines",
			filename: "foo.go",
			src: `package foo

/* F does a thing
and more. */
func F() {}
`,
			lim:       Limits{Doc: 1, DocGoPkg: 20},
			wantLines: []int{3},
			wantMsg:   "",
		},
		{
			name:     "package doc flagged outside doc.go",
			filename: "foo.go",
			src: `// Package foo does things.
// At length.
package foo
`,
			lim:       Limits{Doc: 1, DocGoPkg: 20},
			wantLines: []int{1},
			wantMsg:   "",
		},
		{
			name:     "package doc in doc.go skipped",
			filename: "doc.go",
			src: `// Package foo does things.
// At length.
package foo
`,
			lim:       Limits{Doc: 1, DocGoPkg: 20},
			wantLines: nil,
			wantMsg:   "",
		},
		{
			name:     "higher limit allowed",
			filename: "foo.go",
			src: `package foo

// F does a thing.
// And explains it.
func F() {}
`,
			lim:       Limits{Doc: 2, DocGoPkg: 20},
			wantLines: nil,
			wantMsg:   "",
		},
		{
			name:     "doc.go package doc over its own limit flagged",
			filename: "pkg/doc.go",
			src: `// Copyright header.
// Not counted.

// Package foo does things.
// At length.
// And more.
package foo
`,
			lim:       Limits{Doc: 1, DocGoPkg: 2},
			wantLines: []int{4},
			wantMsg:   "doc comment is 3 lines (limit 2)",
		},
		{
			name:     "cgo preamble and export directive skipped",
			filename: "foo.go",
			src: `package foo

// #include <stdio.h>
// #cgo LDFLAGS: -lm
// static int one() { return 1; }
import "C"

// F is exported to C.
//
//export F
func F() {}
`,
			lim:       Limits{Doc: 1, DocGoPkg: 20},
			wantLines: nil,
			wantMsg:   "",
		},
		{
			name:     "disabled when limit is zero",
			filename: "foo.go",
			src: `package foo

// F does a thing.
// And explains it.
func F() {}
`,
			lim:       Limits{Doc: 0, DocGoPkg: 20},
			wantLines: nil,
			wantMsg:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, tt.filename, tt.src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			diags := Check(fset, tt.filename, f, tt.lim)
			if len(diags) != len(tt.wantLines) {
				t.Fatalf(
					"got %d diagnostics, want %d: %v",
					len(diags), len(tt.wantLines), diags,
				)
			}
			for i, d := range diags {
				if d.Line != tt.wantLines[i] {
					t.Errorf("diag %d line = %d, want %d", i, d.Line, tt.wantLines[i])
				}
				if tt.wantMsg != "" && d.Message != tt.wantMsg {
					t.Errorf("diag %d message = %q, want %q", i, d.Message, tt.wantMsg)
				}
			}
		})
	}
}
