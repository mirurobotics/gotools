package bgctx

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		src      string
		wantN    int
		wantMsg  string
	}{
		{
			name:     "context.Background flagged",
			filename: "foo.go",
			src: `package foo

import "context"

func f() context.Context {
	return context.Background()
}
`,
			wantN: 1,
			wantMsg: "context.Background() is not allowed;" +
				" use mctx package to create contexts",
		},
		{
			name:     "context.TODO not flagged",
			filename: "foo.go",
			src: `package foo

import "context"

func f() context.Context {
	return context.TODO()
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "no context import not flagged",
			filename: "foo.go",
			src: `package foo

func f() string {
	return "hello"
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "test file exempt",
			filename: "foo_test.go",
			src: `package foo

import "context"

func f() context.Context {
	return context.Background()
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint comment on same line",
			filename: "foo.go",
			src: `package foo

import "context"

func f() context.Context {
	return context.Background() //nolint:bgctx
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint comment on different line does not exempt",
			filename: "foo.go",
			src: `package foo

import "context"

func f() context.Context {
	//nolint:bgctx
	return context.Background()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "multiple calls flagged",
			filename: "foo.go",
			src: `package foo

import "context"

func a() context.Context { return context.Background() }
func b() context.Context { return context.Background() }
`,
			wantN:   2,
			wantMsg: "",
		},
		{
			name:     "aliased context import flagged",
			filename: "foo.go",
			src: `package foo

import ctx "context"

func f() ctx.Context {
	return ctx.Background()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "nolint other rule does not exempt",
			filename: "foo.go",
			src: `package foo

import "context"

func f() context.Context {
	return context.Background() //nolint:mutableglobal
}
`,
			wantN:   1,
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, tt.filename, tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			diags := Check(fset, tt.filename, f)
			if len(diags) != tt.wantN {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tt.wantN, diags)
			}
			if tt.wantN > 0 && tt.wantMsg != "" && diags[0].Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", diags[0].Message, tt.wantMsg)
			}
		})
	}
}

func TestIsBackgroundCall(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"context.Background()", true},
		{"context.TODO()", false},
		{"Background()", false},
		{"a.context.Background()", false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			e, err := parser.ParseExpr(tt.expr)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			call, ok := e.(*ast.CallExpr)
			if !ok {
				t.Fatalf("%q is not a call", tt.expr)
			}
			if got := isBackgroundCall(call, "context"); got != tt.want {
				t.Errorf("isBackgroundCall(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}
