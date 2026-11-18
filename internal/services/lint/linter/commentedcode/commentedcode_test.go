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
			name: "block comment and control flow flagged",
			src: `package foo

/*
if ok {
	go run()
}
*/

func f(ch chan int) {
	/* ch <- 1 */
	// (close(ch))
	// for i := range ch {}
	// <-ch
}
`,
			wantLines: []int{3, 10},
		},
		{
			name: "parenthesized call and receive flagged",
			src: `package foo

// (run())

// <-done
`,
			wantLines: []int{3, 5},
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
//go:generate go run gen.go
//nolint:all

func ExampleF() {
	println("a")
	// Output:
	// x = 1
}
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
			f, err := parser.ParseFile(fset, "foo_test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			diags := Check(fset, "foo_test.go", f, []byte(tt.src))
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
