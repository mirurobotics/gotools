package tempdir

import (
	"go/parser"
	"go/token"
	"testing"
)

const wantMessage = "t.TempDir() is not allowed; use test_dirs.CreateTemp(t)" +
	" to create temporary test directories"

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		src      string
		wantN    int
		wantMsg  string
	}{
		{
			name:     "t.TempDir flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

func TestF(t *testing.T) {
	dir := t.TempDir()
	_ = dir
}
`,
			wantN:   1,
			wantMsg: wantMessage,
		},
		{
			name:     "b.TempDir flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

func BenchmarkF(b *testing.B) {
	_ = b.TempDir()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "f.TempDir flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

func FuzzF(f *testing.F) {
	_ = f.TempDir()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "testing.TB handle flagged",
			filename: "helper.go",
			src: `package foo

import "testing"

func helper(tb testing.TB) string {
	return tb.TempDir()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "handle held on a fixture struct flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

type fixture struct {
	t *testing.T
}

func (s *fixture) dir() string {
	return s.t.TempDir()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "handle passed to a plain helper flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

func run(t *testing.T) {
	_ = t.TempDir()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "no testing import not flagged",
			filename: "foo.go",
			src: `package foo

type fake struct{}

func (f fake) TempDir() string { return "" }

func g() string {
	t := fake{}
	return t.TempDir()
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "TempDir on non-handle receiver not flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

type fake struct{}

func (f fake) TempDir() string { return "" }

func TestF(t *testing.T) {
	other := fake{}
	_ = other.TempDir()
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "other handle methods not flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

func TestF(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {})
	t.Setenv("K", "V")
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint comment on same line",
			filename: "os.go",
			src: `package foo

import "testing"

func CreateTemp(t *testing.T) string {
	return t.TempDir() //nolint:tempdir
}
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "nolint comment on different line does not exempt",
			filename: "foo_test.go",
			src: `package foo

import "testing"

func TestF(t *testing.T) {
	//nolint:tempdir
	_ = t.TempDir()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "nolint other rule does not exempt",
			filename: "foo_test.go",
			src: `package foo

import "testing"

func TestF(t *testing.T) {
	_ = t.TempDir() //nolint:bgctx
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "aliased testing import flagged",
			filename: "foo_test.go",
			src: `package foo

import tst "testing"

func TestF(t *tst.T) {
	_ = t.TempDir()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "multiple calls flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

func TestA(t *testing.T) { _ = t.TempDir() }
func TestB(t *testing.T) { _ = t.TempDir() }
`,
			wantN:   2,
			wantMsg: "",
		},
		{
			name:     "non-testing field types ignored",
			filename: "foo_test.go",
			src: `package foo

import (
	"os"
	"testing"
)

type fixture struct {
	f *os.File
	m *testing.M
	t *testing.T
}

func (s *fixture) dir() string {
	return s.t.TempDir()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "testing imported but no handle declared",
			filename: "foo_test.go",
			src: `package foo

import "testing"

var _ = testing.Verbose

type fake struct{}

func (f fake) TempDir() string { return "" }

func g() string { return fake{}.TempDir() }
`,
			wantN:   0,
			wantMsg: "",
		},
		{
			name:     "method value flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

func TestF(t *testing.T) {
	fns := []func() string{t.TempDir}
	_ = fns[0]()
}
`,
			wantN:   1,
			wantMsg: "",
		},
		{
			name:     "call-expression receiver not flagged",
			filename: "foo_test.go",
			src: `package foo

import "testing"

type fake struct{}

func (f fake) TempDir() string { return "" }

func newFake() fake { return fake{} }

func TestF(t *testing.T) {
	_ = newFake().TempDir()
}
`,
			wantN:   0,
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
