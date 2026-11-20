package funcinline

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

func TestFuncInline_CheckerFixerAgreement_Inlineable(t *testing.T) {
	assertCheckerFixerAgree(t, `package foo

func f() int {
	return 42
}
`)
}

func TestFuncInline_CheckerFixerAgreement_AlreadySingleLine(t *testing.T) {
	assertCheckerFixerAgree(t, `package foo

func f() int { return 42 }
`)
}

func TestFuncInline_CheckerFixerAgreement_MultipleStatements(t *testing.T) {
	assertCheckerFixerAgree(t, `package foo

func f() int {
	x := 42
	return x
}
`)
}

func TestFuncInline_CheckerFixerAgreement_CommentInBody(t *testing.T) {
	assertCheckerFixerAgree(t, `package foo

func f() int {
	// comment
	return 42
}
`)
}

func assertCheckerFixerAgree(t *testing.T, src string) {
	t.Helper()
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	candidates := FindCandidates(fset, f, srcBytes, 88, 4)
	fixed := analysis.Fix(fset, srcBytes, candidates)
	changed := string(fixed) != src

	fsetCheck := token.NewFileSet()
	fCheck, _ := parser.ParseFile(fsetCheck, "test.go", srcBytes, parser.ParseComments)
	diags := Check(fsetCheck, "test.go", fCheck, srcBytes, 88, 4)
	hasDiags := len(diags) > 0

	if changed && !hasDiags {
		t.Error("fixer changed file but checker reported no diagnostics")
	}
	if !changed && hasDiags {
		t.Errorf(
			"fixer left file unchanged but checker reported diagnostics: %v",
			diags,
		)
	}
}
