package nofmt

import (
	"go/ast"
	"go/parser"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename string
	src      string
	wantN    int
	wantMsg  string
}

func TestCheck_FmtPrintlnInLibrary(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src: `package foo; import "fmt"` +
			`; func f() { fmt.Println("hi") }`,
		wantN:   1,
		wantMsg: "fmt.Println" + messageSuffix,
	})
}

func TestCheck_FmtPrintfInLibrary(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src: `package foo; import "fmt"` +
			`; func f() { fmt.Printf("%s", "hi") }`,
		wantN:   1,
		wantMsg: "fmt.Printf" + messageSuffix,
	})
}

func TestCheck_FmtPrintlnInPackageMain(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "main.go",
		src: `package main; import "fmt"` +
			`; func run() { fmt.Println("hi") }`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_PrintlnBuiltinInLibrary(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src: `package foo` +
			`; func f() { println("hi") }`,
		wantN:   1,
		wantMsg: "println" + messageSuffix,
	})
}

func TestCheck_PrintBuiltinInLibrary(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src: `package foo` +
			`; func f() { print("hi") }`,
		wantN:   1,
		wantMsg: "print" + messageSuffix,
	})
}

func TestCheck_LogPrintfInLibrary(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src: `package foo; import "log"` +
			`; func f() { log.Printf("hi") }`,
		wantN:   1,
		wantMsg: "log.Printf" + messageSuffix,
	})
}

func TestCheck_LogFatalInLibrary(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src: `package foo; import "log"` +
			`; func f() { log.Fatal("hi") }`,
		wantN:   1,
		wantMsg: "log.Fatal" + messageSuffix,
	})
}

func TestCheck_TestFileNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar_test.go",
		src: `package foo; import "fmt"` +
			`; func f() { fmt.Println("hi") }`,
		wantN:   1,
		wantMsg: "fmt.Println" + messageSuffix,
	})
}

func TestCheck_ServicesLintExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "internal/services/lint/linter/check.go",
		src: `package linter; import "fmt"` +
			`; func f() { fmt.Println("hi") }`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_CmdDirectoryExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "cmd/server/run.go",
		src: `package server; import "fmt"` +
			`; func f() { fmt.Println("hi") }`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_FunctionLiteralCallNoDiagnostic(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src:      `package foo; func f() { func() {}() }`,
		wantN:    0,
		wantMsg:  "",
	})
}

func TestCheck_FmtFprintfNotBanned(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src: `package foo; import "fmt"` +
			`; func f() { fmt.Fprintf(nil, "x") }`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintNofmtExemptsCall(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src: "package foo\nimport \"fmt\"\n" +
			"func f() { fmt.Println(\"hi\") " +
			"//nolint:nofmt\n}",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NolintOnDifferentLineDoesNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src: "package foo\nimport \"fmt\"\n" +
			"//nolint:nofmt\n" +
			"func f() { fmt.Println(\"hi\") }",
		wantN:   1,
		wantMsg: "fmt.Println" + messageSuffix,
	})
}

func TestCallName(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"fmt.Println()", "fmt.Println"},
		{"println()", "println"},
		{"a.fmt.Println()", ""},
		{"func() {}()", ""},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			e, err := parser.ParseExpr(tt.expr)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			call, ok := e.(*ast.CallExpr)
			if !ok {
				t.Fatalf("%q is not a call", tt.expr)
			}
			if got := callName(call); got != tt.want {
				t.Errorf("callName(%q) = %q, want %q", tt.expr, got, tt.want)
			}
		})
	}
}

func TestCheck_ExampleInTestFileExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar_test.go",
		src: `package foo; import "fmt"` +
			`; func ExampleF() { fmt.Println("hi") }; func Example() { println() }`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_HelperBesideExampleFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar_test.go",
		src: `package foo; import "fmt"` +
			`; func ExampleF() { fmt.Println() }; func helper() { fmt.Println() }`,
		wantN:   1,
		wantMsg: "fmt.Println" + messageSuffix,
	})
}

func TestCheck_ExampleOutsideTestFileFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar.go",
		src:      `package foo; import "fmt"; func ExampleF() { fmt.Println("hi") }`,
		wantN:    1,
		wantMsg:  "fmt.Println" + messageSuffix,
	})
}

func TestCheck_ExampleLookalikesInTestFileFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "pkg/foo/bar_test.go",
		src: `package foo; import "fmt"; type s struct{}` +
			`; func (s) ExampleM() { fmt.Println("hi") }` +
			`; func Examplehelper() { fmt.Println("hi") }`,
		wantN:   2,
		wantMsg: "fmt.Println" + messageSuffix,
	})
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset, f := linttest.Parse(t, tt.filename, tt.src)
	linttest.AssertCount(t, Check(fset, tt.filename, f), tt.wantN, tt.wantMsg)
}
