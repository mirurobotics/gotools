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
	lay := &layout{
		m:      m,
		unit:   map[int][]int{},
		unitOf: map[int]int{},
		done:   map[int]bool{},
		below:  map[int]bool{},
		out:    nil,
		roots:  nil,
	}
	keys := make([]int, len(units))
	for unitIdx, unit := range units {
		keys[unitIdx] = unit[0]
		lay.unit[unit[0]] = unit
		for _, i := range unit {
			lay.unitOf[i] = unit[0]
		}
	}
	walk(keys, lay.unitReady, lay.unitNext, lay.emitUnit)
	out := m.tuckTypes(lay.out, lay.roots, lay.below)
	return m.placeTypes(units, out), lay.below
}

func (m *model) units(settled bool) [][]int {
	owner := m.attach(settled)
	blocks := attached(owner)
	var units [][]int
	group := map[string]int{}
	for i, fnc := range m.fns {
		_, owned := owner[i]
		_, plain := m.plain[i]
		switch {
		case owned || plain:
		case fnc == nil || fnc.recv == "":
			units = append(units, append(blocks[i], i))
		case group[fnc.recv] > 0:
			slot := group[fnc.recv] - 1
			units[slot] = append(units[slot], i)
		default:
			group[fnc.recv] = len(units) + 1
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

func (m *model) unitReady(unit []int, done map[int]bool) bool {
	fnc := m.fns[unit[len(unit)-1]]
	if fnc == nil || fnc.recv == "" {
		return m.ready(fnc, done)
	}
	typeIdx, ok := m.typeDecl[fnc.recv]
	return !ok || done[typeIdx] || slices.Contains(unit, typeIdx)
}

func (m *model) ready(fnc *fn, done map[int]bool) bool {
	if fnc == nil || fnc.exempt {
		return true
	}
	isDone := func(i int) bool { return done[i] }
	for _, caller := range fnc.callers {
		if !done[caller] {
			return false
		}
	}
	if m.waitsInCycle(fnc) && !slices.ContainsFunc(fnc.cycleCallers, isDone) {
		return false
	}
	return m.ranked(fnc, isDone)
}

func (m *model) ranked(fnc *fn, isDone func(int) bool) bool {
	for _, other := range m.funcs() {
		if mustPrecede(other, fnc) && !isDone(other.idx) {
			return false
		}
	}
	return true
}

func mustPrecede(fnc, other *fn) bool {
	if fnc.rank != 0 || other.rank != 1 || fnc.exempt || other.exempt {
		return false
	}
	return fnc.recv == other.recv && !fnc.hasInFileCaller && !other.hasInFileCaller
}

type layout struct {
	m      *model
	unit   map[int][]int
	unitOf map[int]int
	done   map[int]bool
	below  map[int]bool
	out    []int
	roots  []int
}

func (l *layout) unitReady(unitIdx int) bool {
	return l.m.unitReady(l.unit[unitIdx], l.done)
}

func (l *layout) unitNext(unitIdx int) []int {
	var next []int
	for _, i := range l.out {
		if l.unitOf[i] != unitIdx {
			continue
		}
		for _, nextIdx := range l.m.next(i) {
			next = append(next, l.unitOf[nextIdx])
		}
	}
	return next
}

func (l *layout) emitUnit(unitIdx int, below bool) {
	if !below {
		l.roots = append(l.roots, len(l.out))
	}
	walk(l.unit[unitIdx], l.fnReady, l.m.next, func(i int, inner bool) {
		l.done[i] = true
		l.below[i] = below || inner
		l.out = append(l.out, i)
	})
}

func (l *layout) fnReady(i int) bool { return l.m.ready(l.m.fns[i], l.done) }

func walk(
	items []int, ready func(int) bool, next func(int) []int, emit func(int, bool),
) {
	wlk := &walker{left: slices.Clone(items), stack: nil, ready: ready, next: next}
	for len(wlk.left) > 0 {
		i, below := wlk.pick()
		picked := func(idx int) bool { return idx == i }
		wlk.left = slices.DeleteFunc(wlk.left, picked)
		wlk.stack = append(wlk.stack, i)
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
		if key := slices.IndexFunc(next, w.pending); key >= 0 {
			return next[key], true
		}
		w.stack = w.stack[:len(w.stack)-1]
	}
	return w.left[max(slices.IndexFunc(w.left, w.ready), 0)], false
}

func (w *walker) pending(i int) bool { return slices.Contains(w.left, i) && w.ready(i) }
