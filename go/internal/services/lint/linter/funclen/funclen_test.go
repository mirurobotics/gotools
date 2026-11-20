package funclen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename string
	src      string
	maxLen   int
	wantN    int
	wantMsg  string
}

func TestCheck_DisabledWhenLimitZero(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			strings.Repeat("\tx := 1\n", 100) + "}\n",
		maxLen:  0,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_UnderLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      "package foo\n\nfunc f() {\n\tx := 1\n\t_ = x\n}\n",
		maxLen:   50,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_AtExactlyLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			strings.Repeat("\tx := 1\n", 5) + "}\n",
		maxLen:  5,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_OverLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			strings.Repeat("\tx := 1\n", 6) + "}\n",
		maxLen:  5,
		wantMsg: "function f is 6 lines (limit 5)",
		wantN:   1,
	})
}

func TestCheck_BlankLinesNotCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      "package foo\n\nfunc f() {\n\tx := 1\n\n\n\ty := 2\n}\n",
		maxLen:   2,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_CommentLinesNotCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			"\t// comment\n\tx := 1\n" +
			"\t// another comment\n\ty := 2\n}\n",
		maxLen:  2,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_FunctionLiteralChecked(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nvar f = func() {\n" +
			strings.Repeat("\tx := 1\n", 4) + "}\n",
		maxLen:  3,
		wantN:   1,
		wantMsg: "anonymous function is 4 lines (limit 3)",
	})
}

func TestCheck_EmptyFunction(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      "package foo\n\nfunc f() {\n}\n",
		maxLen:   50,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_NilBodyFuncDecl(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src:      "package foo\n\nfunc Foo()\n",
		maxLen:   1,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_FuncLiteralUnderLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nvar f = func() {\n" +
			strings.Repeat("\tx := 1\n", 2) + "}\n",
		maxLen:  5,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TestHelperNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "tests/helpers/foo.go",
		src: "package helpers\n\nfunc f() {\n" +
			strings.Repeat("\tx := 1\n", 100) +
			"}\n",
		maxLen:  5,
		wantN:   1,
		wantMsg: "function f is 100 lines (limit 5)",
	})
}

func TestCheck_TestFileNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo_test.go",
		src: "package foo\n\nfunc f() {\n" +
			strings.Repeat("\tx := 1\n", 100) +
			"}\n",
		maxLen:  5,
		wantN:   1,
		wantMsg: "function f is 100 lines (limit 5)",
	})
}

func TestCheck_NolintFunclenOnFuncLineExemptsFuncDecl(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: "package foo\n\n" +
			"//nolint:funclen\n" +
			"func f() {\n" +
			strings.Repeat("\tx := 1\n", 6) + "}\n",
		maxLen:  5,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintFunclenInlineOnFuncLineExemptsFuncDecl(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: "package foo\n\n" +
			"func f() { //nolint:funclen\n" +
			strings.Repeat("\tx := 1\n", 6) + "}\n",
		maxLen:  5,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintFunclenOnFuncLiteralLineExemptsFuncLit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: "package foo\n\n" +
			"//nolint:funclen\n" +
			"var f = func() {\n" +
			strings.Repeat("\tx := 1\n", 6) + "}\n",
		maxLen:  5,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintOtherRuleDoesNotExemptFuncDecl(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "foo.go",
		src: "package foo\n\n" +
			"//nolint:mutableglobal\n" +
			"func f() {\n" +
			strings.Repeat("\tx := 1\n", 6) + "}\n",
		maxLen:  5,
		wantN:   1,
		wantMsg: "function f is 6 lines (limit 5)",
	})
}

func TestFuncBody_NilLitBody(t *testing.T) {
	if _, _, ok := funcBody(&ast.FuncLit{Type: nil, Body: nil}); ok {
		t.Error("funcBody accepted a func literal without a body")
	}
}

func TestCountLines_ShortSource(t *testing.T) {
	src := "package p\n\nfunc F() {\n\tx := 1\n\t_ = x\n}\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := f.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decl is %T, want *ast.FuncDecl", f.Decls[0])
	}
	body := fn.Body
	lines := [][]byte{[]byte("package p"), nil, []byte("func F() {"), []byte("x := 1")}
	if got := countLines(fset, body, lines); got != 1 {
		t.Errorf("countLines = %d, want 1", got)
	}
}

func TestCheck_MultiLineLiteralCountsAsOneLine(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			"\ttests := []struct {\n\t\tname string\n\t}{\n" +
			strings.Repeat("\t\t{\n\t\t\tname: \"a\",\n\t\t},\n", 20) +
			"\t}\n\t_ = tests\n}\n",
		maxLen:  2,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_LinesAroundLiteralStillCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			"\tx := []int{\n\t\t1,\n\t\t2,\n\t}\n" +
			strings.Repeat("\t_ = x\n", 3) + "}\n",
		maxLen:  3,
		wantN:   1,
		wantMsg: "function f is 4 lines (limit 3)",
	})
}

func TestCheck_FuncLiteralInsideLiteralStillChecked(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			"\tfns := []func(){\n\t\tfunc() {\n" +
			strings.Repeat("\t\t\tx := 1\n", 4) +
			"\t\t},\n\t}\n\t_ = fns\n}\n",
		maxLen:  3,
		wantN:   1,
		wantMsg: "anonymous function is 4 lines (limit 3)",
	})
}

func TestCheck_CodeAfterLiteralOnClosingLineCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			"\tif err := g(T{\n\t\tA: 1,\n\t}); err != nil {\n" +
			"\t\treturn\n\t}\n}\n",
		maxLen:  3,
		wantN:   1,
		wantMsg: "function f is 4 lines (limit 3)",
	})
}

func TestCheck_PunctuationAfterLiteralNotCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			"\tg(xs, []int{\n\t\t1,\n\t}...) // spread\r\n" +
			"\th(T{\n\t\tA: 1,\n\t}, // note\n\t)\n}\n",
		maxLen:  3,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_LiteralStartingOnClosingLineCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			"\tg(A{\n\t\tX: 1,\n\t}, B{\n\t\tY: 2,\n\t})\n}\n",
		maxLen:  1,
		wantN:   1,
		wantMsg: "function f is 2 lines (limit 1)",
	})
}

func TestCheck_TrailingCommentOnVarLiteralNotCounted(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: "package foo\n\nfunc f() {\n" +
			"\tvar x = []int{\n\t\t1,\n\t} // cases\n\t_ = x\n}\n",
		maxLen:  2,
		wantN:   0,
		wantMsg: "",
	})
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	filename := tt.filename
	if filename == "" {
		filename = "test.go"
	}
	fset, f := linttest.Parse(t, filename, tt.src)
	diags := Check(fset, filename, f, []byte(tt.src), tt.maxLen)
	linttest.AssertCount(t, diags, tt.wantN, tt.wantMsg)
}
