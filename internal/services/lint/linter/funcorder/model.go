package funcorder

import (
	"go/ast"
	"go/token"
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
	cycleCallers    []int
	comp            int
}

func build(f *ast.File) *model {
	m := &model{
		fns:        make([]*fn, len(f.Decls)),
		typeDecl:   map[string]int{},
		initVars:   nil,
		plain:      nil,
		typeOwner:  nil,
		typeBlocks: nil,
		opener:     nil,
		decls:      f.Decls,
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
	m.markComponents()
	for _, x := range m.funcs() {
		m.linkCallees(x)
	}
	m.opener = m.cycleOpeners()
	m.initVars = m.initVarDecls()
	m.plain = m.plainTypes()
	m.typeOwner = m.attachToTypes()
	m.typeBlocks = attached(m.typeOwner)
	return m
}

func callees(x *fn, byName map[string]int) []int {
	if x.decl.Body == nil {
		return nil
	}
	recv := recvName(x.decl)
	var out []int
	add := func(key string) {
		if j, ok := byName[key]; ok && j != x.idx && !slices.Contains(out, j) {
			out = append(out, j)
		}
	}
	onSel := func(n *ast.SelectorExpr) {
		if id, ok := n.X.(*ast.Ident); ok && recv != nil && id.Obj == recv.Obj {
			add(x.recv + "." + n.Sel.Name)
		}
	}
	refIdents(x.decl.Body, onSel, func(id *ast.Ident) bool {
		if id.Obj == nil || id.Obj.Kind == ast.Fun {
			add(id.Name)
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

func refIdents(n ast.Node, onSel func(*ast.SelectorExpr), ref func(*ast.Ident) bool) {
	skip := map[*ast.Ident]bool{}
	more := true
	ast.Inspect(n, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			skip[n.Sel] = true
			onSel(n)
		case *ast.KeyValueExpr:
			k, _ := n.Key.(*ast.Ident)
			skip[k] = true
		case *ast.Ident:
			more = more && (skip[n] || ref(n))
		}
		return more
	})
}

type model struct {
	fns        []*fn
	typeDecl   map[string]int
	initVars   map[int]bool
	plain      map[int][]int
	typeOwner  map[int]int
	typeBlocks map[int][]int
	opener     map[int]int
	decls      []ast.Decl
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
		if callable(d) {
			byName[x.name] = i
		}
	}
}

func (m *model) markComponents() {
	t := &tarjan{
		fns:     m.fns,
		index:   map[int]int{},
		low:     map[int]int{},
		stack:   nil,
		onStack: map[int]bool{},
	}
	for _, x := range m.funcs() {
		if _, seen := t.index[x.idx]; !seen {
			t.visit(x.idx)
		}
	}
}

func (m *model) linkCallees(x *fn) {
	for _, j := range x.callees {
		to := m.fns[j]
		switch {
		case !enforced(x, to):
		case to.comp == x.comp:
			to.cycleCallers = append(to.cycleCallers, x.idx)
		default:
			to.callers = append(to.callers, x.idx)
		}
	}
}

func (m *model) cycleOpeners() map[int]int {
	opener := map[int]int{}
	for _, x := range m.funcs() {
		_, seen := opener[x.comp]
		switch {
		case x.exempt:
		case len(x.callers) > 0:
			opener[x.comp] = -1
		case !seen:
			opener[x.comp] = x.idx
		}
	}
	return opener
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

func (m *model) initVarDecls() map[int]bool {
	initVars := map[int]bool{}
	for i, d := range m.decls {
		if gd, ok := d.(*ast.GenDecl); ok && hasInitializer(gd) {
			initVars[i] = true
		}
	}
	return initVars
}

func (m *model) waitsInCycle(x *fn) bool {
	inCycleOnly := len(x.callers) == 0 && len(x.cycleCallers) > 0
	return inCycleOnly && m.opener[x.comp] != x.idx
}

func (m *model) next(i int) []int {
	x := m.fns[i]
	if x == nil {
		return nil
	}
	var next []int
	for _, j := range x.callees {
		if enforced(x, m.fns[j]) {
			next = append(next, j)
		}
	}
	return next
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
		cycleCallers:    nil,
		comp:            0,
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

func enforced(from, to *fn) bool { return !from.exempt && !to.exempt }

func hasInitializer(d *ast.GenDecl) bool {
	if d.Tok != token.VAR {
		return false
	}
	return slices.ContainsFunc(d.Specs, func(s ast.Spec) bool {
		vs, ok := s.(*ast.ValueSpec)
		return ok && len(vs.Values) > 0
	})
}

type tarjan struct {
	fns     []*fn
	index   map[int]int
	low     map[int]int
	stack   []int
	onStack map[int]bool
}

func (t *tarjan) visit(i int) {
	n := len(t.index)
	t.index[i], t.low[i] = n, n
	t.stack = append(t.stack, i)
	t.onStack[i] = true
	for _, j := range t.fns[i].callees {
		if _, seen := t.index[j]; !seen {
			t.visit(j)
		}
		if t.onStack[j] {
			t.low[i] = min(t.low[i], t.low[j])
		}
	}
	if t.low[i] != t.index[i] {
		return
	}
	k := slices.Index(t.stack, i)
	for _, j := range t.stack[k:] {
		t.onStack[j] = false
		t.fns[j].comp = i
	}
	t.stack = t.stack[:k]
}
