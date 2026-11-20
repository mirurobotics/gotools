package collapse

import (
	"testing"
)

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
	candidates := findCandidates(t, src, 25)
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
	candidates := findCandidates(t, src, 26)
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
	candidates := findCandidates(t, src, 29)
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
	candidates := findCandidates(t, src, 27)
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
	candidates := findCandidates(t, src, 34)
	for _, c := range candidates {
		if c.Collapsed == "fn(arg)" {
			t.Error(
				"should not collapse fn(arg) when sibling with" +
					" multi-line value has a longer key",
			)
		}
	}
}
