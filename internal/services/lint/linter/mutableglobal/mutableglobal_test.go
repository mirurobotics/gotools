package mutableglobal

import (
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
			name:     "const is allowed",
			filename: "foo.go",
			src: `package foo

const maxRetries = 3
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "function is allowed",
			filename: "foo.go",
			src: `package foo

func GetConfig() int { return 42 }
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "mutable var flagged",
			filename: "foo.go",
			src: `package foo

var counter int
`,
			wantN:   1,
			wantMsg: "mutable global variable: use const or a function instead",
		},
		{
			name:     "mutable var with value flagged",
			filename: "foo.go",
			src: `package foo

var name = "hello"
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "mutable map flagged",
			filename: "foo.go",
			src: `package foo

var cache = map[string]string{}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "error sentinel allowed",
			filename: "foo.go",
			src: `package foo

import "errors"

var ErrNotFound = errors.New("not found")
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "fmt.Errorf sentinel allowed",
			filename: "foo.go",
			src: `package foo

import "fmt"

var ErrBadInput = fmt.Errorf("bad input: %s", "x")
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "interface compliance check allowed",
			filename: "foo.go",
			src: `package foo

type Foo struct{}
type Bar interface{ Do() }

var _ Bar = (*Foo)(nil)
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "test file exempt",
			filename: "foo_test.go",
			src: `package foo

var testData = "hello"
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint comment on var block",
			filename: "foo.go",
			src: `package foo

//nolint:mutableglobal
var debug = false
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint inline comment",
			filename: "foo.go",
			src: `package foo

var debug = false //nolint:mutableglobal
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "multiple vars in one decl",
			filename: "foo.go",
			src: `package foo

import "errors"

var (
	ErrA = errors.New("a")
	counter int
	name string
)
`,
			wantN:   2,
			wantMsg: "",
		},
		{
			name:     "regexp MustCompile flagged",
			filename: "foo.go",
			src: `package foo

import "regexp"

var re = regexp.MustCompile("^foo$")
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "sync.Mutex flagged",
			filename: "foo.go",
			src: `package foo

import "sync"

var mu sync.Mutex
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "nolint on grouped var",
			filename: "foo.go",
			src: `package foo

import "sync"

//nolint:mutableglobal
var (
	mu sync.Mutex
	counter int
)
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint on single spec in group",
			filename: "foo.go",
			src: `package foo

var (
	clean = "ok"
	//nolint:mutableglobal
	dirty = "also ok"
)
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "local var not flagged",
			filename: "foo.go",
			src: `package foo

func f() {
	var x int
	_ = x
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "non-error call expression flagged",
			filename: "foo.go",
			src: `package foo

import "os"

var stdout = os.Stdout
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "custom NewFoo sentinel allowed",
			filename: "foo.go",
			src: `package foo

var ErrNotFound = myerrors.NewNotFound("x")
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "Errorf suffix sentinel allowed",
			filename: "foo.go",
			src: `package foo

var ErrUnavailable = grpc.Errorf("unavailable")
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "lowercase err prefix with pkg.New allowed",
			filename: "foo.go",
			src: `package foo

var errInternal = pkg.New("x")
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "non-Err var with pkg.New still flagged",
			filename: "foo.go",
			src: `package foo

var counter = pkg.New("x")
`,
			wantN:   1,
			wantMsg: "mutable global variable: use const or a function instead",
		},
		{
			name:     "NewLowercase not treated as constructor",
			filename: "foo.go",
			src: `package foo

var ErrBad = pkg.Newsletter("x")
`,
			wantN:   1,
			wantMsg: "mutable global variable: use const or a function instead",
		},
		{
			name:     "exactly 3-char Err name allowed",
			filename: "foo.go",
			src: `package foo

var Err = errors.New("x")
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "exactly 3-char err name allowed",
			filename: "foo.go",
			src: `package foo

var err = errors.New("x")
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "less than 3-char er name flagged",
			filename: "foo.go",
			src: `package foo

var Er = errors.New("x")
`,
			wantN:   1,
			wantMsg: "mutable global variable: use const or a function instead",
		},
		{
			name:     "unqualified New not treated as constructor",
			filename: "foo.go",
			src: `package foo

func New(s string) error { return nil }

var ErrBad = New("x")
`,
			wantN:   1,
			wantMsg: "mutable global variable: use const or a function instead",
		},
		{
			name:     "all-caps ERR prefix does not match Err or err",
			filename: "foo.go",
			src: `package foo

var ERR = errors.New("x")
`,
			wantN:   1,
			wantMsg: "mutable global variable: use const or a function instead",
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
