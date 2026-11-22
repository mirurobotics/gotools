package exprwrap

import (
	"testing"
)

func TestCheck_StringConcatenationExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(name string) string {
	msg := "hello " + name +
		" and welcome"
	return msg
}
`,
		wantLines: nil,
	})
}

func TestCheck_StringConcatWithWrappedCallOperand(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(g func(bool) string, a, b bool) string {
	return "x" + g(a ||
		b)
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_StringConcatWithWrappedSelectorOperand(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(x interface{ String() string }) string {
	return "prefix" + x.
		String()
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_StringConcatWithWrappedIndexOperand(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(x []string) string {
	return "prefix" + x[1+
		2]
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_RawStringOperandExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: "package foo\n\nfunc f(y string) bool {\n" +
			"\tok := y == `a\nb`\n\treturn ok\n}\n",
		wantLines: nil,
	})
}

func TestCheck_MultiLineCallAndLiteralOperandsExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type T struct{ a, b int }

func f(g func(int, int) bool, t T, ok bool) bool {
	x := g(
		1,
		2,
	) || ok
	y := t == T{
		a: 1,
		b: 2,
	}
	return x && y
}
`,
		wantLines: nil,
	})
}

func TestCheck_StringChainWithBreakBeforeTheLiteral(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b string) string {
	s := a +
		b + "!"
	return s
}
`,
		wantLines: nil,
	})
}

func TestCheck_StringChainWithWrappedNonOperand(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(b, c string) string {
	return "a" + (b ==
		c)
}
`,
		wantLines: []int{4},
	})
}

func TestCheck_StringConcatWithParensAndNonLiterals(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b string) string {
	return (a + b + "c") +
		a
}
`,
		wantLines: nil,
	})
}

func TestCheck_NumericAdditionIsFlagged(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

func f(a, b int) int {
	return a + b +
		a
}
`,
		wantLines: []int{4},
	})
}
