package funcorder

import (
	"go/ast"
	"go/token"
	"slices"
)

func (m *model) order() ([]int, map[int]bool) {
	out, below := m.arrange(false)
	if m.keepsInitOrder(out) {
		return out, below
	}
	return m.arrange(true)
}

func (m *model) arrange(settled bool) ([]int, map[int]bool) {
	units := m.units(settled)
	l := &layout{
		m:      m,
		unit:   map[int][]int{},
		unitOf: map[int]int{},
		done:   map[int]bool{},
		below:  map[int]bool{},
		out:    nil,
	}
	keys := make([]int, len(units))
	for k, u := range units {
		keys[k] = u[0]
		l.unit[u[0]] = u
		for _, i := range u {
			l.unitOf[i] = u[0]
		}
	}
	walk(keys, l.unitReady, l.unitNext, l.emitUnit)
	return m.placeTypes(units, l.out), l.below
}

func (m *model) units(settled bool) [][]int {
	heads := m.heads
	if settled {
		heads = m.settledHeads()
	}
	var units [][]int
	group := map[string]int{}
	glued := map[int]bool{}
	for _, head := range heads {
		for _, i := range head {
			glued[i] = true
		}
	}
	lead := m.leads(glued, settled)
	for i, x := range m.fns {
		switch {
		case glued[i] || m.inPlain(i):
		case x == nil || x.recv == "":
			units = appendFree(units, i, lead)
		case group[x.recv] > 0:
			u := group[x.recv] - 1
			units[u] = append(units[u], i)
		default:
			group[x.recv] = len(units) + 1
			units = append(units, unitHead(heads, x.recv, i))
		}
	}
	return units
}

func (m *model) settledHeads() map[string][]int {
	isInit := func(i int) bool { return m.initVars[i] }
	heads := map[string][]int{}
	for recv, head := range m.heads {
		heads[recv] = head
		if k := slices.IndexFunc(head, isInit); k > 0 && !m.placed(recv) {
			heads[recv] = head[:k]
		}
	}
	return heads
}

func (m *model) placed(recv string) bool {
	isMethod := func(x *fn) bool { return x != nil && x.recv == recv }
	head := m.heads[recv]
	return head[len(head)-1] == slices.IndexFunc(m.fns, isMethod)-1
}

func (m *model) leads(glued map[int]bool, settled bool) map[int]bool {
	first := slices.IndexFunc(m.fns, func(x *fn) bool { return x != nil })
	lead := map[int]bool{}
	for i, x := range m.fns {
		isFree := x != nil && x.recv == ""
		for j := i - 1; isFree && j > first && m.leadable(j, glued, settled); j-- {
			lead[j] = true
		}
	}
	return lead
}

func (m *model) leadable(i int, glued map[int]bool, settled bool) bool {
	gd, ok := m.decls[i].(*ast.GenDecl)
	if !ok || glued[i] || m.inPlain(i) || settled && m.initVars[i] {
		return false
	}
	return gd.Tok == token.CONST || gd.Tok == token.VAR
}

func (m *model) keepsInitOrder(order []int) bool {
	var vars []int
	for _, i := range order {
		if m.initVars[i] {
			vars = append(vars, i)
		}
	}
	return slices.IsSorted(vars)
}

func (m *model) unitReady(u []int, done map[int]bool) bool {
	x := m.fns[u[len(u)-1]]
	if x == nil || x.recv == "" {
		return m.ready(x, done)
	}
	t, ok := m.typeDecl[x.recv]
	return !ok || done[t] || slices.Contains(u, t)
}

func (m *model) ready(x *fn, done map[int]bool) bool {
	if x == nil || x.exempt {
		return true
	}
	isDone := func(i int) bool { return done[i] }
	for _, c := range x.callers {
		if !done[c] {
			return false
		}
	}
	if x.waitsInCycle && !slices.ContainsFunc(x.cycleCallers, isDone) {
		return false
	}
	return m.ranked(x, isDone)
}

func (m *model) ranked(y *fn, isDone func(int) bool) bool {
	for _, x := range m.funcs() {
		if mustPrecede(x, y) && !isDone(x.idx) {
			return false
		}
	}
	return true
}

func appendFree(units [][]int, i int, lead map[int]bool) [][]int {
	if n := len(units); n > 0 && lead[units[n-1][len(units[n-1])-1]] {
		units[n-1] = append(units[n-1], i)
		return units
	}
	return append(units, []int{i})
}

func unitHead(heads map[string][]int, recv string, i int) []int {
	if head, ok := heads[recv]; ok {
		return append(slices.Clip(head), i)
	}
	return []int{i}
}

func mustPrecede(x, y *fn) bool {
	if x.rank != 0 || y.rank != 1 || x.exempt || y.exempt {
		return false
	}
	return x.recv == y.recv && !x.hasInFileCaller && !y.hasInFileCaller
}

type layout struct {
	m      *model
	unit   map[int][]int
	unitOf map[int]int
	done   map[int]bool
	below  map[int]bool
	out    []int
}

func (l *layout) unitReady(key int) bool { return l.m.unitReady(l.unit[key], l.done) }

func (l *layout) unitNext(key int) []int {
	var next []int
	for _, i := range l.out {
		if l.unitOf[i] != key {
			continue
		}
		for _, j := range l.m.next(i) {
			next = append(next, l.unitOf[j])
		}
	}
	return next
}

func (l *layout) emitUnit(key int, below bool) {
	walk(l.unit[key], l.fnReady, l.m.next, func(i int, inner bool) {
		l.done[i] = true
		l.below[i] = below || inner
		l.out = append(l.out, i)
	})
}

func (l *layout) fnReady(i int) bool { return l.m.ready(l.m.fns[i], l.done) }

func walk(
	items []int, ready func(int) bool, next func(int) []int, emit func(int, bool),
) {
	w := &walker{left: slices.Clone(items), stack: nil, ready: ready, next: next}
	for len(w.left) > 0 {
		i, below := w.pick()
		w.left = slices.DeleteFunc(w.left, func(j int) bool { return j == i })
		w.stack = append(w.stack, i)
		emit(i, below)
	}
}

type walker struct {
	left, stack []int
	ready       func(int) bool
	next        func(int) []int
}

func (w *walker) pick() (int, bool) {
	for len(w.stack) > 0 {
		next := w.next(w.stack[len(w.stack)-1])
		if k := slices.IndexFunc(next, w.pending); k >= 0 {
			return next[k], true
		}
		w.stack = w.stack[:len(w.stack)-1]
	}
	return w.left[max(slices.IndexFunc(w.left, w.ready), 0)], false
}

func (w *walker) pending(i int) bool { return slices.Contains(w.left, i) && w.ready(i) }
