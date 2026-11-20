package funcinline

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/analysis"
)

func TestFuncInline_SingleReturn(t *testing.T) {
	src := `package foo

func Code() int {
	return 42
}
`
	want := `package foo

func Code() int { return 42 }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_MethodWithReceiver(t *testing.T) {
	src := `package foo

type Err struct{}

func (e Err) Code() int {
	return 42
}
`
	want := `package foo

type Err struct{}

func (e Err) Code() int { return 42 }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_SingleAssignment(t *testing.T) {
	src := `package foo

var x int

func set() {
	x = 42
}
`
	want := `package foo

var x int

func set() { x = 42 }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_SingleExpressionStmt(t *testing.T) {
	src := `package foo

func die() {
	panic("fatal")
}
`
	want := `package foo

func die() { panic("fatal") }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_NoReturnType(t *testing.T) {
	src := `package foo

var x int

func bump() {
	x++
}
`
	want := `package foo

var x int

func bump() { x++ }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_PointerReceiver(t *testing.T) {
	src := `package foo

type S struct{ val int }

func (s *S) Set(v int) {
	s.val = v
}
`
	want := `package foo

type S struct{ val int }

func (s *S) Set(v int) { s.val = v }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_AlreadySingleLine(t *testing.T) {
	src := `package foo

func f() int { return 42 }
`
	assertFixResult(t, src, src)
}

func TestFuncInline_ExactlyAtLimit(t *testing.T) {
	src := `package foo

func f() int {
	return 42
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, []byte(src), 26, 4)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate at limit, got %d", len(candidates))
	}
}

func TestFuncInline_OnePastLimit(t *testing.T) {
	src := `package foo

func f() int {
	return 42
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, []byte(src), 25, 4)
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates past limit, got %d", len(candidates))
	}
}

func TestFuncInline_MultipleFunctions(t *testing.T) {
	src := `package foo

func a() int {
	return 1
}

func b() int {
	return 2
}

func c() int {
	return 3
}
`
	want := `package foo

func a() int { return 1 }

func b() int { return 2 }

func c() int { return 3 }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_MixedEligibility(t *testing.T) {
	src := `package foo

func simple() int {
	return 1
}

func complex() int {
	x := 1
	return x
}

func alsoSimple() string {
	return "hello"
}
`
	want := `package foo

func simple() int { return 1 }

func complex() int {
	x := 1
	return x
}

func alsoSimple() string { return "hello" }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_Idempotent(t *testing.T) {
	src := `package foo

func f() int {
	return 42
}
`
	want := `package foo

func f() int { return 42 }
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

func TestFuncInline_DeferStatement(t *testing.T) {
	src := `package foo

func cleanup() {
	defer close()
}
`
	want := `package foo

func cleanup() { defer close() }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_GoStatement(t *testing.T) {
	src := `package foo

func spawn() {
	go run()
}
`
	want := `package foo

func spawn() { go run() }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_SendStatement(t *testing.T) {
	src := `package foo

func send(ch chan int) {
	ch <- 42
}
`
	want := `package foo

func send(ch chan int) { ch <- 42 }
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
