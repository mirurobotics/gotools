package paramcount

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		src       string
		maxParams int
		wantN     int
		wantMsg   string
	}{
		{
			name:     "at limit",
			filename: "",
			src: `package foo` +
				`; func f(a, b, c, d, e int) {}`,
			maxParams: 5,
			wantN:     0,
			wantMsg:   "",
		},
		{
			name:     "over limit",
			filename: "",
			src: `package foo` +
				`; func f(a, b, c, d, e, f int) {}`,
			maxParams: 5,
			wantN:     1,
			wantMsg: "function f has 6 parameters" +
				" (limit 5); consider an options struct",
		},
		{
			name:     "context excluded",
			filename: "",
			src: `package foo; import "context"` +
				`; func f(ctx context.Context, a, b, c, d, e int) {}`,
			maxParams: 5,
			wantN:     0,
			wantMsg:   "",
		},
		{
			name:     "grouped params counted correctly",
			filename: "",
			src: `package foo` +
				`; func f(a, b int, c string, d, e bool, f float64) {}`,
			maxParams: 5,
			wantN:     1,
			wantMsg: "function f has 6 parameters" +
				" (limit 5); consider an options struct",
		},
		{
			name:      "main exempt",
			filename:  "",
			src:       `package foo; func main() {}`,
			maxParams: 0,
			wantN:     0,
			wantMsg:   "",
		},
		{
			name:      "init exempt",
			filename:  "",
			src:       `package foo; func init() {}`,
			maxParams: 0,
			wantN:     0,
			wantMsg:   "",
		},
		{
			name:      "under limit",
			filename:  "",
			src:       `package foo; func f(a int) {}`,
			maxParams: 5,
			wantN:     0,
			wantMsg:   "",
		},
		{
			name:     "method with receiver",
			filename: "",
			src: `package foo; type S struct{}` +
				`; func (s S) f(a, b, c, d, e, g int) {}`,
			maxParams: 5,
			wantN:     1,
			wantMsg: "function f has 6 parameters" +
				" (limit 5); consider an options struct",
		},
		{
			name:     "benchmark exempt",
			filename: "",
			src: `package foo; import "testing"` +
				`; func BenchmarkFoo(b *testing.B,` +
				` a, c, d, e, g int) {}`,
			maxParams: 5,
			wantN:     0,
			wantMsg:   "",
		},
		{
			name:     "unnamed parameters",
			filename: "",
			src: `package foo` +
				`; func f(int, string, bool) {}`,
			maxParams: 2,
			wantN:     1,
			wantMsg: "function f has 3 parameters" +
				" (limit 2);" +
				" consider an options struct",
		},
		{
			name:     "aliased context import",
			filename: "",
			src: `package foo; import ctx "context"` +
				`; func f(c ctx.Context,` +
				` a, b, d, e, g int) {}`,
			maxParams: 5,
			wantN:     0,
			wantMsg:   "",
		},
		{
			name:     "tools/lint exempt",
			filename: "tools/lint/check.go",
			src: `package lint` +
				`; func f(a, b, c, d, e, g int) {}`,
			maxParams: 5,
			wantN:     0,
			wantMsg:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.filename
			if filename == "" {
				filename = "test.go"
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, filename, tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, filename, f, tt.maxParams)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}
