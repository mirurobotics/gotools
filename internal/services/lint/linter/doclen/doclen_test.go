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
		maxLines  int
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
			maxLines:  1,
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
			maxLines:  1,
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
			maxLines:  1,
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
			maxLines:  1,
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
			maxLines:  1,
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
			maxLines:  1,
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
			maxLines:  2,
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
			maxLines:  0,
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
			diags := Check(fset, tt.filename, f, tt.maxLines)
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
