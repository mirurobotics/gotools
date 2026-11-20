package fieldassert

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

func TestCheck_UseOutsideAssertStartsNewGroup(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestW(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	got.Stop()
	assert.Equal(t, 3, got.C)
}

func TestX(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	fill(got)
	assert.Equal(t, 3, got.C)
}

func TestY(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	for got = range xs {
	}
	assert.Equal(t, 3, got.C)
}
`)
}

func TestCheck_ReadingTestifyArgKeepsGroup(t *testing.T) {
	assertGotFlagged(t, `func TestZ(t *testing.T) {
	got := build()
	assert.Equal(t, 1, got.A)
	assert.Len(t, string(got.Name), max(len(got.Items), int(int32(got.N))))
	assert.Equal(t, 2, got.B)
	err := other()
	assert.Equal(t, 3, got.C)
	_ = err
}
`, 13)
}

func TestCheck_SameFieldNameElsewhereKeepsGroup(t *testing.T) {
	fset, f := linttest.Parse(t, "foo_test.go", header+`func TestX(t *testing.T) {
	got := build()
	assert.Equal(t, 1, got.Meta.A)
	_ = other.Meta
	assert.Equal(t, 2, got.Meta.B)
	assert.Equal(t, 3, got.Meta.C)
}
`)
	diags := Check(fset, "foo_test.go", f)
	linttest.AssertLines(t, diags, []int{13}, wantMsg("got.Meta", "got"))
}

func TestCheck_ChangingTestifyArgStartsNewGroup(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestW(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	assert.NotSame(t, &got, prev)
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}

func TestX(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	require.NoError(t, got.Start())
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}

func TestY(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	assert.Eventually(t, func() bool { return got.Ready }, time.Second, 0)
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}

func TestZ(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	require.Positive(t, -got.N)
	assert.Len(t, string(got.Name), max(len(got.Items), int(int32(got.N))))
	assert.Equal(t, 2, got.B)
	assert.Equal(t, 3, got.C)
}
`, 37)
}

func TestCheck_HelperCallInTestifyArgStartsNewGroup(t *testing.T) {
	assertLines(t, "foo_test.go", `func TestX(t *testing.T) {
	var got T
	assert.Equal(t, 1, got.A)
	assert.Equal(t, 2, got.B)
	require.NoError(t, mutate(got))
	assert.Equal(t, 3, got.C)
}
`)
}
