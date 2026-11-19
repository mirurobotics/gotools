package typeassert

import (
	"go/parser"
	"go/token"
	"testing"
)

type checkCase struct {
	filename string
	src      string
	wantN    int
	wantMsg  string
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
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
}

func TestCheck_CommaOkForm(t *testing.T) {
	runCheckCase(t, checkCase{
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
	})
}

func TestCheck_BareAssertionInAssign(t *testing.T) {
	runCheckCase(t, checkCase{
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
	})
}

func TestCheck_TypeSwitchExempt(t *testing.T) {
	runCheckCase(t, checkCase{
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
	})
}

func TestCheck_BareAssertionInFunctionCall(t *testing.T) {
	runCheckCase(t, checkCase{
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
	})
}

func TestCheck_TestFileNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: `package foo

func f() {
	var x interface{} = 1
	val := x.(int)
	_ = val
}
`,
		wantN:   1,
		wantMsg: "bare type assertion may panic; use val, ok := x.(T)",
	})
}

func TestCheck_TypeSwitchWithAssign(t *testing.T) {
	runCheckCase(t, checkCase{
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
	})
}
