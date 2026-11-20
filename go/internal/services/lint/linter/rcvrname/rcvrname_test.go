package rcvrname

import (
	"go/ast"
	"testing"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/linttest"
)

type checkCase struct {
	src     string
	wantN   int
	wantMsg string
}

func TestCheck_ShortReceiverR(t *testing.T) {
	runCheckCase(t, checkCase{
		src:     "package foo\n\ntype Repo struct{}\n\nfunc (r Repo) Get() {}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_TwoCharReceiverRp(t *testing.T) {
	runCheckCase(t, checkCase{
		src:     "package foo\n\ntype Repo struct{}\n\nfunc (rp Repo) Get() {}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_LongReceiverRepo(t *testing.T) {
	runCheckCase(t, checkCase{
		src:   "package foo\n\ntype Repo struct{}\n\nfunc (repo Repo) Get() {}\n",
		wantN: 1,
		wantMsg: `receiver name "repo" is 4 chars;` +
			` use 1-2 chars (e.g., "r")`,
	})
}

func TestCheck_InconsistentReceivers(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type Repo struct{}

func (r Repo) Get() {}
func (rp Repo) Set() {}
`,
		wantN: 1,
		wantMsg: `receiver name "rp" for type Repo` +
			` differs from "r"` +
			` used elsewhere in this file`,
	})
}

func TestCheck_ConsistentReceivers(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type Repo struct{}

func (r Repo) Get() {}
func (r Repo) Set() {}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_PointerReceiver(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type Repo struct{}

func (r *Repo) Get() {}
func (r *Repo) Set() {}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_DifferentTypesIndependent(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type Repo struct{}
type Service struct{}

func (r Repo) Get() {}
func (s Service) Run() {}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_GenericTypeReceiver(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type Cache[K comparable] struct{}

func (c Cache[K]) Get() {}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_GenericMultiParamReceiver(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type Pair[K, V any] struct{}

func (p Pair[K, V]) Key() {}
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_UnnamedReceiverSkipped(t *testing.T) {
	runCheckCase(t, checkCase{
		src: `package foo

type Repo struct{}

func (Repo) String() string { return "" }
`,
		wantN:   0,
		wantMsg: "",
	})
}

func TestCheck_NoMethods(t *testing.T) {
	runCheckCase(t, checkCase{
		src:     "package foo\n\ntype Repo struct{}\n",
		wantN:   0,
		wantMsg: "",
	})
}

func TestReceiverTypeName_Unnamed(t *testing.T) {
	arr := &ast.ArrayType{Lbrack: 0, Len: nil, Elt: nil}
	if got := receiverTypeName(arr); got != "" {
		t.Errorf("receiverTypeName([]T) = %q, want empty", got)
	}
}

func runCheckCase(t *testing.T, tt checkCase) {
	t.Helper()
	fset, f := linttest.Parse(t, "test.go", tt.src)
	linttest.AssertCount(t, Check(fset, "test.go", f), tt.wantN, tt.wantMsg)
}
