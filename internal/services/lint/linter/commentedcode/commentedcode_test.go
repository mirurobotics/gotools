package commentedcode

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
			name: "statements in function flagged once per group",
			src: `package foo

func f() int {
	// x := 1
	// x++
	// return x
	return 2
}
`,
			wantLines: []int{4},
		},
		{
			name: "declarations at top level flagged",
			src: `package foo

// func old() {
// 	println("hi")
// }

// var legacy = 3
`,
			wantLines: []int{3, 7},
		},
		{
			name: "code line after prose flagged",
			src: `package foo

func f() {
	// the old way:
	//cleanup(true)
}
`,
			wantLines: []int{5},
		},
		{
			name: "block comments and control flow flagged",
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
			wantLines: []int{3, 10, 12, 17, 19},
		},
		{
			name: "gofmt'd and column-zero toggled code flagged",
			src: `package foo

// func old() {
//	println("hi")
// }

//	x := 1
//	return x
`,
			wantLines: []int{3, 7},
		},
		{
			name: "parenthesized call and receive flagged",
			src: `package foo

// (run())

// <-done

// (*p), q.x, a[0] = 1, 2, 3

// prev :=
//	cur
`,
			wantLines: []int{3, 5, 7, 9},
		},
		{
			name: "prose and doc comments allowed",
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
// deprecated
//
func F() {}
`,
			wantLines: nil,
		},
		{
			name: "trailing comments, examples, directives, and output skipped",
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
		},
		{
			name: "block docs with indented examples and cgo preambles skipped",
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
*/
import "C"

import (
	/*
	int y = 2;
	*/
	"C"
)
`,
			wantLines: nil,
		},
		{
			name: "empty and blank comments skipped",
			src: `package foo

//
/**/
`,
			wantLines: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			const name = "foo_test.go"
			f, err := parser.ParseFile(fset, name, tt.src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			diags := Check(fset, name, f, []byte(tt.src))
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
