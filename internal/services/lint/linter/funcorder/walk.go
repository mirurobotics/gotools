package funcorder

import "slices"

func (m *model) arrange(heads map[string][]int) ([]int, map[int]bool) {
	units := m.units(heads)
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
