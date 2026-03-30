package typeassert

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		src      string
		wantN    int
		wantMsg  string
	}{
		{
			name:     "comma-ok form",
			filename: "foo.go",
			src: `package foo

func f() {
	var x interface{} = 1
	val, ok := x.(int)
	_ = val
	_ = ok
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "bare assertion in assign",
			filename: "foo.go",
			src: `package foo

func f() {
	var x interface{} = 1
	val := x.(int)
	_ = val
}
`,
			wantN:   1,
			wantMsg: "bare type assertion may panic; use val, ok := x.(T)",
		},
		{
			name:     "type switch exempt",
			filename: "foo.go",
			src: `package foo

func f() {
	var x interface{} = 1
	switch x.(type) {
	case int:
	}
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "bare assertion in function call",
			filename: "foo.go",
			src: `package foo

func g(i int) {}

func f() {
	var x interface{} = 1
	g(x.(int))
}
`,
			wantN:   1,
			wantMsg: "bare type assertion may panic; use val, ok := x.(T)",
		},
		{
			name:     "test file exempt",
			filename: "foo_test.go",
			src: `package foo

func f() {
	var x interface{} = 1
	val := x.(int)
	_ = val
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "type switch with assign",
			filename: "foo.go",
			src: `package foo

func f() {
	var x interface{} = 1
	switch v := x.(type) {
	case int:
		_ = v
	}
}
`,
			wantN:   0,
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, tt.filename, tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, tt.filename, f)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}
