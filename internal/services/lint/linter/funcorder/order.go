package funcorder

import "slices"

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
	owner := m.owner
	if settled {
		owner = m.attach(true)
	}
	blocks := attached(owner)
	var units [][]int
	group := map[string]int{}
	for i, x := range m.fns {
		_, owned := owner[i]
		_, plain := m.plain[i]
		switch {
		case owned || plain:
		case x == nil || x.recv == "":
			units = append(units, append(blocks[i], i))
		case group[x.recv] > 0:
			u := group[x.recv] - 1
			units[u] = append(units[u], i)
		default:
			group[x.recv] = len(units) + 1
			units = append(units, append(blocks[i], i))
		}
	}
	return units
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
	if m.waitsInCycle(x) && !slices.ContainsFunc(x.cycleCallers, isDone) {
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
