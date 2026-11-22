package collapse

import (
	"testing"
)

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
	candidates := findCandidates(t, src, 22)
	for _, cand := range candidates {
		if cand.Collapsed == "[]int{1, 2, 3}" {
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
	candidates := findCandidates(t, src, 30)
	for _, cand := range candidates {
		if cand.Collapsed == "S{A: 1}" {
			t.Error(
				"should not collapse S{A: 1} when trailing" +
					" .Build() exceeds limit",
			)
		}
	}
}

func TestCollapse_TrailingCloseParen_Simple_OverLimit(t *testing.T) {
	src := `package foo

type S struct{ A int }

func f() {
	foo(S{
		A: 1,
	})
}
`
	candidates := findCandidates(t, src, 15)
	for _, cand := range candidates {
		if cand.Collapsed == "S{A: 1}" {
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
	candidates := findCandidates(t, src, 23)
	found := false
	for _, cand := range candidates {
		if cand.Collapsed == "[]int{1, 2, 3}" {
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
	candidates := findCandidates(t, src, 31)
	found := false
	for _, cand := range candidates {
		if cand.Collapsed == "S{A: 1}" {
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
