package collapse

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

func TestCollapse_SingleArg(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1,
	)
}
`
	want := `package foo

func f() {
	foo(arg1)
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_MultiArg(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1,
		arg2,
		arg3,
	)
}
`
	want := `package foo

func f() {
	foo(arg1, arg2, arg3)
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_MethodCall(t *testing.T) {
	src := `package foo

func f() {
	obj.Method(
		arg1,
		arg2,
	)
}
`
	want := `package foo

func f() {
	obj.Method(arg1, arg2)
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_ExactlyAtLimit(t *testing.T) {
	src := `package foo

func f() {
	foo(
		aaaaaaaaaa,
		bbbbbbbbbb,
	)
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, file, []byte(src), 31, 4)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate at limit, got %d", len(candidates))
	}
}

func TestCollapse_OnePastLimit(t *testing.T) {
	src := `package foo

func f() {
	foo(
		aaaaaaaaaa,
		bbbbbbbbbb,
	)
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, file, []byte(src), 30, 4)
	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates past limit, got %d", len(candidates))
	}
}

func TestCollapse_DeeplyIndented(t *testing.T) {
	src := `package foo

func f() {
	if true {
		if true {
			foo(
				a,
				b,
			)
		}
	}
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, file, []byte(src), 21, 4)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	fset2 := token.NewFileSet()
	astFile, _ := parser.ParseFile(fset2, "test.go", []byte(src), parser.ParseComments)
	candidates2 := FindCandidates(fset2, astFile, []byte(src), 20, 4)
	if len(candidates2) != 0 {
		t.Fatalf("expected 0 candidates with tight width, got %d", len(candidates2))
	}
}

func TestCollapse_Idempotent(t *testing.T) {
	src := `package foo

func f() {
	foo(arg1, arg2)
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_FixThenFixAgain(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1,
		arg2,
	)
}
`
	want := `package foo

func f() {
	foo(arg1, arg2)
}
`
	result := fixSource(t, src)
	if result != want {
		t.Fatalf(
			"first fix produced unexpected result.\ngot:\n%s\nwant:\n%s",
			result, want,
		)
	}
	result2 := fixSource(t, result)
	if result2 != want {
		t.Errorf("second fix was not idempotent.\ngot:\n%s\nwant:\n%s", result2, want)
	}
}

func TestCollapse_MultipleCalls(t *testing.T) {
	src := `package foo

func f() {
	foo(
		a,
		b,
	)
	bar(
		c,
		d,
	)
}
`
	want := `package foo

func f() {
	foo(a, b)
	bar(c, d)
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_ChainedMethod(t *testing.T) {
	src := `package foo

func f() {
	obj.Foo().Bar(
		a,
		b,
	)
}
`
	want := `package foo

func f() {
	obj.Foo().Bar(a, b)
}
`
	assertFixResult(t, src, want)
}

func assertFixResult(t *testing.T, src, want string) {
	got := fixSource(t, src)
	if got != want {
		t.Errorf(
			"collapse fix produced unexpected result.\ngot:\n%s\nwant:\n%s",
			got, want,
		)
	}
}

func fixSource(t *testing.T, src string) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, file, []byte(src), 88, 4)
	return string(analysis.Fix(fset, []byte(src), candidates))
}

func TestVisualWidth(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		tabWidth int
		want     int
	}{
		{"empty", "", 4, 0},
		{"spaces", "    ", 4, 4},
		{"single tab", "\t", 4, 4},
		{"tab after content", "ab\t", 4, 4},
		{"mixed tabs and spaces", "\t  \t", 4, 8},
		{"tab mid word", "a\tb", 4, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := analysis.VisualWidth(tc.input, tc.tabWidth)
			if got != tc.want {
				t.Errorf(
					"VisualWidth(%q, %d) = %d, want %d",
					tc.input, tc.tabWidth, got, tc.want,
				)
			}
		})
	}
}

func TestLineStartOffset(t *testing.T) {
	src := []byte("first\nsecond\nthird")
	cases := []struct {
		name string
		off  int
		want int
	}{
		{"first line", 3, 0},
		{"start of second line", 6, 6},
		{"mid second line", 9, 6},
		{"offset zero", 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := analysis.LineStartOffset(src, tc.off)
			if got != tc.want {
				t.Errorf("LineStartOffset(src, %d) = %d, want %d", tc.off, got, tc.want)
			}
		})
	}
}

func findCandidates(t *testing.T, src string, limit int) []analysis.Candidate {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return FindCandidates(fset, file, []byte(src), limit, 4)
}
