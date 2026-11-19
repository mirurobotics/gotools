package funccomment

import (
	"go/parser"
	"go/token"
	"testing"
)

type checkCase struct {
	filename  string
	src       string
	wantLines []int
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, tt.filename, tt.src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	diags := Check(fset, tt.filename, f)
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
}

func TestCheck_LineCommentInBodyFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func f() {
	// explain
	x := 1
	_ = x
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_TrailingAndBlockCommentsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func f() int {
	x := 1 // trailing
	/* block */
	return x
}
`,
		wantLines: []int{4, 5},
	})
}

func TestCheck_FuncLiteralFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

var g = func() {
	// inside literal
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_DirectivesAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func f() {
	x := 1 //nolint:gosec // reason
	// #nosec G304 -- reason
	_ = x
}
`,
		wantLines: nil,
	})
}

func TestCheck_ExampleOutputAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

func Example() {
	println(1)
	// Output:
	// 1
}

var x = 1

func ExampleExternal()

func ExampleUnordered() {
	println(1)
	// Unordered Output:
	// 1
}

func ExampleLower() {
	println(1)
	// output: 1
}
`,
		wantLines: nil,
	})
}

func TestCheck_ProseInExampleAndOutputInNonExampleFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

func Example() {
	// prose
	println(1)
	// Output:
	// 1
}

func helper() {
	// Output:
	// 1
}
`,
		wantLines: []int{4, 11, 12},
	})
}

func TestCheck_OutputCommentOutsideExampleTestFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func Example() {
	// Output:
	// 1
}
`,
		wantLines: []int{4, 5},
	})
}

func TestCheck_DocCommentsAndDeclarationsWithoutBodyAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

// f does a thing.
func f() {}

// g is external.
func g()
`,
		wantLines: nil,
	})
}
