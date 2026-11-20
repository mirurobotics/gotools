package analysis

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
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
	file, err := parser.ParseFile(fset, "foo.go", commentsSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if got := nonNil(DocGroups(file)); got != 7 {
		t.Errorf("DocGroups = %d non-nil groups, want 7", got)
	}
	if got := nonNil(LineComments(file)); got != 3 {
		t.Errorf("LineComments = %d non-nil groups, want 3", got)
	}
	bodies := FuncBodies(file)
	if len(bodies) != 2 {
		t.Fatalf("FuncBodies = %d, want 2", len(bodies))
	}
	if !InAnyBody(bodies[1].Lbrace+1, bodies) {
		t.Error("InAnyBody inside a body = false, want true")
	}
	if InAnyBody(file.Package, bodies) {
		t.Error("InAnyBody at package clause = true, want false")
	}
}

func TestProseLineCount(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"line comments", "// a\n// b\n", 2},
		{"inner blank line counted", "// a\n//\n// b\n", 3},
		{"directive and blank lines dropped", "//\n// a\n//\n//nolint:x\n", 1},
		{"block comment trimmed", "/*\na\n*/\n", 1},
		{"starred block comment trimmed", "/*\n * a\n *\n * b\n */\n", 3},
		{"trailing bare star dropped", "/*\n * a\n *\n */\n", 1},
		{"empty block comment", "/* */\n", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			src := tc.src + "package foo\n"
			file, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			if got := ProseLineCount(file.Doc); got != tc.want {
				t.Errorf("ProseLineCount = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestNamedDocs(t *testing.T) {
	src := commentsSrc + `
// Pair doc.
var a, b = 1, 2

// Group doc.
const (
	// C doc.
	C = 1
)

func Local() {
	// x doc.
	var x = 1 // x line
	_ = x
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-", "-", "T", "A", "V", "F", "H", "-", "-", "C"}
	if got := documentedNames(file); !reflect.DeepEqual(got, want) {
		t.Errorf("NamedDocs names = %v, want %v", got, want)
	}
	if count := nonNil(LineComments(file)); count != 3 {
		t.Errorf("LineComments = %d non-nil groups, want 3 (none from bodies)", count)
	}
}

func nonNil[T any](items []*T) int {
	count := 0
	for _, item := range items {
		if item != nil {
			count++
		}
	}
	return count
}

func TestNamedDocs_Import(t *testing.T) {
	const importSrc = "package bar\n\n// os doc.\nimport \"os\"\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "bar.go", importSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := documentedNames(file)
	if want := []string{"-"}; !reflect.DeepEqual(got, want) {
		t.Errorf("NamedDocs on import = %v, want %v", got, want)
	}
}

func documentedNames(file *ast.File) []string {
	var got []string
	for _, doc := range NamedDocs(file) {
		if doc.Doc == nil {
			continue
		}
		name := "-"
		if doc.Name != nil {
			name = doc.Name.Name
		}
		got = append(got, name)
	}
	return got
}

func TestProseLines(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"line comments", "// a\n// b\n", []string{" a", " b"}},
		{"directive dropped", "// a\n//nolint:x\n", []string{" a"}},
		{"line comment star kept", "// * a\n", []string{" * a"}},
		{"starred block split", "/*\n * a\n */\n", []string{"", " a", " "}},
		{"plain block split", "/* a\nb */\n", []string{" a", "b "}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			src := tc.src + "package foo\n"
			file, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			if got := ProseLines(file.Doc); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ProseLines = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestComments(t *testing.T) {
	src := "// a\n// b\npackage p\n\n/* c */\nvar x = 1 // d\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := Comments(file)
	want := []string{"// a", "// b", "/* c */", "// d"}
	if len(got) != len(want) {
		t.Fatalf("got %d comments, want %d", len(got), len(want))
	}
	for i, comment := range got {
		if comment.Text != want[i] {
			t.Errorf("comment %d = %q, want %q", i, comment.Text, want[i])
		}
	}
}
