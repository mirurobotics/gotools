package analysis

import (
	"go/ast"
	"go/types"
	"reflect"
	"testing"
)

type walkedName struct {
	name string
	kind NameKind
	typ  string
}

func TestWalkDeclaredNames(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []walkedName
	}{
		{
			"func params and results",
			"func run(left, right int, rest ...string) (out bool) { return }",
			[]walkedName{
				{"run", KindFunc, ""},
				{"left", KindParam, "int"},
				{"right", KindParam, "int"},
				{"rest", KindParam, "...string"},
				{"out", KindResult, "bool"},
			},
		},
		{
			"unnamed params",
			"func run(int, string) error { return nil }",
			[]walkedName{{"run", KindFunc, ""}},
		},
		{
			"test func",
			"func TestRunAll(t *testing.T) {}",
			[]walkedName{
				{"TestRunAll", KindTestFunc, ""},
				{"t", KindParam, "*testing.T"},
			},
		},
		{
			"method skips receiver",
			"func (s *S) Run(arg int) {}",
			[]walkedName{{"Run", KindMethod, ""}, {"arg", KindParam, "int"}},
		},
		{
			"type params skipped",
			"func Map[T any](val T) {}\ntype Box[T any] struct{ Val T }",
			[]walkedName{
				{"Map", KindFunc, ""},
				{"val", KindParam, "T"},
				{"Box", KindType, ""},
				{"Val", KindField, "T"},
			},
		},
		{
			"package vars",
			"var count, total = 1, 2",
			[]walkedName{{"count", KindPackageVar, ""}, {"total", KindPackageVar, ""}},
		},
		{
			"func literal in package var",
			`var handler = func(arg int) (res int) {
	var inner int
	sum := arg + inner
	return sum
}`,
			[]walkedName{
				{"handler", KindPackageVar, ""},
				{"arg", KindParam, "int"},
				{"res", KindResult, "int"},
				{"inner", KindLocalVar, ""},
				{"sum", KindLocalVar, ""},
			},
		},
		{
			"local decls",
			`func run() {
	var total int
	const limit = 3
	type pair struct{ left int }
}`,
			[]walkedName{
				{"run", KindFunc, ""},
				{"total", KindLocalVar, ""},
				{"limit", KindConst, ""},
				{"pair", KindType, ""},
				{"left", KindField, "int"},
			},
		},
		{
			"const values not walked",
			`const (
	low = iota
	size = unsafe.Sizeof(struct{ field int }{})
)`,
			[]walkedName{{"low", KindConst, ""}, {"size", KindConst, ""}},
		},
		{
			"struct fields",
			`type Config struct {
	Name, Path string
	Nested struct{ Depth int }
	Embedded
}`,
			[]walkedName{
				{"Config", KindType, ""},
				{"Name", KindField, "string"},
				{"Path", KindField, "string"},
				{"Nested", KindField, "struct{Depth int}"},
				{"Depth", KindField, "int"},
			},
		},
		{
			"anonymous struct types",
			"var origin struct{ Lat int }\nfunc run(opts struct{ Verbose bool }) {}",
			[]walkedName{
				{"origin", KindPackageVar, ""},
				{"Lat", KindField, "int"},
				{"run", KindFunc, ""},
				{"opts", KindParam, "struct{Verbose bool}"},
				{"Verbose", KindField, "bool"},
			},
		},
		{
			"interface methods",
			"type Store interface {\n\tGet(key string) int\n\tio.Reader\n}",
			[]walkedName{
				{"Store", KindType, ""},
				{"Get", KindIfaceMethod, "func(key string) int"},
			},
		},
		{
			"func type params skipped",
			"type Handler func(req int) error\nfunc apply(visit func(inner int)) {}",
			[]walkedName{
				{"Handler", KindType, ""},
				{"apply", KindFunc, ""},
				{"visit", KindParam, "func(inner int)"},
			},
		},
		{
			"redeclared idents skipped",
			`func run() {
	first, err := load()
	second, err := load()
	first = second
}`,
			[]walkedName{
				{"run", KindFunc, ""},
				{"first", KindLocalVar, ""},
				{"err", KindLocalVar, ""},
				{"second", KindLocalVar, ""},
			},
		},
		{
			"range vars",
			`func run(items []int) {
	var pos int
	for pos = range items {
	}
	for key, val := range items {
	}
}`,
			[]walkedName{
				{"run", KindFunc, ""},
				{"items", KindParam, "[]int"},
				{"pos", KindLocalVar, ""},
				{"key", KindLocalVar, ""},
				{"val", KindLocalVar, ""},
			},
		},
		{
			"labels and imports skipped",
			`import fmtx "fmt"

func run() {
outer:
	for {
		break outer
	}
}`,
			[]walkedName{{"run", KindFunc, ""}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := parseTestFile(t, "package foo\n"+tc.src)
			if got := walkedNames(file); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("WalkDeclaredNames = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func walkedNames(file *ast.File) []walkedName {
	var got []walkedName
	WalkDeclaredNames(file, func(name DeclaredName) {
		typ := ""
		if name.Type != nil {
			typ = types.ExprString(name.Type)
		}
		got = append(got, walkedName{name.Ident.Name, name.Kind, typ})
	})
	return got
}
