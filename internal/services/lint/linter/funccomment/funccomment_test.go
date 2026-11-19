package funccomment

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename  string
	src       string
	wantLines []int
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset, f := linttest.Parse(t, tt.filename, tt.src)
	linttest.AssertLines(t, Check(fset, tt.filename, f), tt.wantLines, message)
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

func TestCheck_OutputCommentInExampleLookalikesFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

func (T) ExampleM() {
	// Output: 1
}

func Examplehelper() {
	// Output: 2
}
`,
		wantLines: []int{4, 8},
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
