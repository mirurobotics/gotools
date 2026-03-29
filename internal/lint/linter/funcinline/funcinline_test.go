package funcinline

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
)

// --- basic inlining tests ---

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

// --- exclusion tests ---

func TestFuncInline_SkipMultipleStatements(t *testing.T) {
	src := `package foo

func f() {
	x := 1
	_ = x
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_SkipEmptyBody(t *testing.T) {
	src := `package foo

func f() {
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_SkipBodyWithComment(t *testing.T) {
	src := `package foo

func f() int {
	// important
	return 42
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_SkipMultiLineStatement(t *testing.T) {
	src := `package foo

type S struct{ A, B int }

func f() S {
	return S{
		A: 1,
		B: 2,
	}
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_SkipFuncLitInBody(t *testing.T) {
	src := `package foo

func f() func() {
	return func() {}
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_AlreadySingleLine(t *testing.T) {
	src := `package foo

func f() int { return 42 }
`
	assertFixResult(t, src, src)
}

// --- width boundary tests ---

func TestFuncInline_ExactlyAtLimit(t *testing.T) {
	// "func f() int { return 42 }" = 26 chars
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

// --- multiple functions ---

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

// --- idempotency ---

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

// --- checker/fixer agreement ---

func TestFuncInline_CheckerFixerAgreement(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "inlineable",
			src: `package foo

func f() int {
	return 42
}
`,
		},
		{
			name: "already single line",
			src: `package foo

func f() int { return 42 }
`,
		},
		{
			name: "multiple statements",
			src: `package foo

func f() int {
	x := 42
	return x
}
`,
		},
		{
			name: "comment in body",
			src: `package foo

func f() int {
	// comment
	return 42
}
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

// --- nil body (external/assembly declaration) ---

func TestFuncInline_SkipNilBody(t *testing.T) {
	src := `package foo

func Asm()
`
	assertFixResult(t, src, src)
}

// --- statement type variety ---

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

// --- function literal inlining ---

func TestFuncInline_FuncLitInArgument(t *testing.T) {
	src := `package foo

func f() []string {
	return convert(
		items,
		func(x Item) string {
			return string(x)
		},
	)
}
`
	want := `package foo

func f() []string {
	return convert(
		items,
		func(x Item) string { return string(x) },
	)
}
`
	assertFixResult(t, src, want)
}

func TestFuncInline_FuncLitAlreadySingleLine(t *testing.T) {
	src := `package foo

func f() {
	apply(func(x int) int { return x + 1 })
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_FuncLitMultipleStatements(t *testing.T) {
	src := `package foo

func f() {
	apply(func(x int) int {
		y := x + 1
		return y
	})
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_FuncLitWithComment(t *testing.T) {
	src := `package foo

func f() {
	apply(func(x int) int {
		// important
		return x
	})
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_FuncLitNestedFuncLit(t *testing.T) {
	src := `package foo

func f() {
	apply(func() func() {
		return func() {}
	})
}
`
	assertFixResult(t, src, src)
}

func TestFuncInline_FuncLitAssignedToVar(t *testing.T) {
	src := `package foo

var fn = func(x int) int {
	return x * 2
}
`
	want := `package foo

var fn = func(x int) int { return x * 2 }
`
	assertFixResult(t, src, want)
}

func TestFuncInline_BothDeclAndLit(t *testing.T) {
	src := `package foo

func simple() int {
	return 1
}

func wrapper() {
	apply(func(x int) int {
		return x
	})
}
`
	want := `package foo

func simple() int { return 1 }

func wrapper() {
	apply(func(x int) int { return x })
}
`
	assertFixResult(t, src, want)
}

// --- func lit width with trailing content ---

func TestFuncInline_FuncLitTrailingContentExceedsLimit(t *testing.T) {
	// The func lit alone (indent + collapsed) fits within the limit,
	// but the trailing ", other)" after } pushes the line over.
	src := `package foo

func f() {
	call(func(x int) int {
		return x
	}, other)
}
`
	// collapsed = "func(x int) int { return x }" = 28 chars
	// indent to func = "\t" + "call(" = 4 + 5 = 9
	// trailing = ", other)" = 8
	// total = 9 + 28 + 8 = 45
	// Set limit to 44 so trailing pushes it over.
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 44, 4)
	if len(candidates) != 0 {
		t.Fatalf(
			"expected 0 candidates (trailing content exceeds limit), got %d",
			len(candidates),
		)
	}
}

func TestFuncInline_FuncLitTrailingContentWithinLimit(t *testing.T) {
	src := `package foo

func f() {
	call(func(x int) int {
		return x
	}, other)
}
`
	// total = 9 + 28 + 8 = 45; set limit to 45 so it fits exactly.
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 45, 4)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate at exact limit, got %d", len(candidates))
	}
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
