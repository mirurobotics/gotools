package funcorder

import (
	"go/ast"
	"slices"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

type fn struct {
	decl    *ast.FuncDecl
	idx     int
	name    string
	recv    string
	rank    int
	exempt  bool
	callees []int
	callers []int
}

type model struct {
	decls    []ast.Decl
	fns      map[int]*fn
	fnIdx    []int
	typeDecl map[string]int
}

func build(f *ast.File) *model {
	m := &model{
		decls:    f.Decls,
		fns:      map[int]*fn{},
		fnIdx:    nil,
		typeDecl: map[string]int{},
	}
	byName := map[string]int{}
	for i, d := range f.Decls {
		m.addDecl(i, d, byName)
	}
	for _, i := range m.fnIdx {
		m.fns[i].callees = callees(m.fns[i], byName)
	}
	for _, i := range m.fnIdx {
		m.linkCallees(m.fns[i])
	}
	return m
}

func (m *model) addDecl(i int, d ast.Decl, byName map[string]int) {
	switch d := d.(type) {
	case *ast.GenDecl:
		for _, name := range typeNames(d) {
			m.typeDecl[name] = i
		}
	case *ast.FuncDecl:
		x := newFn(i, d)
		m.fns[i] = x
		m.fnIdx = append(m.fnIdx, i)
		if callable(d) {
			byName[x.name] = i
		}
	}
}

func (m *model) linkCallees(x *fn) {
	for _, j := range x.callees {
		to := m.fns[j]
		if enforced(x, to) && !m.reaches(j, x.idx) {
			to.callers = append(to.callers, x.idx)
		}
	}
}

func (m *model) reaches(from, to int) bool {
	seen := map[int]bool{}
	stack := []int{from}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == to {
			return true
		}
		if !seen[n] {
			seen[n] = true
			stack = append(stack, m.fns[n].callees...)
		}
	}
	return false
}

func typeNames(gd *ast.GenDecl) []string {
	var names []string
	for _, s := range gd.Specs {
		if ts, ok := s.(*ast.TypeSpec); ok {
			names = append(names, ts.Name.Name)
		}
	}
	return names
}

func newFn(i int, d *ast.FuncDecl) *fn {
	x := &fn{
		decl:    d,
		idx:     i,
		name:    d.Name.Name,
		recv:    recvType(d),
		rank:    rank(d),
		exempt:  d.Doc != nil && analysis.HasNolint(d.Doc, "funcorder"),
		callees: nil,
		callers: nil,
	}
	if x.recv != "" {
		x.name = x.recv + "." + x.name
	}
	return x
}

func recvType(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	if id := analysis.BaseIdent(d.Recv.List[0].Type); id != nil {
		return id.Name
	}
	return ""
}

func rank(d *ast.FuncDecl) int {
	name := d.Name.Name
	if ast.IsExported(name) || d.Recv == nil && (name == "main" || name == "init") {
		return 0
	}
	return 1
}

func callable(d *ast.FuncDecl) bool {
	return d.Name.Name != "_" && (d.Recv != nil || d.Name.Name != "init")
}

func callees(x *fn, byName map[string]int) []int {
	if x.decl.Body == nil {
		return nil
	}
	recv := recvName(x.decl)
	sels := map[*ast.Ident]bool{}
	var out []int
	add := func(key string) {
		if j, ok := byName[key]; ok && j != x.idx && !slices.Contains(out, j) {
			out = append(out, j)
		}
	}
	ast.Inspect(x.decl.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			sels[n.Sel] = true
			if id, ok := n.X.(*ast.Ident); ok && recv != "" && id.Name == recv {
				add(x.recv + "." + n.Sel.Name)
			}
		case *ast.Ident:
			if !sels[n] && (n.Obj == nil || n.Obj.Kind == ast.Fun) {
				add(n.Name)
			}
		}
		return true
	})
	return out
}

func recvName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 || len(d.Recv.List[0].Names) == 0 {
		return ""
	}
	return d.Recv.List[0].Names[0].Name
}

func enforced(from, to *fn) bool {
	if from.exempt || to.exempt || from.recv != "" && to.recv == "" && to.rank == 0 {
		return false
	}
	return to.rank >= from.rank
}
