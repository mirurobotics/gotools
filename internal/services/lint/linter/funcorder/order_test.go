package funcorder

import (
	"strings"
	"testing"
)

func TestCheck_HelpersFollowCallOrder(t *testing.T) {
	assertMessages(t, `package p

func Generate() { validateArgs(); _ = checksum() }

func Verify() bool { return checksum() == 0 }

func validateArgs() { validatePrefix(); validateSize() }

func validatePrefix() {}

func validateSize() {}

func checksum() int { return 0 }
`,
		`"validateArgs" should be directly below "Generate"`,
		`"checksum" should be directly below "Verify"`,
	)
}

func TestFix_HelpersFollowCallOrder(t *testing.T) {
	src := []string{
		"func Generate() { validateArgs(); _ = checksum() }\n",
		"func Verify() bool { return checksum() == 0 }\n",
		"func validateArgs() { validatePrefix(); validateSize() }\n",
		"func validatePrefix() {}\n", "func validateSize() {}\n",
		"func checksum() int { return 0 }\n",
	}
	want := []string{src[0], src[2], src[3], src[4], src[1], src[5]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestCheck_BlockMovesWithFunctionBelowIt(t *testing.T) {
	assertMessages(t, "package p\n\nfunc Run() { f() }\n\nfunc g() {}\n\n"+
		"const c = 1\n\nfunc f() {}\n",
		`"f" should be directly below "Run"`)
	assertMessages(t, "package p\n\nfunc Run() { f() }\n\nfunc g() {}\n\n"+
		"type K int\n\nfunc f(K) {}\n",
		`"f" should be directly below "Run"`)
}

func TestFix_BlockMovesWithFunctionBelowIt(t *testing.T) {
	src := []string{
		"func a() {}\n", "const c = 1\n", "func b() {}\n", "func Run() { b(); a() }\n",
	}
	want := []string{src[3], src[1], src[2], src[0]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestFix_BlockStaysToKeepVarInitOrder(t *testing.T) {
	src := []string{
		"func f() {}\n", "var a = record(\"a\")\n", "func g() {}\n",
		"var b = record(\"b\")\n", "func h() {}\n", "func Run() { h(); g() }\n",
	}
	want := []string{src[1], src[3], src[5], src[4], src[2], src[0]}
	assertFix(t, "package p\n\n"+strings.Join(src, "\n"),
		"package p\n\n"+strings.Join(want, "\n"))
}

func TestWalk_NothingReady(t *testing.T) {
	var got []int
	never := func(int) bool { return false }
	none := func(int) []int { return nil }
	walk([]int{1, 2}, never, none, func(i int, _ bool) { got = append(got, i) })
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("got %v, want [1 2]", got)
	}
}

func TestCheck_CycleMemberFollowsItsCaller(t *testing.T) {
	assertMessages(t, `package p

func b() { a() }

func Run() { a() }

func a() { b() }
`,
		`"b" should be directly below "a"`,
	)
	assertMessages(t, `package p

func b() { a() }

func a() { b() }

func Run() {}
`)
}
