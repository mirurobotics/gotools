package funccomment

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		src       string
		wantLines []int
	}{
		{
			filename: "foo.go",
			name:     "line comment in body flagged",
			src: `package foo

func f() {
	// explain
	x := 1
	_ = x
}
`,
			wantLines: []int{4},
		},
		{
			filename: "foo.go",
			name:     "trailing and block comments flagged",
			src: `package foo

func f() int {
	x := 1 // trailing
	/* block */
	return x
}
`,
			wantLines: []int{4, 5},
		},
		{
			filename: "foo.go",
			name:     "func literal flagged",
			src: `package foo

var g = func() {
	// inside literal
}
`,
			wantLines: []int{4},
		},
		{
			filename: "foo.go",
			name:     "directives allowed",
			src: `package foo

func f() {
	x := 1 //nolint:gosec // reason
	// #nosec G304 -- reason
	_ = x
}
`,
			wantLines: nil,
		},
		{
			filename: "foo_test.go",
			name:     "example output allowed",
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
	// Unordered output:
	// 1
}
`,
			wantLines: nil,
		},
		{
			filename: "foo.go",
			name:     "output comment outside example test flagged",
			src: `package foo

func Example() {
	// Output:
	// 1
}
`,
			wantLines: []int{4, 5},
		},
		{
			filename: "foo.go",
			name:     "doc comments and declarations without body allowed",
			src: `package foo

// f does a thing.
func f() {}

// g is external.
func g()
`,
			wantLines: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}
