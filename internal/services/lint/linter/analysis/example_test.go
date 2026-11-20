package analysis

import (
	"go/parser"
	"go/token"
	"testing"
)

const exampleSrc = `package foo

// Output: not in a function
func ExampleF() {
	// x := 1

	println()
	// Output:
	// 1
}

func ExampleG() {
	// Unordered output:
	// 2
}

func ExampleH()

func H() {
	// Output:
	// 3
}

var V = 1
`

func TestExampleOutputs(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo_test.go", exampleSrc, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := ExampleOutputs("foo_test.go", file)
	want := map[string]bool{
		"Output: not in a function\n": false,
		"x := 1\n":                    false,
		"Output:\n1\n":                true,
		"Unordered output:\n2\n":      true,
		"Output:\n3\n":                false,
	}
	for _, group := range file.Comments {
		if text := group.Text(); got[group] != want[text] {
			t.Errorf("ExampleOutputs[%q] = %v, want %v", text, got[group], want[text])
		}
	}
	if len(got) != 2 {
		t.Errorf("ExampleOutputs in test file = %d groups, want 2", len(got))
	}
	if got := len(ExampleOutputs("foo.go", file)); got != 0 {
		t.Errorf("ExampleOutputs in non-test file = %d groups, want 0", got)
	}
}
