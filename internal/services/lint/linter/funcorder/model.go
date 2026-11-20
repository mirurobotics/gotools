package funcorder

import (
	"go/ast"
	"slices"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

type fn struct {
	decl            *ast.FuncDecl
	idx             int
	name            string
	recv            string
	rank            int
	exempt          bool
	hasInFileCaller bool
	callees         []int
	callers         []int
}

func build(f *ast.File) *model {
	m := &model{
		fns:      make([]*fn, len(f.Decls)),
		typeDecl: map[string]int{},
		soloType: map[string]int{},
	}
	byName := map[string]int{}
	for i, d := range f.Decls {
		m.addDecl(i, d, byName)
	}
	for _, x := range m.funcs() {
		x.callees = callees(x, byName)
		for _, j := range x.callees {
			m.fns[j].hasInFileCaller = true
		}
	}
	for _, x := range m.funcs() {
		m.linkCallees(x)
	}
	return m
}

type model struct {
	fns      []*fn
	typeDecl map[string]int
	soloType map[string]int
}

func (m *model) funcs() []*fn {
	var out []*fn
	for _, x := range m.fns {
		if x != nil {
			out = append(out, x)
		}
	}
	return out
}

func (m *model) addDecl(i int, d ast.Decl, byName map[string]int) {
	switch d := d.(type) {
	case *ast.GenDecl:
		names := typeNames(d)
		for _, name := range names {
			m.typeDecl[name] = i
		}
		if len(d.Specs) == 1 && len(names) == 1 {
			m.soloType[names[0]] = i
		}
	case *ast.FuncDecl:
		x := newFn(i, d)
		m.fns[i] = x
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
		decl:            d,
		idx:             i,
		name:            d.Name.Name,
		recv:            recvType(d),
		rank:            rank(d),
		exempt:          d.Doc != nil && analysis.HasNolint(d.Doc, "funcorder"),
		hasInFileCaller: false,
		callees:         nil,
		callers:         nil,
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
	skip := map[*ast.Ident]bool{}
	var out []int
	add := func(key string) {
		if j, ok := byName[key]; ok && j != x.idx && !slices.Contains(out, j) {
			out = append(out, j)
		}
	}
	ast.Inspect(x.decl.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			skip[n.Sel] = true
			if id, ok := n.X.(*ast.Ident); ok && recv != nil && id.Obj == recv.Obj {
				add(x.recv + "." + n.Sel.Name)
			}
		case *ast.KeyValueExpr:
			if k, ok := n.Key.(*ast.Ident); ok {
				skip[k] = true
			}
		case *ast.Ident:
			if !skip[n] && (n.Obj == nil || n.Obj.Kind == ast.Fun) {
				add(n.Name)
			}
		}
		return true
	})
	return out
}

func recvName(d *ast.FuncDecl) *ast.Ident {
	if d.Recv == nil || len(d.Recv.List) == 0 || len(d.Recv.List[0].Names) == 0 {
		return nil
	}
	return d.Recv.List[0].Names[0]
}

func enforced(from, to *fn) bool { return !from.exempt && !to.exempt }
