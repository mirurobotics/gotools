package fieldassert

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

func TestCheck_ThreeFieldAssertsFlagged(t *testing.T) {
	fset, f := linttest.Parse(t, "foo_test.go", header+`func TestX(t *testing.T) {
	got := build()
	assert.Equal(t, 1, got.A)
	assert.Equal(t, "b", got.B)
	require.Equal(t, true, got.C)
}
`)
	linttest.AssertLines(t, Check(fset, "foo_test.go", f), []int{13},
		"3 asserts compare fields of got one at a time; compare the whole value"+
			" in one assert, or move the checks into a helper that takes got")
}

func TestCheck_TwoFieldAssertsAllowed(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	got := build()
	assert.Equal(t, 1, got.A)
	assert.Equal(t, "b", got.B)
}
`)
}

func TestCheck_NonTestFileSkipped(t *testing.T) {
	assertLines(t, "foo.go", `func TestX(t *testing.T) {
	got := build()
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`)
}

func TestCheck_NoTestifyImportSkipped(t *testing.T) {
	src := `package foo

import "testing"

func TestX(t *testing.T) {
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`
	fset, f := linttest.Parse(t, "foo_test.go", src)
	linttest.AssertCount(t, Check(fset, "foo_test.go", f), 0, "")
}

func TestCheck_HelperTakingValueAllowed(t *testing.T) {
	assertLines(t, "foo_test.go", `func checkThing(t *testing.T, got Thing) {
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}

func TestX(t *testing.T) {
	check := func(got *Thing) {
		assert.Equal(t, 1, got.Meta.A)
		assert.Equal(t, 2, got.Meta.B)
		assert.Equal(t, 3, got.Meta.C)
	}
	check(build())
}
`)
}

func TestCheck_HelperBuildingValueFlagged(t *testing.T) {
	assertLines(t, "foo_test.go", `func checkThing(t *testing.T, in string) {
	got := build(in)
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`, 13)
}

func TestCheck_ExpectedFieldsGroupByActual(t *testing.T) {
	fset, f := linttest.Parse(t, "foo_test.go", header+`func TestX(t *testing.T) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := build(tc.in)
			assert.Equal(t, tc.wantA, got.A)
			assert.Equal(t, tc.wantB, got.B)
			assert.Equal(t, tc.wantC, got.C)
			assert.Equal(t, tc.wantD, other)
		})
	}
}
`)
	diags := Check(fset, "foo_test.go", f)
	linttest.AssertLines(t, diags, []int{15}, wantMsg("got", "got"))
}

func TestCheck_SwappedArgsCounted(t *testing.T) {
	bodies := []string{
		"got.A, 1)\n\tassert.Exactlyf(t, got.B, 2, \"msg\")\n\tassert.EqualValues(t, got.C, 3)",
		"got.A, -1)\n\tassert.Equal(t, got.B, -2)\n\tassert.Equal(t, got.C, -3)",
		"got.A, true)\n\tassert.Equal(t, got.B, false)\n\tassert.Equal(t, got.C, (nil))",
		"got.A, T{X: 1})\n\tassert.Equal(t, got.B, []int{1})\n\tassert.Equal(t, got.C, http.MethodGet)",
	}
	for _, b := range bodies {
		assertLines(t, "foo_test.go", "func TestX(t *testing.T) {\n\tgot := build()\n"+
			"\tassert.Equal(t, "+b+"\n}\n", 13)
	}
}

func TestCheck_ExpectedFieldsWithoutStructAllowed(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	code, body, err := do(tc.req)
	assert.Equal(t, tc.wantCode, code)
	assert.Equal(t, tc.wantBody, body)
	assert.Equal(t, tc.wantErr, err)
	assert.Equal(t, tc.wantName, got.GetName())
	assert.Equal(t, tc.wantKind, -code)
	assert.Equal(t, tc.wantOK, false)
	assert.Equal(t, tc.wantID, ident)
	assert.Equal(t, code, 200)
	assert.Equal(t, tc.wantIDs, []string{a.ID})
	assert.Equal(t, tc.wantMap, T{X: a.ID})
}
`)
}

func TestCheck_PackageSelectorsIgnored(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	assert.Equal(t, 200, http.StatusOK)
	assert.Equal(t, 404, http.StatusNotFound)
	assert.Equal(t, 500, http.StatusInternalServerError)
}
`)
}

func TestCheck_NonEqualityAssertsIgnored(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	got := build()
	assert.Contains(t, got.A, "x")
	assert.Len(t, got.B, 2)
	assert.NotEqual(t, 1, got.C)
	assert.Equal(t, 1)
	other.Equal(t, 1, got.D)
	(build()).Equal(t, 1, got.E)
	_ = got.F
	<-done
}
`)
}

func TestCheck_SeparateBasesAndBlocks(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	a, b := build(), build()
	assert.Equal(t, 1, a.X)
	assert.Equal(t, 2, b.X)
	assert.Equal(t, 3, a.Y)
	if ok {
		assert.Equal(t, 4, a.Z)
	}
	switch {
	case ok:
		assert.Equal(t, 1, s[0].A)
		assert.Equal(t, 2, s[0].B)
		assert.Equal(t, 3, (s[0]).C)
	}
	select {
	case <-ch:
		assert.Equal(t, 1, (*p).A)
		assert.Equal(t, 2, (*p).B)
		assert.Equal(t, 3, (*p).C)
	}
}
`, 21, 27)
}

func TestCheck_AssertObjectFlagged(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)
	x, y := 1, 2
	got := build()
	a.Equal(1, got.A)
	r.Equal(2, got.B)
	a.Equal(3, got.C)
	a.Equal(3)
	_, _ = x, y
}
`, 16)
}

func TestCheck_AliasedImportFlagged(t *testing.T) {
	src := `package foo

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"
)

func TestX(t *testing.T) {
	got := build()
	tassert.Equal(t, 1, got.A)
	tassert.Equal(t, 2, got.B)
	tassert.Equal(t, 3, got.C)
	o1 := other.New(t)
	o2 := New(t)
	o3 := tassert.Other(t)
	o4 := x
	o5 := (tassert).New(t)
	o1.Equal(1, got.D)
	o2.Equal(1, got.D)
	o3.Equal(1, got.D)
	o4.Equal(1, got.D)
	o5.Equal(1, got.D)
}
`
	fset, f := linttest.Parse(t, "foo_test.go", src)
	diags := Check(fset, "foo_test.go", f)
	linttest.AssertLines(t, diags, []int{12}, wantMsg("got", "got"))
}

func TestCheck_CallRootAndOddBases(t *testing.T) {
	fset, f := linttest.Parse(t, "foo_test.go", header+`func TestX(t *testing.T) {
	assert.Equal(t, 1, build().A)
	assert.Equal(t, 2, build().B)
	assert.Equal(t, 3, build().C)
	assert.Equal(t, 1, []T{}[0].A)
	assert.Equal(t, 2, []T{}[0].B)
	assert.Equal(t, 3, []T{}[0].C)
}
`)
	diags := Check(fset, "foo_test.go", f)
	linttest.AssertLines(t, diags, []int{12, 15}, "")
	if diags[0].Message != wantMsg("build()", "") {
		t.Errorf("message = %q", diags[0].Message)
	}
}

func TestCheck_Nolint(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	got := build()
	assert.Equal(t, 1, got.A) //nolint:fieldassert
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`)
}

func TestCheck_ShadowStartsNewGroup(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	got := build()
	t.Run("x", func(t *testing.T) {
		assert.Equal(t, 1, got.A)
		got := got.Child
		assert.Equal(t, 2, got.B)
		assert.Equal(t, 3, got.C)
	})
}
`)
}
