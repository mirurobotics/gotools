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

func TestCollapse_CompositeLit(t *testing.T) {
	src := `package foo

func f() {
	x := []int{
		1,
		2,
		3,
	}
	_ = x
}
`
	want := `package foo

func f() {
	x := []int{1, 2, 3}
	_ = x
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_StructLit(t *testing.T) {
	src := `package foo

type S struct {
	A int
	B string
}

func f() {
	s := S{
		A: 1,
		B: "hello",
	}
	_ = s
}
`
	want := `package foo

type S struct {
	A int
	B string
}

func f() {
	s := S{A: 1, B: "hello"}
	_ = s
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
	f, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, []byte(src), 31, 4)
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
	f, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, []byte(src), 30, 4)
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
	f, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, []byte(src), 21, 4)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	fset2 := token.NewFileSet()
	f2, _ := parser.ParseFile(fset2, "test.go", []byte(src), parser.ParseComments)
	candidates2 := FindCandidates(fset2, f2, []byte(src), 20, 4)
	if len(candidates2) != 0 {
		t.Fatalf("expected 0 candidates with tight width, got %d", len(candidates2))
	}
}

func TestCollapse_SkipCommentInsideArgs(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1,
		// important comment
		arg2,
	)
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_SkipNolintDirective(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1, //nolint:errcheck
		arg2,
	)
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_SkipFuncLitArg(t *testing.T) {
	src := `package foo

func f() {
	foo(
		arg1,
		func() { return },
	)
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_SkipNestedMultiLineArg(t *testing.T) {
	src := `package foo

func f() {
	foo(
		bar(
			baz,
		),
		arg2,
	)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", []byte(src), parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, []byte(src), 88, 4)

	for _, c := range candidates {
		pos := fset.Position(c.Pos)
		if pos.Line == 4 {
			t.Error(
				"foo() should not be collapsed because" +
					" bar() arg spans multiple lines",
			)
		}
	}
}

func TestCollapse_SkipRawStringWithNewlines(t *testing.T) {
	src := "package foo\n\nfunc f() {\n\tfoo(\n\t\t`multi\nline`,\n\t\targ2,\n\t)\n}\n"
	assertFixResult(t, src, src)
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

func TestCollapse_CheckerFixerAgreement(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "collapsible call",
			src: `package foo

func f() {
	foo(
		arg1,
		arg2,
	)
}
`,
		},
		{
			name: "already single line",
			src: `package foo

func f() {
	foo(arg1, arg2)
}
`,
		},
		{
			name: "skip func lit",
			src: `package foo

func f() {
	foo(
		arg1,
		func() { return },
	)
}
`,
		},
		{
			name: "skip comment",
			src: `package foo

func f() {
	foo(
		arg1,
		// comment
		arg2,
	)
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

func TestCollapse_VariadicCall(t *testing.T) {
	src := `package foo

func f() {
	args := []string{"a", "b"}
	foo(
		arg1,
		args...,
	)
	_ = args
}
`
	want := `package foo

func f() {
	args := []string{"a", "b"}
	foo(arg1, args...)
	_ = args
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_EmptyArgs_NoCollapse(t *testing.T) {
	src := `package foo

func f() {
	foo()
}
`
	assertFixResult(t, src, src)
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

func TestCollapse_MapLit(t *testing.T) {
	src := `package foo

func f() {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	_ = m
}
`
	want := `package foo

func f() {
	m := map[string]int{"a": 1, "b": 2}
	_ = m
}
`
	assertFixResult(t, src, want)
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

func TestCollapse_TypelessCompositeLit(t *testing.T) {
	src := `package foo

func f() {
	x := [][]int{
		{
			4,
			5,
		},
	}
	_ = x
}
`
	want := `package foo

func f() {
	x := [][]int{
		{4, 5},
	}
	_ = x
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_VariadicSingleArg(t *testing.T) {
	src := `package foo

func f() {
	args := []string{"a"}
	foo(
		args...,
	)
	_ = args
}
`
	want := `package foo

func f() {
	args := []string{"a"}
	foo(args...)
	_ = args
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_RawStringInCompositeLit(t *testing.T) {
	src := "package foo\n\nfunc f() {\n\tx := []string{\n\t\t`multi\nline`,\n\t}\n\t_ = x\n}\n"
	assertFixResult(t, src, src)
}

func TestCollapse_EmptyArgs_MultiLine_NoCollapse(t *testing.T) {
	src := `package foo

func f() {
	foo(
	)
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_EmptyCompositeLit_MultiLine_NoCollapse(t *testing.T) {
	src := `package foo

func f() {
	x := []int{
	}
	_ = x
}
`
	assertFixResult(t, src, src)
}

func TestCollapse_RegularStringInArgs(t *testing.T) {
	src := `package foo

func f() {
	foo(
		"hello",
		"world",
	)
}
`
	want := `package foo

func f() {
	foo("hello", "world")
}
`
	assertFixResult(t, src, want)
}

func TestCollapse_BacktickStringWithoutNewlines(t *testing.T) {
	src := "package foo\n\nfunc f() {\n\tfoo(\n\t\t`hello`,\n\t\t`world`,\n\t)\n}\n"
	want := "package foo\n\nfunc f() {\n\tfoo(`hello`, `world`)\n}\n"
	assertFixResult(t, src, want)
}

func TestCollapse_AlignDelta_ExceedsLimit(t *testing.T) {
	src := `package foo

type S struct {
	LongName int
	B        int
}

func f() {
	_ = S{
		LongName: 1,
		B: fn(
			arg,
		),
	}
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 25, 4)
	for _, c := range candidates {
		if c.Collapsed == "fn(arg)" {
			t.Error(
				"should not collapse fn(arg) when gofmt" +
					" alignment would exceed limit",
			)
		}
	}
}

func TestCollapse_AlignDelta_WithinLimit(t *testing.T) {
	src := `package foo

type S struct {
	LongName int
	B        int
}

func f() {
	_ = S{
		LongName: 1,
		B: fn(
			arg,
		),
	}
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 26, 4)
	found := false
	for _, c := range candidates {
		if c.Collapsed == "fn(arg)" {
			found = true
		}
	}
	if !found {
		t.Error(
			"expected collapse when gofmt alignment" +
				" stays within limit",
		)
	}
}

func TestCollapse_AlignDelta_MapLit_ExceedsLimit(t *testing.T) {
	src := `package foo

func f() {
	_ = map[string]string{
		short():  fn1(),
		longerkey(): fn2(
			arg,
		),
	}
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 29, 4)
	for _, c := range candidates {
		if c.Collapsed == "fn2(arg)" {
			t.Error(
				"should not collapse fn2(arg) when" +
					" result exceeds limit",
			)
		}
	}
}

func TestCollapse_AlignDelta_UnaryExpr(t *testing.T) {
	src := `package foo

type S struct{ A int }

func f() {
	type T struct {
		LongField int
		B         *S
	}
	_ = T{
		LongField: 1,
		B: &S{
			A: 1,
		},
	}
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 27, 4)
	for _, c := range candidates {
		if c.Collapsed == "S{A: 1}" {
			t.Error(
				"should not collapse S{A: 1} when gofmt" +
					" alignment of &S{...} exceeds limit",
			)
		}
	}
}

func TestCollapse_AlignDelta_MultiLineSiblingValue(t *testing.T) {
	src := `package foo

type Other struct{ X int }
type S struct {
	VeryLongFieldName Other
	B                 int
}

func f() {
	_ = S{
		VeryLongFieldName: Other{
			X: 1,
		},
		B: fn(
			arg,
		),
	}
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 34, 4)
	for _, c := range candidates {
		if c.Collapsed == "fn(arg)" {
			t.Error(
				"should not collapse fn(arg) when sibling with" +
					" multi-line value has a longer key",
			)
		}
	}
}

func TestCollapse_TrailingCloseParen_ExceedsLimit(t *testing.T) {
	src := `package foo

func f() {
	foo([]int{
		1,
		2,
		3,
	})
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 22, 4)
	for _, c := range candidates {
		if c.Collapsed == "[]int{1, 2, 3}" {
			t.Error(
				"should not collapse []int{1, 2, 3} when trailing" +
					" ) exceeds limit",
			)
		}
	}
}

func TestCollapse_TrailingMethodChain_ExceedsLimit(t *testing.T) {
	src := `package foo

type S struct{ A int }

func f() S { return S{A: 0} }

func g() {
	f().Method(S{
		A: 1,
	}).Build()
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 30, 4)
	for _, c := range candidates {
		if c.Collapsed == "S{A: 1}" {
			t.Error(
				"should not collapse S{A: 1} when trailing" +
					" .Build() exceeds limit",
			)
		}
	}
}

func TestCollapse_TrailingCloseParen_Simple_ExceedsLimit(t *testing.T) {
	src := `package foo

type S struct{ A int }

func f() {
	foo(S{
		A: 1,
	})
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 15, 4)
	for _, c := range candidates {
		if c.Collapsed == "S{A: 1}" {
			t.Error(
				"should not collapse S{A: 1} when trailing" +
					" ) exceeds limit",
			)
		}
	}
}

func TestCollapse_TrailingCloseParen_WithinLimit(t *testing.T) {
	src := `package foo

func f() {
	foo([]int{
		1,
		2,
		3,
	})
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 23, 4)
	found := false
	for _, c := range candidates {
		if c.Collapsed == "[]int{1, 2, 3}" {
			found = true
		}
	}
	if !found {
		t.Error("expected collapse candidate when within limit including trailing )")
	}
}

func TestCollapse_TrailingMethodChain_WithinLimit(t *testing.T) {
	src := `package foo

type S struct{ A int }

func f() S { return S{A: 0} }

func g() {
	f().Method(S{
		A: 1,
	}).Build()
}
`
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 31, 4)
	found := false
	for _, c := range candidates {
		if c.Collapsed == "S{A: 1}" {
			found = true
		}
	}
	if !found {
		t.Error(
			"expected collapse candidate when within" +
				" limit including trailing .Build()",
		)
	}
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
	srcBytes := []byte(src)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", srcBytes, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	candidates := FindCandidates(fset, f, srcBytes, 88, 4)
	return string(analysis.Fix(fset, srcBytes, candidates))
}
