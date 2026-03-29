package funcsig

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
)

// --- basic collapsing tests ---

func TestFuncsig_SingleParam(t *testing.T) {
	src := `package foo

func NewScope(
	dbScope Scope,
) Scope {
	return dbScope
}
`
	want := `package foo

func NewScope(dbScope Scope) Scope {
	return dbScope
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_MultiParam(t *testing.T) {
	src := `package foo

func Create(
	name string,
	age int,
	active bool,
) error {
	return nil
}
`
	want := `package foo

func Create(name string, age int, active bool) error {
	return nil
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_Receiver(t *testing.T) {
	src := `package foo

type S struct{}

func (s S) Method(
	x int,
) int {
	return x
}
`
	want := `package foo

type S struct{}

func (s S) Method(x int) int {
	return x
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_PointerReceiver(t *testing.T) {
	src := `package foo

type S struct{ val int }

func (s *S) Set(
	val int,
) {
	s.val = val
}
`
	want := `package foo

type S struct{ val int }

func (s *S) Set(val int) {
	s.val = val
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_NamedReturns(t *testing.T) {
	src := `package foo

func Parse(
	input string,
) (result int, err error) {
	return 0, nil
}
`
	want := `package foo

func Parse(input string) (result int, err error) {
	return 0, nil
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_MultiReturns(t *testing.T) {
	src := `package foo

func Split(
	s string,
) (string, string) {
	return s, s
}
`
	want := `package foo

func Split(s string) (string, string) {
	return s, s
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_Variadic(t *testing.T) {
	src := `package foo

func Log(
	format string,
	args ...interface{},
) {
	_ = format
	_ = args
}
`
	want := `package foo

func Log(format string, args ...interface{}) {
	_ = format
	_ = args
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_FuncLit(t *testing.T) {
	src := `package foo

var fn = func(
	x int,
	y int,
) int {
	return x + y
}
`
	want := `package foo

var fn = func(x int, y int) int {
	return x + y
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_Generics(t *testing.T) {
	src := `package foo

func Map[T any, U any](
	items []T,
	fn func(T) U,
) []U {
	return nil
}
`
	want := `package foo

func Map[T any, U any](items []T, fn func(T) U) []U {
	return nil
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_MultiNameSameType(t *testing.T) {
	src := `package foo

func Add(
	x, y int,
) int {
	return x + y
}
`
	want := `package foo

func Add(x, y int) int {
	return x + y
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_NoParams(t *testing.T) {
	src := `package foo

func Get(
) string {
	return "hello"
}
`
	want := `package foo

func Get() string {
	return "hello"
}
`
	assertFixResult(t, src, want)
}

// --- width boundary tests ---

func TestFuncsig_ExactlyAtLimit(t *testing.T) {
	// "func Get(key string) string {" = 29 chars
	src := `package foo

func Get(
	key string,
) string {
	return key
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 29, 4)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate at limit, got %d", len(candidates))
	}
}

func TestFuncsig_OnePastLimit(t *testing.T) {
	src := `package foo

func Get(
	key string,
) string {
	return key
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 28, 4)
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates past limit, got %d", len(candidates))
	}
}

// --- exclusion tests ---

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
	// Collapsed: "func VeryLongFunctionName(parameter string) string {" = 52 chars
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
	// Collapsed: "func VeryLongFunctionName(parameter string) string {"
	// = 52 chars + trailing " // inline comment" = 18 chars → total 70.
	// Set limit to 69 so trailing comment pushes it over.
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
	// total = 70 (see above), set limit to 70.
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

// --- idempotency ---

func TestFuncsig_Idempotent(t *testing.T) {
	src := `package foo

func Get(
	key string,
) string {
	return key
}
`
	want := `package foo

func Get(key string) string {
	return key
}
`
	result := fixSource(t, src)
	if result != want {
		t.Fatalf("first fix: got:\n%s\nwant:\n%s", result, want)
	}
	result2 := fixSource(t, result)
	if result2 != want {
		t.Errorf("second fix not idempotent: got:\n%s\nwant:\n%s", result2, want)
	}
}

// --- checker/fixer agreement ---

func TestFuncsig_CheckerFixerAgreement(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "collapsible",
			src: `package foo

func Get(
	key string,
) string {
	return key
}
`,
		},
		{
			name: "already single line",
			src: `package foo

func Get(key string) string {
	return key
}
`,
		},
		{
			name: "comment in params",
			src: `package foo

func Get(
	// key comment
	key string,
) string {
	return key
}
`,
		},
		{
			name: "no body",
			src: `package foo

func Asm()
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srcBytes := []byte(tc.src)
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}

			candidates := FindCandidates(fset, f, srcBytes, 88, 4)
			fixed := analysis.Fix(fset, srcBytes, candidates)
			changed := string(fixed) != tc.src

			fsetCheck := token.NewFileSet()
			fCheck, _ := parser.ParseFile(
				fsetCheck, "test.go", srcBytes,
				parser.ParseComments,
			)
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
		})
	}
}

// --- multiple functions ---

func TestFuncsig_MultipleFunctions(t *testing.T) {
	src := `package foo

func A(
	x int,
) int {
	return x
}

func B(
	y string,
) string {
	return y
}
`
	want := `package foo

func A(x int) int {
	return x
}

func B(y string) string {
	return y
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_MixedEligibility(t *testing.T) {
	src := `package foo

func Simple(
	x int,
) int {
	return x
}

func HasComment(
	// comment
	x int,
) int {
	return x
}

func AlsoSimple(
	y string,
) string {
	return y
}
`
	want := `package foo

func Simple(x int) int {
	return x
}

func HasComment(
	// comment
	x int,
) int {
	return x
}

func AlsoSimple(y string) string {
	return y
}
`
	assertFixResult(t, src, want)
}

// --- helpers ---

func assertFixResult(t *testing.T, src, want string) {
	got := fixSource(t, src)
	if got != want {
		t.Errorf("fix produced unexpected result.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func fixSource(t *testing.T, src string) string {
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 88, 4)
	return string(analysis.Fix(fset, srcBytes, candidates))
}
