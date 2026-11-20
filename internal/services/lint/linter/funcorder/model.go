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

func build(file *ast.File) *model {
	graph := &model{
		fns:        make([]*fn, len(file.Decls)),
		typeDecl:   map[string]int{},
		initVars:   nil,
		plain:      nil,
		typeOwner:  nil,
		typeBlocks: nil,
		opener:     nil,
		decls:      file.Decls,
	}
	byName := map[string]int{}
	for i, decl := range file.Decls {
		graph.addDecl(i, decl, byName)
	}
	for _, fnc := range graph.funcs() {
		fnc.callees = callees(fnc, byName)
		for _, calleeIdx := range fnc.callees {
			graph.fns[calleeIdx].hasInFileCaller = true
		}
	}
	graph.markComponents()
	for _, fnc := range graph.funcs() {
		graph.linkCallees(fnc)
	}
	graph.opener = graph.cycleOpeners()
	graph.initVars = graph.initVarDecls()
	graph.plain = graph.plainTypes()
	graph.typeOwner = graph.attachToTypes()
	graph.typeBlocks = attached(graph.typeOwner)
	return graph
}

func callees(fnc *fn, byName map[string]int) []int {
	if fnc.decl.Body == nil {
		return nil
	}
	recv := recvName(fnc.decl)
	var out []int
	add := func(key string) {
		calleeIdx, ok := byName[key]
		if ok && calleeIdx != fnc.idx && !slices.Contains(out, calleeIdx) {
			out = append(out, calleeIdx)
		}
	}
	onSel := func(sel *ast.SelectorExpr) {
		if id, ok := sel.X.(*ast.Ident); ok && recv != nil && id.Obj == recv.Obj {
			add(fnc.recv + "." + sel.Sel.Name)
		}
	}
	refIdents(fnc.decl.Body, onSel, func(id *ast.Ident) bool {
		if id.Obj == nil || id.Obj.Kind == ast.Fun {
			add(id.Name)
		}
		return true
	})
	return out
}

func recvName(decl *ast.FuncDecl) *ast.Ident {
	recv := decl.Recv
	if recv == nil || len(recv.List) == 0 || len(recv.List[0].Names) == 0 {
		return nil
	}
	return decl.Recv.List[0].Names[0]
}

func refIdents(
	node ast.Node, onSel func(*ast.SelectorExpr), ref func(*ast.Ident) bool,
) {
	skip := map[*ast.Ident]bool{}
	more := true
	ast.Inspect(node, func(astNode ast.Node) bool {
		switch astNode := astNode.(type) {
		case *ast.SelectorExpr:
			skip[astNode.Sel] = true
			onSel(astNode)
		case *ast.KeyValueExpr:
			ident, _ := astNode.Key.(*ast.Ident)
			skip[ident] = true
		case *ast.Ident:
			more = more && (skip[astNode] || ref(astNode))
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

func (m *model) addDecl(i int, decl ast.Decl, byName map[string]int) {
	switch decl := decl.(type) {
	case *ast.GenDecl:
		for _, name := range typeNames(decl) {
			m.typeDecl[name] = i
		}
	case *ast.FuncDecl:
		fnc := newFn(i, decl)
		m.fns[i] = fnc
		if callable(decl) {
			byName[fnc.name] = i
		}
	}
}

func (m *model) markComponents() {
	scc := &tarjan{
		fns:     m.fns,
		index:   map[int]int{},
		low:     map[int]int{},
		stack:   nil,
		onStack: map[int]bool{},
	}
	for _, fnc := range m.funcs() {
		if _, seen := scc.index[fnc.idx]; !seen {
			scc.visit(fnc.idx)
		}
	}
}

func (m *model) linkCallees(fnc *fn) {
	for _, calleeIdx := range fnc.callees {
		callee := m.fns[calleeIdx]
		switch {
		case !enforced(fnc, callee):
		case callee.comp == fnc.comp:
			callee.cycleCallers = append(callee.cycleCallers, fnc.idx)
		default:
			callee.callers = append(callee.callers, fnc.idx)
		}
	}
}

func (m *model) cycleOpeners() map[int]int {
	opener := map[int]int{}
	for _, fnc := range m.funcs() {
		_, seen := opener[fnc.comp]
		switch {
		case fnc.exempt:
		case len(fnc.callers) > 0:
			opener[fnc.comp] = -1
		case !seen:
			opener[fnc.comp] = fnc.idx
		}
	}
	return opener
}

func (m *model) funcs() []*fn {
	var out []*fn
	for _, fnc := range m.fns {
		if fnc != nil {
			out = append(out, fnc)
		}
	}
	return out
}

func (m *model) initVarDecls() map[int]bool {
	initVars := map[int]bool{}
	for i, decl := range m.decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && hasInitializer(genDecl) {
			initVars[i] = true
		}
	}
	return initVars
}

func (m *model) waitsInCycle(fnc *fn) bool {
	inCycleOnly := len(fnc.callers) == 0 && len(fnc.cycleCallers) > 0
	return inCycleOnly && m.opener[fnc.comp] != fnc.idx
}

func (m *model) next(i int) []int {
	fnc := m.fns[i]
	if fnc == nil {
		return nil
	}
	var next []int
	for _, calleeIdx := range fnc.callees {
		if enforced(fnc, m.fns[calleeIdx]) {
			next = append(next, calleeIdx)
		}
	}
	return next
}

func typeNames(decl *ast.GenDecl) []string {
	var names []string
	for _, spec := range decl.Specs {
		if typeSpec, ok := spec.(*ast.TypeSpec); ok {
			names = append(names, typeSpec.Name.Name)
		}
	}
	return names
}

func newFn(i int, decl *ast.FuncDecl) *fn {
	fnc := &fn{
		decl:            decl,
		idx:             i,
		name:            decl.Name.Name,
		recv:            recvType(decl),
		rank:            rank(decl),
		exempt:          decl.Doc != nil && analysis.HasNolint(decl.Doc, "funcorder"),
		hasInFileCaller: false,
		callees:         nil,
		callers:         nil,
		cycleCallers:    nil,
		comp:            0,
	}
	if fnc.recv != "" {
		fnc.name = fnc.recv + "." + fnc.name
	}
	return fnc
}

func recvType(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return ""
	}
	if id := analysis.BaseIdent(decl.Recv.List[0].Type); id != nil {
		return id.Name
	}
	return ""
}

func rank(decl *ast.FuncDecl) int {
	name := decl.Name.Name
	if ast.IsExported(name) || decl.Recv == nil && (name == "main" || name == "init") {
		return 0
	}
	return 1
}

func callable(decl *ast.FuncDecl) bool {
	return decl.Name.Name != "_" && (decl.Recv != nil || decl.Name.Name != "init")
}

func enforced(caller, callee *fn) bool { return !caller.exempt && !callee.exempt }

func hasInitializer(decl *ast.GenDecl) bool {
	if decl.Tok != token.VAR {
		return false
	}
	return slices.ContainsFunc(decl.Specs, func(spec ast.Spec) bool {
		valueSpec, ok := spec.(*ast.ValueSpec)
		return ok && len(valueSpec.Values) > 0
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
	num := len(t.index)
	t.index[i], t.low[i] = num, num
	t.stack = append(t.stack, i)
	t.onStack[i] = true
	for _, calleeIdx := range t.fns[i].callees {
		if _, seen := t.index[calleeIdx]; !seen {
			t.visit(calleeIdx)
		}
		if t.onStack[calleeIdx] {
			t.low[i] = min(t.low[i], t.low[calleeIdx])
		}
	}
	if t.low[i] != t.index[i] {
		return
	}
	start := slices.Index(t.stack, i)
	for _, member := range t.stack[start:] {
		t.onStack[member] = false
		t.fns[member].comp = i
	}
	t.stack = t.stack[:start]
}
