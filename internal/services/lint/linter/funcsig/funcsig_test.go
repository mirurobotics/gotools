package funcsig

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

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

func TestFuncsig_ExactlyAtLimit(t *testing.T) {
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
