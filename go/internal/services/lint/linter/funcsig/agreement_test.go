package funcsig

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/analysis"
)

func TestFuncsig_CheckerFixerAgreement_Collapsible(t *testing.T) {
	assertCheckerFixerAgree(t, `package foo

func Get(
	key string,
) string {
	return key
}
`)
}

func TestFuncsig_CheckerFixerAgreement_AlreadySingleLine(t *testing.T) {
	assertCheckerFixerAgree(t, `package foo

func Get(key string) string {
	return key
}
`)
}

func TestFuncsig_CheckerFixerAgreement_CommentInParams(t *testing.T) {
	assertCheckerFixerAgree(t, `package foo

func Get(
	// key comment
	key string,
) string {
	return key
}
`)
}

func TestFuncsig_CheckerFixerAgreement_NoBody(t *testing.T) {
	assertCheckerFixerAgree(t, `package foo

func Asm()
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
