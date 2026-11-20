package exprwrap

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	src       string
	wantLines []int
}

func TestCheck_SingleLine(t *testing.T) {
	runCheckCase(t, checkCase{
		src:       "package foo\n\nfunc f(a, b bool) bool {\n\treturn a || b\n}\n",
		wantLines: nil,
	})
}

func TestCheck_BooleanChainInAssignment(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b, c bool) {
	skip := a || b ||
		c
	_ = skip
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_IfCondition(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b bool) {
	if a &&
		b {
		return
	}
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_ReturnValue(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b bool) bool {
	return a ||
		b
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_SelectorChain(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(x interface{ Y() int }) int {
	return x.
		Y()
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_MultiLineCallArgumentsExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(g func(int, int) int) int {
	return g(
		1,
		2,
	)
}
`,
		wantLines: nil,
	})
}

func TestCheck_WrappedCallArgumentIsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(g func(bool) bool, a, b bool) bool {
	return g(
		a ||
			b,
	)
}
`,
		wantLines: []int{5},
	})
}

func TestCheck_CallChainedAfterMultiLineArgsExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type T struct{}

func (T) Do() int { return 0 }

func mk(int) T { return T{} }

func f() int {
	return mk(
		1,
	).Do()
}
`,
		wantLines: nil,
	})
}

func TestCheck_TableTestCompositeLiteralExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	tests := []struct {
		name string
		n    int
	}{
		{name: "a", n: 1},
		{
			name: "b",
			n:    2,
		},
	}
	_ = tests
}
`,
		wantLines: nil,
	})
}

func TestCheck_WrappedCompositeValueIsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b bool) {
	m := map[string]bool{
		"x": a ||
			b,
	}
	_ = m
}
`,
		wantLines: []int{5},
	})
}

func TestCheck_FuncLiteralExemptBodyChecked(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() func() bool {
	return func() bool {
		x := 1
		return x == 1 ||
			x == 2
	}
}
`,
		wantLines: []int{6},
	})
}

func TestCheck_ReportedOnTheLineOfTheBreak(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(g func(int, int) bool, ok bool) bool {
	x := g(
		1,
		2,
	) ||
		ok
	return x
}
`,
		wantLines: []int{7},
	})
}

func TestCheck_PackageLevelVar(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

var a, b = true, false

var c = a ||
	b
`,
		wantLines: []int{5},
	})
}

func TestCheck_ForSwitchCaseSendIncdecRangeGoDefer(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b bool, ch chan bool, m map[int]int, g func(bool), r func(bool) []int) {
	for a &&
		b {
	}
	switch a ||
		b {
	case a &&
		b:
	}
	ch <- a ||
		b
	m[1+
		1]++
	for range r(a ||
		b) {
	}
	go g(a ||
		b)
	defer g(a ||
		b)
}
`,
		wantLines: []int{4, 7, 9, 12, 14, 16, 19, 21},
	})
}

func TestCheck_TypeSetUnionInTypeParameterListSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func Sum[T ~int |
	~float64](xs []T) T {
	var t T
	return t
}

type Pair[T ~int |
	~string] struct{ a, b T }
`,
		wantLines: nil,
	})
}

func TestCheck_BitwiseOrOutsideATypeSetIsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b int) int {
	return a |
		b
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_TypeSetUnionInInterfaceSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type Number interface {
	~int |
		~float64
}
`,
		wantLines: nil,
	})
}

func TestCheck_ExpressionStatementBareForAndSwitch(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

// g does nothing.
func g(bool) {}

func f(a, b bool) {
	g(a)
	for {
		switch {
		default:
			return
		}
	}
}
`,
		wantLines: nil,
	})
}

func TestCheck_ChainWithSeveralBreaksReportsEach(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b, c bool) bool {
	return a ||
		b ||
		c
}
`,
		wantLines: []int{4, 5},
	})
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset, f := linttest.Parse(t, "foo.go", tt.src)
	linttest.AssertLines(t, Check(fset, "foo.go", f), tt.wantLines, message)
}
