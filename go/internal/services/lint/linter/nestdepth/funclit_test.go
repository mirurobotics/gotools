package nestdepth

import (
	"testing"
)

func TestCheck_FuncLiteralNesting(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

var f = func() {
	if true {
		if true {
			_ = 1
		}
	}
}
`,
		maxDepth: 1,
		wantN:    1,
		wantMsg: "nesting depth is 2 (limit 1)" +
			" — consider early returns" +
			" or extracting a helper",
	})
}

func TestCheck_NestedFuncLiteral(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

var f = func() {
	g := func() {
		if true {
			if true {
				_ = 1
			}
		}
	}
	_ = g
}
`,
		maxDepth: 1,
		wantN:    1,
		wantMsg: "nesting depth is 2 (limit 1)" +
			" — consider early returns" +
			" or extracting a helper",
	})
}
