package typeassert

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename string
	src      string
	wantN    int
	wantMsg  string
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

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset, f := linttest.Parse(t, tt.filename, tt.src)
	linttest.AssertCount(t, Check(fset, tt.filename, f), tt.wantN, tt.wantMsg)
}
