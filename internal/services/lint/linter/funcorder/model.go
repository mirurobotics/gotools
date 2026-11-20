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
	waitsInCycle    bool
}

func build(f *ast.File) *model {
	m := &model{
		fns:      make([]*fn, len(f.Decls)),
		typeDecl: map[string]int{},
		heads:    nil,
		initVars: nil,
		plain:    nil,
		plainOf:  nil,
		decls:    f.Decls,
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
	m.markCycleWaits()
	m.heads = m.typeHeads()
	m.initVars = m.initVarDecls()
	m.plain = m.plainTypes()
	m.plainOf = map[int]int{}
	for t, p := range m.plain {
		for _, i := range p.head {
			m.plainOf[i] = t
		}
	}
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
	fns      []*fn
	typeDecl map[string]int
	heads    map[string][]int
	initVars map[int]bool
	plain    map[int]plainType
	plainOf  map[int]int
	decls    []ast.Decl
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

func (m *model) linkCallees(x *fn) {
	for _, j := range x.callees {
		to := m.fns[j]
		switch {
		case !enforced(x, to):
		case m.reaches(j, x.idx):
			to.cycleCallers = append(to.cycleCallers, x.idx)
		default:
			to.callers = append(to.callers, x.idx)
		}
	}
}

func (m *model) markCycleWaits() {
	for _, x := range m.funcs() {
		inCycleOnly := len(x.callers) == 0 && len(x.cycleCallers) > 0
		x.waitsInCycle = inCycleOnly && !m.opensCycle(x)
	}
}

func (m *model) opensCycle(x *fn) bool {
	for _, y := range m.funcs() {
		inCycle := y != x && !y.exempt && m.reaches(x.idx, y.idx)
		inCycle = inCycle && m.reaches(y.idx, x.idx)
		if inCycle && (y.idx < x.idx || len(y.callers) > 0) {
			return false
		}
	}
	return true
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

func (m *model) typeHeads() map[string][]int {
	isType := map[int]bool{}
	for _, i := range m.typeDecl {
		isType[i] = true
	}
	soloType := map[string]int{}
	for i, d := range m.decls {
		if ts := soloTypeSpec(d); ts != nil {
			soloType[ts.Name.Name] = i
		}
	}
	heads := map[string][]int{}
	for _, x := range m.funcs() {
		t, ok := soloType[x.recv]
		if _, seen := heads[x.recv]; ok && !seen {
			heads[x.recv] = m.typeHead(t, isType)
		}
	}
	return heads
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

func (m *model) typeHead(t int, isType map[int]bool) []int {
	head := []int{t}
	for i := t + 1; i < len(m.fns) && m.fns[i] == nil && !isType[i]; i++ {
		head = append(head, i)
	}
	return head
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
		waitsInCycle:    false,
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
