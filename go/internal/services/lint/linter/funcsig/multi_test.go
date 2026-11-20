package funcsig

import (
	"testing"
)

func TestFuncsig_Idempotent(t *testing.T) {
	src := `package foo

func Get(
	key string,
) string {
	return key
}
`
	want := `package foo

func Get(key string) string {
	return key
}
`
	result := fixSource(t, src)
	if result != want {
		t.Fatalf("first fix: got:\n%s\nwant:\n%s", result, want)
	}
	result2 := fixSource(t, result)
	if result2 != want {
		t.Errorf("second fix not idempotent: got:\n%s\nwant:\n%s", result2, want)
	}
}

func TestFuncsig_MultipleFunctions(t *testing.T) {
	src := `package foo

func A(
	x int,
) int {
	return x
}

func B(
	y string,
) string {
	return y
}
`
	want := `package foo

func A(x int) int {
	return x
}

func B(y string) string {
	return y
}
`
	assertFixResult(t, src, want)
}

func TestFuncsig_MixedEligibility(t *testing.T) {
	assertFixResult(t, `package foo

func Simple(
	x int,
) int {
	return x
}

func HasComment(
	// comment
	x int,
) int {
	return x
}

func AlsoSimple(
	y string,
) string {
	return y
}
`, `package foo

func Simple(x int) int {
	return x
}

func HasComment(
	// comment
	x int,
) int {
	return x
}

func AlsoSimple(y string) string {
	return y
}
`)
}
