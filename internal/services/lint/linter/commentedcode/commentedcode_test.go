package commentedcode

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename  string
	src       string
	wantLines []int
}

func TestCheck_StatementsInFunctionFlaggedOncePerGroup(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

func f() int {
	// x := 1
	// x++
	// return x
	return 2
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_DeclarationsAtTopLevelFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

// func old() {
// 	println("hi")
// }

// var legacy = 3
`,
		wantLines: []int{3, 7},
	})
}

func TestCheck_CodeLineAfterProseFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

func f() {
	// the old way:
	//cleanup(true)
}
`,
		wantLines: []int{5},
	})
}

func TestCheck_BlockCommentsAndControlFlowFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

/*
		if ok {
	go run()
}
*/

func f(ch chan int) {
	/* ch <- 1 */

	/* old way:
	cleanup(true)

	*/

	// for i := range ch {}

	// switch {}
}
`,
		wantLines: []int{4, 10, 13, 17, 19},
	})
}

func TestCheck_GofmtdAndColumnZeroToggledCodeFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

// func old() {
//	println("hi")
// }

//	x := 1
//	return x
`,
		wantLines: []int{3, 7},
	})
}

func TestCheck_ParenthesizedCallAndReceiveFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

// (run())

// <-done

// (*p), q.x, a[0] = 1, 2, 3

// prev :=
//	cur
`,
		wantLines: []int{3, 5, 7, 9},
	})
}

func TestCheck_MethodChainSplitAcrossLinesFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

func f() {
	// b.
	// 	Run()
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_CommentOnGroupedCgoImportIsNotAPreamble(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

// fmt.Println("x")
import (
	"C"
	"fmt"
)
`,
		wantLines: []int{3},
	})
}

func TestCheck_BlockCommentLinesInACRLFFile(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: "package foo\r\n\r\n/*\r\n\t\tif ok {}\r\n*/\r\n\r\n" +
			"/* old way:\r\ncleanup(true) */\r\n",
		wantLines: []int{4, 8},
	})
}

func TestCheck_BlockCommentLinesAfterALineDirective(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

//line x.go:1000
/* old way:
cleanup(true) */
`,
		wantLines: []int{1001},
	})
}

func TestCheck_CodeInAnExampleBodyFlaggedOutputSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

func ExampleF() {
	// x := 1

	println("a")
	// Output:
	// x = 1
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_OutputCommentOutsideAnExampleTestFileFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: `package foo

func ExampleF() {
	// x := 1

	println("a")
	// Output:
	// x = 1
}
`,
		wantLines: []int{4, 8},
	})
}

func TestCheck_FirstCommentAfterAByteOrderMarkChecked(t *testing.T) {
	runCheckCase(t, checkCase{
		filename:  "foo_test.go",
		src:       "\uFEFF// x := 1\npackage foo\n",
		wantLines: []int{1},
	})
}

func TestCheck_NolintDirectiveNamingTheRuleSkipsTheGroup(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

//nolint:commentedcode
// x := 1

//nolint:todoissue
// y := 2
`,
		wantLines: []int{7},
	})
}

func TestCheck_UnboundTypeSwitchFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

/*
switch x.(type) {
case int:
	f()
}
*/
`,
		wantLines: []int{4},
	})
}

func TestCheck_NonASCIIStringLiteralFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

// fmt.Println("héllo")
`,
		wantLines: []int{3},
	})
}

func TestCheck_DeclarationsWithValuesOrCompositeTypesFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

// type T struct{}

// type A = B

// var xs []int

// const c = 1

// import "fmt"

func f() {
	// var m map[string]int
}
`,
		wantLines: []int{3, 5, 7, 9, 11, 14},
	})
}

func TestCheck_CodeAfterDirectiveMidGroupFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

// old way:
//nolint:todoissue
// x := 1
`,
		wantLines: []int{5},
	})
}

func TestCheck_BareSpecsInLargerRunsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

// const (
//	A = iota
//	B
// )

// var mu sync.Mutex
//
// func lock() {
//	mu.Lock()
// }

func f() {
	// var wg sync.WaitGroup
	// wg.Add(1)
}
`,
		wantLines: []int{3, 8, 15},
	})
}

func TestCheck_NestedBareVarFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

func f() {
	// for _, it := range items {
	//	var buf bytes.Buffer
	//	render(&buf, it)
	// }
}
`,
		wantLines: []int{4},
	})
}

func runCheckCase(t *testing.T, tc checkCase) {
	t.Helper()
	fset, file := linttest.Parse(t, tc.filename, tc.src)
	diags := Check(fset, tc.filename, file, []byte(tc.src))
	linttest.AssertLines(t, diags, tc.wantLines, message)
}
