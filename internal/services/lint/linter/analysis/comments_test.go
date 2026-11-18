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
	if got := nonNil(DocGroups(f)); got != 6 {
		t.Errorf("DocGroups = %d non-nil groups, want 6", got)
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
