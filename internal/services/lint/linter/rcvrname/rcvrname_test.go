package rcvrname

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
			name:    "short receiver r",
			src:     "package foo\n\ntype Repo struct{}\n\nfunc (r Repo) Get() {}\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name:    "two-char receiver rp",
			src:     "package foo\n\ntype Repo struct{}\n\nfunc (rp Repo) Get() {}\n",
			wantN:   0,
			wantMsg: "",
		},
		{
			name:  "long receiver repo",
			src:   "package foo\n\ntype Repo struct{}\n\nfunc (repo Repo) Get() {}\n",
			wantN: 1,
			wantMsg: `receiver name "repo" is 4 chars;` +
				` use 1-2 chars (e.g., "r")`,
		},
		{
			name: "inconsistent receivers",
			src: `package foo

type Repo struct{}

func (r Repo) Get() {}
func (rp Repo) Set() {}
`,
			wantN: 1,
			wantMsg: `receiver name "rp" for type Repo` +
				` differs from "r"` +
				` used elsewhere in this file`,
		},
		{
			name: "consistent receivers",
			src: `package foo

type Repo struct{}

func (r Repo) Get() {}
func (r Repo) Set() {}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "pointer receiver",
			src: `package foo

type Repo struct{}

func (r *Repo) Get() {}
func (r *Repo) Set() {}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "different types independent",
			src: `package foo

type Repo struct{}
type Service struct{}

func (r Repo) Get() {}
func (s Service) Run() {}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "generic type receiver",
			src: `package foo

type Cache[K comparable] struct{}

func (c Cache[K]) Get() {}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "generic multi-param receiver",
			src: `package foo

type Pair[K, V any] struct{}

func (p Pair[K, V]) Key() {}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name: "unnamed receiver skipped",
			src: `package foo

type Repo struct{}

func (Repo) String() string { return "" }
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:    "no methods",
			src:     "package foo\n\ntype Repo struct{}\n",
			wantN:   0,
			wantMsg: "",
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
