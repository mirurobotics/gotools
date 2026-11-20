package fieldassert

import (
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

func TestCheck_ShadowingAssertObjectFlagged(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	assert := assert.New(t)
	got := build()
	assert.Equal(1, got.A)
	assert.Equal(2, got.B)
	assert.EqualExportedValuesf(3, got.C, "msg")
	t.Run("sub", func(t *testing.T) {
		assert.Equal(1, got.A)
		assert.Equal(2, got.B)
		assert.Equal(3, got.C)
	})
}
`, 14, 18)
}

func TestCheck_RepeatedFieldAllowed(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	c := newCounter()
	c.Inc()
	assert.Equal(t, 1, c.Count)
	c.Inc()
	assert.Equal(t, 2, c.Count)
	c.Inc()
	assert.Equal(t, 3, c.Count)
	assert.Equal(t, 0, c.Errors)
}
`)
}

func TestCheck_ReassignedValueAllowed(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	got := build()
	assert.Equal(t, 1, got.A)
	got = build()
	assert.Equal(t, 2, got.B)
	got.n++
	assert.Equal(t, 3, got.C)
	assert.Equal(t, 3, got.G)
	got.n += 1
	assert.Equal(t, 4, got.D)
	assert.Equal(t, 5, got.E)
	got.n, x = 1, 2
	*ptr() = 1
	assert.Equal(t, 6, got.F)
}
`)
}

func TestCheck_HelperClosuresAndMethodsAllowed(t *testing.T) {
	assertLines(t, "foo_test.go", `func checkThing(t *testing.T, got Thing) {
	t.Run("fields", func(t *testing.T) {
		assert.Equal(t, 1, got.A)
		assert.Equal(t, 2, got.B)
		assert.Equal(t, 3, got.C)
	})
}

func (f *fixture) assertState(t *testing.T) {
	assert.Equal(t, 1, f.state.A)
	assert.Equal(t, 2, f.state.B)
	assert.Equal(t, 3, f.state.C)
}
`)
}

func TestCheck_HelperWithGetterBaseAllowed(t *testing.T) {
	assertLines(t, "foo_test.go", `func checkThing(t *testing.T, got *Thing) {
	assert.Equal(t, 1, got.Meta().A)
	assert.Equal(t, 2, got.Meta().B)
	assert.Equal(t, 3, got.Meta().C)
}
`)
}

func TestCheck_AssertObjectScopedToItsBlock(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	got := build()
	if ok {
		assert := assert.New(t)
		_ = assert
	}
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`, 17)
}

func TestCheck_ShadowedHelperParamFlagged(t *testing.T) {
	assertLines(t, "foo_test.go", `func checkThing(t *testing.T, got Thing) {
	t.Run("x", func(t *testing.T) {
		got := build()
		for i, v := range xs {
			_, _ = i, v
		}
		var z int
		_ = z
		assert.Equal(t, 1, got.A)
		assert.Equal(t, 2, got.B)
		assert.Equal(t, 3, got.C)
	})
}
`, 19)
}

func TestCheck_VersionNamedPackageIsConstant(t *testing.T) {
	src := `package foo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
)

func TestX(t *testing.T) {
	assert.Equal(t, got.A, v1.PodRunning)
	assert.Equal(t, got.B, v1.PodFailed)
	assert.Equal(t, got.C, v1.PodPending)
}
`
	fset, f := linttest.Parse(t, "foo_test.go", src)
	linttest.AssertLines(t, Check(fset, "foo_test.go", f), []int{11},
		"3 asserts compare fields of got one at a time; compare the whole value"+
			" in one assert, or move the checks into a helper that takes got")
}

func TestCheck_IndexChangeStartsNewGroup(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	got, i := build(), 0
	assert.Equal(t, 1, got[i].A)
	i++
	assert.Equal(t, 2, got[i].B)
	i++
	assert.Equal(t, 3, got[i].C)
}
`)
}

func TestCheck_PackageAndGenericCallRootsHaveNoHelperHint(t *testing.T) {
	fset, f := linttest.Parse(t, "foo_test.go", header+`func TestX(t *testing.T) {
	assert.Equal(t, 1, http.Load().A)
	assert.Equal(t, 2, http.Load().B)
	assert.Equal(t, 3, http.Load().C)
	assert.Equal(t, 1, build[int]().A)
	assert.Equal(t, 2, build[int]().B)
	assert.Equal(t, 3, build[int]().C)
	assert.Equal(t, 1, pair[int, string]().A)
	assert.Equal(t, 2, pair[int, string]().B)
	assert.Equal(t, 3, pair[int, string]().C)
}
`)
	diags := Check(fset, "foo_test.go", f)
	linttest.AssertLines(t, diags, []int{12, 15, 18}, "")
	for _, d := range diags {
		if strings.Contains(d.Message, "helper") {
			t.Errorf("unexpected helper hint: %q", d.Message)
		}
	}
}

func TestCheck_NestedReassignStartsNewGroup(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	got := build()
	assert.Equal(t, 1, got.A)
	if ok {
		got = build()
	}
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`)
}

func TestCheck_NamedResultNotAHelperParam(t *testing.T) {
	assertLines(t, "foo_test.go", `func buildAndCheck(t *testing.T) (got Thing) {
	got = build()
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
	return got
}

func noResults(t *testing.T) {}
`, 13)
}

func TestCheck_LocalShadowingImportIsAValue(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	http := build()
	assert.Equal(t, 1, http.A)
	assert.Equal(t, 2, http.B)
	assert.Equal(t, 3, http.C)
}
`, 13)
}

func TestCheck_VarAssertObjectFlagged(t *testing.T) {
	assertGotThree(t, `func TestX(t *testing.T) {
	var a = assert.New(t)
	var b, c int
	got := build()
	a.Equal(1, got.A)
	a.Equal(2, got.B)
	a.Equal(3, got.C)
	b.Equal(1, got.D)
	_ = c
}

func helper(a *assert.Assertions, r require.Assertions, n *other.Assertions, m *x) {
	got := build()
	a.Equal(1, got.A)
	a.Equal(2, got.B)
	a.Equal(3, got.C)
	r.Equal(4, got.D)
	n.Equal(4, got.D)
	m.Equal(4, got.D)
}
`, 15, 24)
}

func TestCheck_MultiAssignAssertObjectFlagged(t *testing.T) {
	assertGotThree(t, `func TestY(t *testing.T) {
	a, r := assert.New(t), require.New(t)
	x, y := build(), 2
	var v, w = 1, require.New(t)
	got := build()
	a.Equal(1, got.A)
	r.Equal(2, got.B)
	w.Equal(3, got.C)
	x.Equal(4, got.D)
	TestY.Equal(4, got.D)
	_, _ = y, v
}
`, 16)
}

func TestCheck_ClosureLocalDoesNotCloseOuterGroup(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	got := build()
	assert.Equal(t, 1, got.A)
	t.Run("x", func(t *testing.T) {
		got := other()
		_ = got
	})
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`, 13)
}

func TestCheck_LocalNamedLikeTestifyIsNotPackage(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	assert := makeChecker()
	got := build()
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`)
}

func TestCheck_AliasedImportBaseIsNotPackage(t *testing.T) {
	src := `package foo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	cfgpkg "example.com/app/config"
)

func TestX(t *testing.T) {
	_ = cfgpkg.X
	assert.Equal(t, 1, config.A)
	assert.Equal(t, 2, config.B)
	assert.Equal(t, 3, config.C)
}
`
	fset, f := linttest.Parse(t, "foo_test.go", src)
	linttest.AssertLines(t, Check(fset, "foo_test.go", f), []int{12}, "")
}

func TestCheck_AddressTakenOrRangeAssignStartsNewGroup(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}

func TestY(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	for got = range xs {
	}
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}

func TestZ(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	_ = -got.A
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`, 30)
}
