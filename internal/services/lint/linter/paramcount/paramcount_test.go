package paramcount

import (
	"testing"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/linttest"
)

type checkCase struct {
	filename  string
	src       string
	maxParams int
	wantN     int
	wantMsg   string
}

func TestCheck_AtLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo` +
			`; func f(a, b, c, d, e int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_InitOverLimitExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo` +
			`; func init(a, b, c, d, e, f int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_OverLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo` +
			`; func f(a, b, c, d, e, f int) {}`,
		maxParams: 5,
		wantN:     1,
		wantMsg: "function f has 6 parameters" +
			" (limit 5); consider an options struct",
	})
}

func TestCheck_ContextExcluded(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo; import "context"` +
			`; func f(ctx context.Context, a, b, c, d, e int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_GroupedParamsCountedCorrectly(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo` +
			`; func f(a, b int, c string, d, e bool, f float64) {}`,
		maxParams: 5,
		wantN:     1,
		wantMsg: "function f has 6 parameters" +
			" (limit 5); consider an options struct",
	})
}

func TestCheck_MainExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename:  "",
		src:       `package foo; func main(a, b, c, d, e, f int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_InitExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename:  "",
		src:       `package foo; func init() {}`,
		maxParams: 0,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_UnderLimit(t *testing.T) {
	runCheckCase(t, checkCase{
		filename:  "",
		src:       `package foo; func f(a int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_MethodWithReceiver(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo; type S struct{}` +
			`; func (s S) f(a, b, c, d, e, g int) {}`,
		maxParams: 5,
		wantN:     1,
		wantMsg: "function f has 6 parameters" +
			" (limit 5); consider an options struct",
	})
}

func TestCheck_BenchmarkExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo; import "testing"` +
			`; func BenchmarkFoo(b *testing.B,` +
			` a, c, d, e, g int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_FuzzExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo; import "testing"` +
			`; func FuzzFoo(f *testing.F,` +
			` a, c, d, e, g int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_ExampleExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename:  "",
		src:       `package foo; func ExampleFoo(a, b, c, d, e, g int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_TestPrefixedWordNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename:  "",
		src:       `package foo; func Testable(a, b, c, d, e, g int) {}`,
		maxParams: 5,
		wantN:     1,
		wantMsg: "function Testable has 6 parameters" +
			" (limit 5); consider an options struct",
	})
}

func TestCheck_TestNamedMethodNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo; type S struct{}` +
			`; func (s S) TestFoo(a, b, c, d, e, g int) {}`,
		maxParams: 5,
		wantN:     1,
		wantMsg: "function TestFoo has 6 parameters" +
			" (limit 5); consider an options struct",
	})
}

func TestCheck_UnnamedParameters(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo` +
			`; func f(int, string, bool) {}`,
		maxParams: 2,
		wantN:     1,
		wantMsg: "function f has 3 parameters" +
			" (limit 2);" +
			" consider an options struct",
	})
}

func TestCheck_AliasedContextImport(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "",
		src: `package foo; import ctx "context"` +
			`; func f(c ctx.Context,` +
			` a, b, d, e, g int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_ServicesLintExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename: "internal/services/lint/check.go",
		src: `package lint` +
			`; func f(a, b, c, d, e, g int) {}`,
		maxParams: 5,
		wantN:     0,
		wantMsg:   "",
	})
}

func TestCheck_TestFileNotExempt(t *testing.T) {
	runCheckCase(t, checkCase{
		filename:  "foo_test.go",
		src:       "package foo\n\nfunc helper(a, b, c int) {}\n",
		maxParams: 2,
		wantN:     1,
		wantMsg: "function helper has 3 parameters (limit 2);" +
			" consider an options struct",
	})
}

func runCheckCase(t *testing.T, tc checkCase) {
	t.Helper()
	filename := tc.filename
	if filename == "" {
		filename = "test.go"
	}
	fset, file := linttest.Parse(t, filename, tc.src)
	diags := Check(fset, filename, file, tc.maxParams)
	linttest.AssertCount(t, diags, tc.wantN, tc.wantMsg)
}
