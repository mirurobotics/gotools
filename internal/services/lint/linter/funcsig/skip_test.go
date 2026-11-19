package funcsig

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestFuncsig_SkipAlreadySingleLine(t *testing.T) {
	src := `package foo

func Get(key string) string {
	return key
}
`
	assertFixResult(t, src, src)
}

func TestFuncsig_SkipNoBody(t *testing.T) {
	src := `package foo

func Asm()
`
	assertFixResult(t, src, src)
}

func TestFuncsig_SkipCommentsInParams(t *testing.T) {
	src := `package foo

func Create(
	// the name
	name string,
	// the age
	age int,
) error {
	return nil
}
`
	assertFixResult(t, src, src)
}

func TestFuncsig_SkipMultiLineFieldType(t *testing.T) {
	src := `package foo

func Apply(
	fn func(
		int,
	) string,
) string {
	return fn(0)
}
`
	assertFixResult(t, src, src)
}

func TestFuncsig_SkipTooWide(t *testing.T) {
	src := `package foo

func VeryLongFunctionName(
	parameter string,
) string {
	return parameter
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 51, 4)
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates (too wide), got %d", len(candidates))
	}
}

func TestFuncsig_SkipTrailingCommentAfterBrace(t *testing.T) {
	src := `package foo

func VeryLongFunctionName(
	parameter string,
) string { // inline comment
	return parameter
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 69, 4)
	if len(candidates) != 0 {
		t.Fatalf(
			"expected 0 candidates (trailing comment exceeds limit), got %d",
			len(candidates),
		)
	}
}

func TestFuncsig_TrailingCommentWithinLimit(t *testing.T) {
	src := `package foo

func VeryLongFunctionName(
	parameter string,
) string { // inline comment
	return parameter
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 70, 4)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate (within limit), got %d", len(candidates))
	}
}
