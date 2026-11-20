package fieldassert

import (
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
	var got, other = build(), 1
	assert.Equal(t, 4, got.D)
	assert.Equal(t, 5, got.E)
	got.n, x = 1, 2
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
