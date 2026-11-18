package privdoc

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
			name: "unexported func and method docs flagged",
			src: `package foo

// f does a thing.
func f() {}

// m does a thing.
func (T) m() {}

// M is exported on an unexported type.
func (*t) M() {}

// G is generic on an unexported type.
func (g[K]) G() {}
`,
			wantLines: []int{3, 6, 9, 12},
		},
		{
			name: "exported docs allowed",
			src: `package foo

// F does a thing.
func F() {}

// M does a thing.
func (*T) M() {}

// T is a type.
type T struct {
	// A is a field.
	A int // trailing
	// Embedded is embedded.
	Embedded
	// Other is embedded from a package.
	*pkg.Other
	// Generic is a generic embed.
	Generic[int]
	// Pair is a generic embed with two arguments.
	Pair[int, string]
	// Inner is an exported nested struct.
	Inner struct {
		// B is a field.
		B int
	}
}

// I is an interface.
type I interface {
	// Do does.
	Do()
}

// C is a constant.
const C = 1

// Values mixes exported and unexported names.
var (
	// V is exported.
	V, v = 1, 2
)

// m is a method on an unknown receiver.
func (*[2]T) M() {}
`,
			wantLines: nil,
		},
		{
			name: "unexported types, fields, and values flagged",
			src: `package foo

// t is a type.
type t struct {
	// A is on an unexported type.
	A int
}

type T struct {
	a int // trailing
	// embedded is embedded.
	embedded
	// Inner is exported.
	Inner struct {
		// b is unexported.
		b int
	}
	hidden struct {
		// B is in an unexported field.
		B int
	}
}

type I interface {
	// do does.
	do()
}

// c is a constant.
const c = 1

// group of unexported values.
var (
	// x is unexported.
	x = 1
	y = 2 // trailing
)
`,
			wantLines: []int{3, 5, 10, 11, 15, 19, 25, 29, 32, 34, 36},
		},
		{
			name: "directives and blank lines allowed",
			src: `package foo

//nolint:funclen // reason
func f() {}

//
//go:generate echo hi
type t int

var x = 1 //nolint:gochecknoglobals
`,
			wantLines: nil,
		},
		{
			name: "imports ignored",
			src: `package foo

// fmt is imported.
import "fmt"
`,
			wantLines: nil,
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
