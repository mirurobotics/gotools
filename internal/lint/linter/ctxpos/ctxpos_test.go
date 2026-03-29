package ctxpos

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantN   int
		wantMsg string
	}{
		{
			name: "context first param",
			src: `package foo; import "context"` +
				`; func f(ctx context.Context, x int) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "context second param",
			src: `package foo; import "context"` +
				`; func f(x int, ctx context.Context) {}`,
			wantN: 1,
			wantMsg: "context.Context should be" +
				" the first parameter",
		},
		{
			name:    "no context param",
			src:     `package foo; func f(x int, y string) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "function literal with context not first",
			src: `package foo; import "context"` +
				`; var f = func(x int, ctx context.Context) {}`,
			wantN: 1,
			wantMsg: "context.Context should be" +
				" the first parameter",
		},
		{
			name: "function literal with context first",
			src: `package foo; import "context"` +
				`; var f = func(ctx context.Context, x int) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "aliased context import",
			src: `package foo; import ctx "context"` +
				`; func f(x int, c ctx.Context) {}`,
			wantN: 1,
			wantMsg: "context.Context should be" +
				" the first parameter",
		},
		{
			name:    "context not imported",
			src:     `package foo; func f(x int) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "only param is context",
			src: `package foo; import "context"` +
				`; func f(ctx context.Context) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "unnamed context param first",
			src: `package foo; import "context"` +
				`; func f(context.Context, int) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "testing.T first then context second",
			src: `package foo; import "context"` +
				`; import "testing"` +
				`; func TestFoo(t *testing.T, ctx context.Context) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "testing.B first then context second",
			src: `package foo; import "context"` +
				`; import "testing"` +
				`; func BenchmarkFoo(b *testing.B, ctx context.Context) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "testing.F first then context second",
			src: `package foo; import "context"` +
				`; import "testing"` +
				`; func FuzzFoo(f *testing.F, ctx context.Context) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "testing.M first then context second",
			src: `package foo; import "context"` +
				`; import "testing"` +
				`; func TestMain(m *testing.M, ctx context.Context) {}`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "non-testing first then context still flagged",
			src: `package foo; import "context"` +
				`; func f(x int, ctx context.Context) {}`,
			wantN: 1,
			wantMsg: "context.Context should be" +
				" the first parameter",
		},
		{
			name: "pointer to local type first then context flagged",
			src: `package foo; import "context"` +
				`; type myType struct{}` +
				`; func f(t *myType, ctx context.Context) {}`,
			wantN: 1,
			wantMsg: "context.Context should be" +
				" the first parameter",
		},
		{
			name: "testing non-TBFM type first then context flagged",
			src: `package foo; import "context"` +
				`; import "testing"` +
				`; func f(r *testing.PB, ctx context.Context) {}`,
			wantN: 1,
			wantMsg: "context.Context should be" +
				" the first parameter",
		},
		{
			name: "pointer to non-testing package first then context flagged",
			src: `package foo; import "context"` +
				`; import "os"` +
				`; func f(f *os.File, ctx context.Context) {}`,
			wantN: 1,
			wantMsg: "context.Context should be" +
				" the first parameter",
		},
		{
			name: "non-context selector before context",
			src: `package foo; import "context"` +
				`; import "io"` +
				`; func f(w io.Writer, ctx context.Context) {}`,
			wantN: 1,
			wantMsg: "context.Context should be" +
				" the first parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, "test.go", f)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}
