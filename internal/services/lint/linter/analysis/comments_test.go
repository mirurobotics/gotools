package analysis

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestComments(t *testing.T) {
	src := "// a\n// b\npackage p\n\n/* c */\nvar x = 1 // d\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := Comments(f)
	want := []string{"// a", "// b", "/* c */", "// d"}
	if len(got) != len(want) {
		t.Fatalf("got %d comments, want %d", len(got), len(want))
	}
	for i, c := range got {
		if c.Text != want[i] {
			t.Errorf("comment %d = %q, want %q", i, c.Text, want[i])
		}
	}
}

func TestCommentsNone(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", "package p\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if got := Comments(f); len(got) != 0 {
		t.Errorf("got %d comments, want 0", len(got))
	}
}
