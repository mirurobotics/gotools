package exprwrap

import (
	"testing"
)

func TestCheck_NolintOnLine(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b bool) bool {
	return a || //nolint:exprwrap // generated shape
		b
}
`,
		wantLines: nil,
	})
}

func TestCheck_NolintAboveAMultiBreakChainCoversEveryBreak(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(g func() bool, a, b, c bool) bool {
	//nolint:exprwrap
	return a ||
		b ||
		g() ||
		c
}
`,
		wantLines: nil,
	})
}

func TestCheck_NolintInsideANestedFuncLiteralLeavesTheOuterBreak(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(run func(func() bool) bool, a, b, c bool) bool {
	ok := run(func() bool {
		//nolint:exprwrap
		return a ||
			b
	}) &&
		c
	return ok
}
`,
		wantLines: []int{8},
	})
}

func TestCheck_TrailingNolintOnTheStartLine(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(g func(int) bool, ok bool) bool {
	x := g( //nolint:exprwrap
		1,
	) ||
		ok
	return x
}
`,
		wantLines: nil,
	})
}

func TestCheck_NolintOnTheBreakLine(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(g func(int) bool, ok bool) bool {
	x := g(
		1,
	) || //nolint:exprwrap
		ok
	return x
}
`,
		wantLines: nil,
	})
}

func TestCheck_NolintOnLineAbove(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b bool) bool {
	//nolint:exprwrap
	return a ||
		b
}
`,
		wantLines: nil,
	})
}
