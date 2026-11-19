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

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	filename := tt.filename
	if filename == "" {
		filename = "test.go"
	}
	fset, f := linttest.Parse(t, filename, tt.src)
	diags := Check(fset, filename, f, tt.maxParams)
	linttest.AssertCount(t, diags, tt.wantN, tt.wantMsg)
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
		src:       `package foo; func main() {}`,
		maxParams: 0,
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
