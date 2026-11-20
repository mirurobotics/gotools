package funcorder

import (
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports functions that are out of control-flow order.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	m := build(f)
	var diags []analysis.Diagnostic
	for _, i := range m.fnIdx {
		x := m.fns[i]
		if x.exempt {
			continue
		}
		line := fset.Position(x.decl.Pos()).Line
		for _, msg := range m.violations(x) {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    line,
				Message: msg,
			})
		}
	}
	return diags
}

// Fix returns src with its declarations reordered, or nil when already ordered.
func Fix(fset *token.FileSet, f *ast.File, src []byte) []byte {
	order := build(f).order()
	if slices.IsSorted(order) {
		return nil
	}
	return render(fset, f, src, order)
}

func (m *model) violations(x *fn) []string {
	checks := []func(*fn) string{m.stepdown, m.rankOrder, m.belowType, m.grouped}
	var msgs []string
	for _, check := range checks {
		if msg := check(x); msg != "" {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

func (m *model) stepdown(x *fn) string {
	if len(x.callers) == 0 || slices.Min(x.callers) < x.idx {
		return ""
	}
	first := m.fns[slices.Min(x.callers)].name
	return fmt.Sprintf("%q should be below its first caller %q", x.name, first)
}

func (m *model) rankOrder(x *fn) string {
	if x.rank != 0 {
		return ""
	}
	for _, i := range m.fnIdx {
		if y := m.fns[i]; i < x.idx && mustPrecede(x, y) {
			return fmt.Sprintf("%q should be above unexported %q", x.name, y.name)
		}
	}
	return ""
}

func (m *model) belowType(x *fn) string {
	if t, ok := m.typeDecl[x.recv]; !ok || t < x.idx {
		return ""
	}
	return fmt.Sprintf("method %q should be below type %q", x.name, x.recv)
}

func (m *model) grouped(x *fn) string {
	prev := -1
	for _, i := range m.fnIdx {
		y := m.fns[i]
		if i < x.idx && x.recv != "" && y.recv == x.recv && !y.exempt {
			prev = i
		}
	}
	if prev < 0 || prev == x.idx-1 {
		return ""
	}
	return fmt.Sprintf(
		"method %q should be grouped with the other methods of %q", x.name, x.recv,
	)
}

func (m *model) order() []int {
	done := map[int]bool{}
	var out []int
	mark := func(i int) {
		done[i] = true
		out = append(out, i)
	}
	unitReady := func(u []int) bool { return m.unitReady(u, done) }
	fnReady := func(i int) bool { return m.ready(m.fns[i], done) }
	emitUnit := func(u []int) { kahn(u, fnReady, mark) }
	kahn(m.units(), unitReady, emitUnit)
	return out
}

func (m *model) units() [][]int {
	var units [][]int
	group := map[string]int{}
	for i := range m.decls {
		x := m.fns[i]
		if x == nil || x.recv == "" || x.exempt {
			units = append(units, []int{i})
			continue
		}
		if u, ok := group[x.recv]; ok {
			units[u] = append(units[u], i)
			continue
		}
		group[x.recv] = len(units)
		units = append(units, []int{i})
	}
	return units
}

func (m *model) unitReady(u []int, done map[int]bool) bool {
	x := m.fns[u[0]]
	if x == nil || x.recv == "" || x.exempt {
		return m.ready(x, done)
	}
	t, ok := m.typeDecl[x.recv]
	withGroup := maps.Clone(done)
	for _, i := range u {
		withGroup[i] = true
	}
	return (!ok || done[t]) && m.ranked(x, withGroup)
}

func (m *model) ready(x *fn, done map[int]bool) bool {
	if x == nil || x.exempt {
		return true
	}
	isDone := func(i int) bool { return done[i] }
	if len(x.callers) > 0 && !slices.ContainsFunc(x.callers, isDone) {
		return false
	}
	return m.ranked(x, done)
}

func (m *model) ranked(y *fn, done map[int]bool) bool {
	for _, i := range m.fnIdx {
		if mustPrecede(m.fns[i], y) && !done[i] {
			return false
		}
	}
	return true
}

func mustPrecede(x, y *fn) bool {
	if x.rank != 0 || x.exempt || y.exempt {
		return false
	}
	if x.recv == "" && y.recv != "" {
		return !ast.IsExported(y.recv)
	}
	return y.recv == x.recv && y.rank == 1
}

func kahn[T any](items []T, ready func(T) bool, emit func(T)) {
	left := slices.Clone(items)
	for len(left) > 0 {
		k := max(slices.IndexFunc(left, ready), 0)
		emit(left[k])
		left = slices.Delete(left, k, k+1)
	}
}

func render(fset *token.FileSet, f *ast.File, src []byte, order []int) []byte {
	tf := fset.File(f.Pos())
	start := lineAfter(src, tf.Offset(f.Name.End()))
	out := slices.Clone(src[:start])
	chunks := make([][]byte, len(f.Decls))
	for i, d := range f.Decls {
		end := lineAfter(src, tf.Offset(d.End()))
		chunks[i] = trimChunk(src[start:end])
		start = end
	}
	for _, i := range order {
		out = append(out, '\n')
		out = append(out, chunks[i]...)
	}
	return append(out, src[start:]...)
}

func lineAfter(src []byte, off int) int {
	return min(analysis.LineEndOffset(src, off)+1, len(src))
}

func trimChunk(chunk []byte) []byte {
	for len(chunk) > 0 && (chunk[0] == '\n' || chunk[0] == '\r') {
		chunk = chunk[1:]
	}
	if len(chunk) == 0 || chunk[len(chunk)-1] != '\n' {
		chunk = append(slices.Clip(chunk), '\n')
	}
	return chunk
}
