package nestdepth

import (
	"go/parser"
	"go/token"
	"testing"
)

type checkCase struct {
	src      string
	maxDepth int
	wantN    int
	wantMsg  string
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	diags := Check(fset, "test.go", f, tt.maxDepth)
	if len(diags) != tt.wantN {
		t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
	}
	if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
		t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
	}
}

func TestCheck_AtLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	if true {
		for i := 0; i < 10; i++ {
			if true {
				switch {
				case true:
					x := 1
					_ = x
				}
			}
		}
	}
}
`,
		maxDepth: 4,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_OverLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	if true {
		for i := 0; i < 10; i++ {
			if true {
				switch {
				case true:
					if true {
						x := 1
						_ = x
					}
				}
			}
		}
	}
}
`,
		maxDepth: 4,
		wantN:    1,
		wantMsg: "nesting depth is 5 (limit 4)" +
			" — consider early returns or extracting a helper",
	})
}

func TestCheck_ForAndIfAtLimit2(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(xs []int) {
	for _, x := range xs {
		if x > 0 {
			_ = x
		}
	}
}
`,
		maxDepth: 2,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_NestedRangeLoopsOverLimit2(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(xss [][]int) {
	for _, xs := range xss {
		for _, x := range xs {
			if x > 0 {
				_ = x
			}
		}
	}
}
`,
		maxDepth: 2,
		wantN:    1,
		wantMsg: "nesting depth is 3 (limit 2)" +
			" — consider early returns or extracting a helper",
	})
}

func TestCheck_SwitchCaseBodyNoExtraDepth(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	switch x := 1; {
	case x > 0:
		if true {
			y := 2
			_ = y
		}
	}
}
`,
		maxDepth: 2,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_NestedIfAndFor(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	if true {
		for i := 0; i < 10; i++ {
			if true {
				x := 1
				_ = x
			}
		}
	}
}
`,
		maxDepth: 3,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_MultipleFunctionsIndependent(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	if true {
		x := 1
		_ = x
	}
}

func g() {
	if true {
		x := 1
		_ = x
	}
}
`,
		maxDepth: 1,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_SelectStmtCounts(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	if true {
		select {
		case <-make(chan int):
			if true {
				x := 1
				_ = x
			}
		}
	}
}
`,
		maxDepth: 3,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_RangeStmtCounts(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	for range []int{1} {
		for range []int{1} {
			for range []int{1} {
				x := 1
				_ = x
			}
		}
	}
}
`,
		maxDepth: 2,
		wantN:    1,
		wantMsg: "nesting depth is 3 (limit 2)" +
			" — consider early returns" +
			" or extracting a helper",
	})
}

func TestCheck_TypeSwitchCounts(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	var x interface{} = 1
	if true {
		switch x.(type) {
		case int:
			if true {
				_ = x
			}
		}
	}
}
`,
		maxDepth: 2,
		wantN:    1,
		wantMsg: "nesting depth is 3 (limit 2)" +
			" — consider early returns" +
			" or extracting a helper",
	})
}

func TestCheck_CommClauseBodyNoExtraDepth(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	ch := make(chan int, 1)
	ch <- 1
	select {
	case <-ch:
		if true {
			_ = 1
		}
	default:
		if true {
			_ = 2
		}
	}
}
`,
		maxDepth: 2,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_LabeledStatement(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
outer:
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			_ = i + j
			continue outer
		}
	}
}
`,
		maxDepth: 2,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_DisabledWhenMaxDepthZero(t *testing.T) {
	runCheckCase(t, checkCase{
		src:      `package foo; func f() { if true { if true { } } }`,
		maxDepth: 0,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_BareBlockStatement(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f() {
	{
		if true {
			if true {
				_ = 1
			}
		}
	}
}
`,
		maxDepth: 2,
		wantN:    0,
		wantMsg:  "",
	})
}
