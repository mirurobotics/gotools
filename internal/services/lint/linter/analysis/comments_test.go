package analysis

import (
	"go/parser"
	"go/token"
	"testing"
)

const commentsSrc = `package foo

import (
	// fmt doc.
	"fmt" // fmt line

	// #include <stdlib.h>
	"C"
)

// T doc.
type T struct {
	// A doc.
	A int // A line
}

// V doc.
var V = 1 // V line

// F doc.
func F() {
	g := func() { fmt.Println() }
	g()
}

// H doc.
func H()
`

func TestCommentHelpers(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", commentsSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if got := nonNil(DocGroups(f)); got != 7 {
		t.Errorf("DocGroups = %d non-nil groups, want 7", got)
	}
	if got := nonNil(LineComments(f)); got != 3 {
		t.Errorf("LineComments = %d non-nil groups, want 3", got)
	}
	bodies := FuncBodies(f)
	if len(bodies) != 2 {
		t.Fatalf("FuncBodies = %d, want 2", len(bodies))
	}
	if !InAnyBody(bodies[1].Lbrace+1, bodies) {
		t.Error("InAnyBody inside a body = false, want true")
	}
	if InAnyBody(f.Package, bodies) {
		t.Error("InAnyBody at package clause = true, want false")
	}
}

func nonNil[T any](xs []*T) int {
	n := 0
	for _, x := range xs {
		if x != nil {
			n++
		}
	}
	return n
}

func TestProseLines(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"line comments", "// a\n// b\n", 2},
		{"inner blank line counted", "// a\n//\n// b\n", 3},
		{"directive and blank lines dropped", "//\n// a\n//\n//nolint:x\n", 1},
		{"block comment trimmed", "/*\na\n*/\n", 1},
		{"empty block comment", "/* */\n", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			src := tt.src + "package foo\n"
			f, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			if got := ProseLines(f.Doc); got != tt.want {
				t.Errorf("ProseLines = %d, want %d", got, tt.want)
			}
		})
	}
}
