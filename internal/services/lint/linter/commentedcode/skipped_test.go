package commentedcode

import "testing"

func TestCheck_ProseAndDocCommentsAllowed(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `// Package foo does things.
package foo

// F returns the value of x.
// See https://example.com/docs for more.
// Note: call Init() first.
// TODO(ben): handle errors
// - list item
// a = b in this context, where f(x) = y.
// 0 < len(p) < 10
// fmt.Println
// x.y
// Get(k) = v
// extension-method = token
// break
// {}
// {name}
// if ok { Note: F() }
// v.(type)
// f()++
// e.g. Close()
// i.e. x.Run()
// cf. foo()
// deprecated
//
func F() {}
`,
		wantLines: nil,
	})
}

func TestCheck_TrailingCommentsExamplesDirectivesAndOutputSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

const n = 20 // len(fmt.Sprint(x))

// Example:
//	F()
//  x := F()
// nolint(all)
//go:generate go run gen.go

func ExampleF() {
	println("a")
	// Output:
	// x = 1
}
`,
		wantLines: nil,
	})
}

func TestCheck_BlockDocsWithIndentedExamplesAndCgoPreamblesSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `/*
Package foo does things.

	x := foo.New()
	x.Run()
*/
package foo

/*
#include <stdio.h>
static int x = 1;
void f() { x++; }
printf("hi");
*/
import "C"

import (
	/*
	int y = 2;
	printf("hi");
	*/
	"C"
)
`,
		wantLines: nil,
	})
}

func TestCheck_BlockUsageIndentedPastItsOpenerSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

/* Usage:
	foo.Bar()
*/
`,
		wantLines: nil,
	})
}

func TestCheck_BlockCommentAfterAByteOrderMarkKeepsItsIndent(t *testing.T) {
	runCheckCase(t, checkCase{
		filename:  "foo_test.go",
		src:       "\uFEFF/* Usage:\n\tfoo.Bar()\n*/\npackage foo\n",
		wantLines: nil,
	})
}

func TestCheck_EmptyAndBlankCommentsSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

//
/**/
`,
		wantLines: nil,
	})
}
