package namelen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

func TestCheck_LocalDefine(t *testing.T) {
	fits, over := boundaryNames("local", MaxLocalNameLen)
	src := fmt.Sprintf(`package foo

func run() {
	%s := 1
	%s := 2
}
`, fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxLocalNameLen))
}

func TestCheck_LocalVar(t *testing.T) {
	fits, over := boundaryNames("local", MaxLocalNameLen)
	src := fmt.Sprintf(`package foo

func run() {
	var %s int
	var %s int
}
`, fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxLocalNameLen))
}

func TestCheck_RangeVars(t *testing.T) {
	fits, over := boundaryNames("local", MaxLocalNameLen)
	src := fmt.Sprintf(`package foo

func run(items []int) {
	for %s, %s := range items {
	}
}
`, fits, over)
	assertReported(t, src, 4, wantMsg(over, MaxLocalNameLen))
}

func TestCheck_Params(t *testing.T) {
	fits, over := boundaryNames("param", MaxLocalNameLen)
	src := fmt.Sprintf("package foo\n\nfunc run(%s, %s int) {}\n", fits, over)
	assertReported(t, src, 3, wantMsg(over, MaxLocalNameLen))
}

func TestCheck_NamedResults(t *testing.T) {
	fits, over := boundaryNames("result", MaxLocalNameLen)
	src := fmt.Sprintf(`package foo

func run() (%s, %s int) { return 0, 0 }
`, fits, over)
	assertReported(t, src, 3, wantMsg(over, MaxLocalNameLen))
}

func TestCheck_FuncLitParams(t *testing.T) {
	fits, over := boundaryNames("param", MaxLocalNameLen)
	src := fmt.Sprintf("package foo\n\nvar handler = func(%s, %s int) {}\n", fits, over)
	assertReported(t, src, 3, wantMsg(over, MaxLocalNameLen))
}

func TestCheck_MultiByteNameCountsRunes(t *testing.T) {
	fits, over := strings.Repeat("ä", 20), strings.Repeat("ä", 21)
	src := fmt.Sprintf(`package foo

func run() {
	%s := 1
	%s := 2
}
`, fits, over)
	assertReported(t, src, 5, fmt.Sprintf("name %q is 21 chars (limit 20)", over))
}

func TestCheck_Message(t *testing.T) {
	src := "package foo\n\nfunc run() {\n\tabcdefghijklmnopqrstu := 1\n}\n"
	assertReported(t, src, 4, `name "abcdefghijklmnopqrstu" is 21 chars (limit 20)`)
}

func TestCheck_Nolint(t *testing.T) {
	_, over := boundaryNames("local", MaxLocalNameLen)
	src := fmt.Sprintf(`package foo

func run() {
	%s := 1 //nolint:namelen // generated name
}
`, over)
	fset, file := linttest.Parse(t, "foo.go", src)
	linttest.AssertLines(t, Check(fset, "foo.go", file), nil, "")
}

func TestCheck_TestFuncInNonTestFile(t *testing.T) {
	fits, over := boundaryNames("Test", MaxTestNameLen)
	src := fmt.Sprintf("package foo\n\nfunc %s() {}\n\nfunc %s() {}\n", fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxTestNameLen))
}

func TestCheck_TestablePrefixGetsNameLimit(t *testing.T) {
	fits, over := boundaryNames("Testable", MaxNameLen)
	src := fmt.Sprintf("package foo\n\nfunc %s() {}\n\nfunc %s() {}\n", fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxNameLen))
}

func TestCheck_PackageVar(t *testing.T) {
	fits, over := boundaryNames("pkgVar", MaxNameLen)
	src := fmt.Sprintf("package foo\n\nvar %s = 1\n\nvar %s = 2\n", fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxNameLen))
}

func TestCheck_Const(t *testing.T) {
	fits, over := boundaryNames("Const", MaxNameLen)
	src := fmt.Sprintf("package foo\n\nconst %s = 1\n\nconst %s = 2\n", fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxNameLen))
}

func TestCheck_Type(t *testing.T) {
	fits, over := boundaryNames("Type", MaxNameLen)
	src := fmt.Sprintf("package foo\n\ntype %s int\n\ntype %s int\n", fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxNameLen))
}

func TestCheck_StructField(t *testing.T) {
	fits, over := boundaryNames("field", MaxNameLen)
	src := fmt.Sprintf(`package foo

type point struct {
	%s int
	%s int
}
`, fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxNameLen))
}

func TestCheck_Func(t *testing.T) {
	fits, over := boundaryNames("run", MaxNameLen)
	src := fmt.Sprintf("package foo\n\nfunc %s() {}\n\nfunc %s() {}\n", fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxNameLen))
}

func TestCheck_Method(t *testing.T) {
	fits, over := boundaryNames("Get", MaxNameLen)
	src := fmt.Sprintf(`package foo

type repo struct{}

func (r repo) %s() {}

func (r repo) %s() {}
`, fits, over)
	assertReported(t, src, 7, wantMsg(over, MaxNameLen))
}

func TestCheck_IfaceMethod(t *testing.T) {
	fits, over := boundaryNames("Get", MaxNameLen)
	src := fmt.Sprintf(`package foo

type store interface {
	%s()
	%s()
}
`, fits, over)
	assertReported(t, src, 5, wantMsg(over, MaxNameLen))
}

func boundaryNames(prefix string, limit int) (fits, over string) {
	fits = prefix + strings.Repeat("X", limit-len(prefix))
	return fits, fits + "X"
}

func assertReported(t *testing.T, src string, line int, msg string) {
	t.Helper()
	fset, file := linttest.Parse(t, "foo.go", src)
	linttest.AssertLines(t, Check(fset, "foo.go", file), []int{line}, msg)
}

func wantMsg(name string, limit int) string {
	return fmt.Sprintf("name %q is %d chars (limit %d)", name, len(name), limit)
}
