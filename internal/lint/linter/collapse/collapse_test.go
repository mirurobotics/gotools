package collapse

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
)

// --- basic collapsing tests ---

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

// --- width boundary tests ---

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

// --- indentation tests ---

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

// --- exclusion tests ---

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

// --- idempotency tests ---

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

// --- checker/fixer agreement tests ---

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

// --- variadic calls ---

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

// --- utility function tests ---

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

// --- typeless composite literal ---

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

// --- empty items on multi-line delimiters ---

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

// --- string literals in args ---

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

// --- gofmt alignment tests ---

func TestCollapse_AlignDelta_ExceedsLimit(t *testing.T) {
	// indent: 2 tabs (8) + "B: " (3) = 11.
	// alignDelta: LongName(8) - B(1) = 7.
	// collapsed: "fn(arg)" = 7. trailing: "," = 1.
	// Total = 11 + 7 + 7 + 1 = 26. Set limit to 25.
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
	// Total with alignment = 26, so limit 26 should allow collapse.
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
	// Map literal with call expression keys of different lengths.
	// Simulates the deployment_config_instance.go bug.
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
	// collapsed = "fn2(arg)" = 8 chars.
	// current indent: 2 tabs (8) + "longerkey(): " (13) = 21.
	// alignDelta: 0 (longerkey is already the longest
	// single-line sibling key is "short()" = 7 vs
	// "longerkey()" = 11; current is longest).
	// Wait, "short()" value is single-line so
	// short key width = 7. longerkey key width = 11.
	// Current key IS the longest → delta = 0.
	// trailing = "," = 1. Total = 21 + 0 + 8 + 1 = 30.
	// Set limit to 29.
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
	// Value is &Struct{...} — the & wraps the composite lit.
	// The alignment delta should still be computed.
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
	// collapsed = "S{A: 1}" = 7 chars.
	// current indent: 2 tabs (8) + "B: &" (4) = 12.
	// alignDelta: LongField(9) - B(1) = 8.
	// trailing = "," = 1. Total = 12 + 8 + 7 + 1 = 28.
	// Set limit to 27. Should NOT collapse.
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
	// Sibling "VeryLongFieldName" has a multi-line value. gofmt still
	// aligns keys, so the delta must include it.
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
	// collapsed = "fn(arg)" = 7 chars.
	// indent: 2 tabs (8) + "B: " (3) = 11.
	// alignDelta: VeryLongFieldName(17) - B(1) = 16.
	// trailing: "," = 1. Total = 11 + 16 + 7 + 1 = 35.
	// Set limit to 34 so alignment pushes it over.
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

// --- trailing text tests ---

func TestCollapse_TrailingCloseParen_ExceedsLimit(t *testing.T) {
	// Inner composite lit as argument — trailing ")" from outer call.
	// Line after collapse: "\tfoo([]int{1, 2, 3})"
	// indent=4, "foo("=4, "[]int{1, 2, 3}"=14, ")"=1 → total=23
	// Set limit to 22 so trailing ) pushes it over.
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
	// Collapsed form of S{A: 1} = "S{A: 1}" (7 chars)
	// On the line: "	f().Method(S{A: 1}).Build()"
	// indent = 4, "f().Method(" = 11, "S{A: 1}" = 7, ").Build()" = 9
	// total = 4 + 11 + 7 + 9 = 31
	// Set limit to 30 so trailing .Build() pushes it over.
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
	// Collapsed: "S{A: 1}" (7 chars)
	// Line: "\tfoo(S{A: 1})"
	// indent = 4, "foo(" = 4, "S{A: 1}" = 7, ")" = 1
	// total = 4 + 4 + 7 + 1 = 16
	// Set limit to 15 so trailing ) pushes it over.
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
	// Same setup as ExceedsLimit but limit is 23, so it fits.
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
	// total = 31 (see ExceedsLimit test), set limit to 31.
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

// --- helpers ---

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
