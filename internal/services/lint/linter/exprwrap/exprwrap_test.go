package exprwrap

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		wantLines []int
	}{
		{
			name:      "single line",
			src:       "package foo\n\nfunc f(a, b bool) bool {\n\treturn a || b\n}\n",
			wantLines: nil,
		},
		{
			name: "boolean chain in assignment",
			src: `package foo

func f(a, b, c bool) {
	skip := a || b ||
		c
	_ = skip
}
`,
			wantLines: []int{4},
		},
		{
			name: "if condition",
			src: `package foo

func f(a, b bool) {
	if a &&
		b {
		return
	}
}
`,
			wantLines: []int{4},
		},
		{
			name: "return value",
			src: `package foo

func f(a, b bool) bool {
	return a ||
		b
}
`,
			wantLines: []int{4},
		},
		{
			name: "selector chain",
			src: `package foo

func f(x interface{ Y() int }) int {
	return x.
		Y()
}
`,
			wantLines: []int{4},
		},
		{
			name: "multi-line call arguments exempt",
			src: `package foo

func f(g func(int, int) int) int {
	return g(
		1,
		2,
	)
}
`,
			wantLines: nil,
		},
		{
			name: "wrapped call argument is flagged",
			src: `package foo

func f(g func(bool) bool, a, b bool) bool {
	return g(
		a ||
			b,
	)
}
`,
			wantLines: []int{5},
		},
		{
			name: "call chained after multi-line args exempt",
			src: `package foo

type T struct{}

func (T) Do() int { return 0 }

func mk(int) T { return T{} }

func f() int {
	return mk(
		1,
	).Do()
}
`,
			wantLines: nil,
		},
		{
			name: "table test composite literal exempt",
			src: `package foo

func f() {
	tests := []struct {
		name string
		n    int
	}{
		{name: "a", n: 1},
		{
			name: "b",
			n:    2,
		},
	}
	_ = tests
}
`,
			wantLines: nil,
		},
		{
			name: "wrapped composite value is flagged",
			src: `package foo

func f(a, b bool) {
	m := map[string]bool{
		"x": a ||
			b,
	}
	_ = m
}
`,
			wantLines: []int{5},
		},
		{
			name: "func literal exempt, body checked",
			src: `package foo

func f() func() bool {
	return func() bool {
		x := 1
		return x == 1 ||
			x == 2
	}
}
`,
			wantLines: []int{6},
		},
		{
			name: "string concatenation exempt",
			src: `package foo

func f(name string) string {
	msg := "hello " + name +
		" and welcome"
	return msg
}
`,
			wantLines: nil,
		},
		{
			name: "string concat with wrapped call operand",
			src: `package foo

func f(g func(bool) string, a, b bool) string {
	return "x" + g(a ||
		b)
}
`,
			wantLines: []int{4},
		},
		{
			name: "string concat with wrapped selector operand",
			src: `package foo

func f(x interface{ String() string }) string {
	return "prefix" + x.
		String()
}
`,
			wantLines: []int{4},
		},
		{
			name: "string concat with wrapped index operand",
			src: `package foo

func f(x []string) string {
	return "prefix" + x[1+
		2]
}
`,
			wantLines: []int{4},
		},
		{
			name: "raw string operand exempt",
			src: "package foo\n\nfunc f(y string) bool {\n" +
				"\tok := y == `a\nb`\n\treturn ok\n}\n",
			wantLines: nil,
		},
		{
			name: "multi-line call and literal operands exempt",
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
		},
		{
			name: "reported on the line of the break",
			src: `package foo

func f(g func(int, int) bool, ok bool) bool {
	x := g(
		1,
		2,
	) ||
		ok
	return x
}
`,
			wantLines: []int{7},
		},
		{
			name: "package-level var",
			src: `package foo

var a, b = true, false

var c = a ||
	b
`,
			wantLines: []int{5},
		},
		{
			name: "for, switch, case, send, incdec, range, go, defer",
			src: `package foo

func f(a, b bool, ch chan bool, m map[int]int, g func(bool), r func(bool) []int) {
	for a &&
		b {
	}
	switch a ||
		b {
	case a &&
		b:
	}
	ch <- a ||
		b
	m[1+
		1]++
	for range r(a ||
		b) {
	}
	go g(a ||
		b)
	defer g(a ||
		b)
}
`,
			wantLines: []int{4, 7, 9, 12, 14, 16, 19, 21},
		},
		{
			name: "string chain with break before the literal",
			src: `package foo

func f(a, b string) string {
	s := a +
		b + "!"
	return s
}
`,
			wantLines: nil,
		},
		{
			name: "string chain with wrapped non-+ operand",
			src: `package foo

func f(b, c string) string {
	return "a" + (b ==
		c)
}
`,
			wantLines: []int{4},
		},
		{
			name: "type-set union in type parameter list skipped",
			src: `package foo

func Sum[T ~int |
	~float64](xs []T) T {
	var t T
	return t
}

type Pair[T ~int |
	~string] struct{ a, b T }
`,
			wantLines: nil,
		},
		{
			name: "bitwise or outside a type set is flagged",
			src: `package foo

func f(a, b int) int {
	return a |
		b
}
`,
			wantLines: []int{4},
		},
		{
			name: "type-set union in interface skipped",
			src: `package foo

type Number interface {
	~int |
		~float64
}
`,
			wantLines: nil,
		},
		{
			name: "expression statement, bare for and switch",
			src: `package foo

// g does nothing.
func g(bool) {}

func f(a, b bool) {
	g(a)
	for {
		switch {
		default:
			return
		}
	}
}
`,
			wantLines: nil,
		},
		{
			name: "string concat with parens and non-literal operands",
			src: `package foo

func f(a, b string) string {
	return (a + b + "c") +
		a
}
`,
			wantLines: nil,
		},
		{
			name: "numeric addition is flagged",
			src: `package foo

func f(a, b int) int {
	return a + b +
		a
}
`,
			wantLines: []int{4},
		},
		{
			name: "nolint on line",
			src: `package foo

func f(a, b bool) bool {
	return a || //nolint:exprwrap // generated shape
		b
}
`,
			wantLines: nil,
		},
		{
			name: "chain with several breaks reports each",
			src: `package foo

func f(a, b, c bool) bool {
	return a ||
		b ||
		c
}
`,
			wantLines: []int{4, 5},
		},
		{
			name: "nolint above a multi-break chain covers every break",
			src: `package foo

func f(g func() bool, a, b, c bool) bool {
	//nolint:exprwrap
	return a ||
		b ||
		g() ||
		c
}
`,
			wantLines: nil,
		},
		{
			name: "nolint inside a nested func literal leaves the outer break",
			src: `package foo

func f(run func(func() bool) bool, a, b, c bool) bool {
	ok := run(func() bool {
		//nolint:exprwrap
		return a ||
			b
	}) &&
		c
	return ok
}
`,
			wantLines: []int{8},
		},
		{
			name: "trailing nolint on the start line",
			src: `package foo

func f(g func(int) bool, ok bool) bool {
	x := g( //nolint:exprwrap
		1,
	) ||
		ok
	return x
}
`,
			wantLines: nil,
		},
		{
			name: "nolint on the break line",
			src: `package foo

func f(g func(int) bool, ok bool) bool {
	x := g(
		1,
	) || //nolint:exprwrap
		ok
	return x
}
`,
			wantLines: nil,
		},
		{
			name: "nolint on line above",
			src: `package foo

func f(a, b bool) bool {
	//nolint:exprwrap
	return a ||
		b
}
`,
			wantLines: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "foo.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, "foo.go", f)
			if want := len(tt.wantLines); len(diags) != want {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), want, diags)
			}
			for i, d := range diags {
				if d.Line != tt.wantLines[i] {
					t.Errorf("diag %d line = %d, want %d", i, d.Line, tt.wantLines[i])
				}
				if d.Message != message {
					t.Errorf("message = %q, want %q", d.Message, message)
				}
			}
		})
	}
}
