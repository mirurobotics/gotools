package funcorder

import (
	"bytes"
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
	add := func(d ast.Decl, msg string) {
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    fset.Position(d.Pos()).Line,
			Message: msg,
		})
	}
	for _, x := range m.funcs() {
		for _, msg := range m.violations(x) {
			add(x.decl, msg)
		}
	}
	uses := m.firstUses(m.units(m.heads), func(i int) int { return i })
	for _, t := range slices.Sorted(maps.Keys(uses)) {
		if !m.typePlaced(t, uses) {
			add(m.decls[t], m.typeMessage(t, uses[t]))
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
	if x.exempt {
		checks = checks[2:]
	}
	var msgs []string
	for _, check := range checks {
		if msg := check(x); msg != "" {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

func (m *model) stepdown(x *fn) string {
	if len(x.callers) == 0 || slices.Max(x.callers) < x.idx {
		return ""
	}
	last := m.fns[slices.Max(x.callers)].name
	return fmt.Sprintf("%q should be below its last caller %q", x.name, last)
}

func (m *model) rankOrder(x *fn) string {
	if x.rank != 0 {
		return ""
	}
	for _, y := range m.funcs() {
		if y.idx < x.idx && mustPrecede(x, y) {
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
	for _, y := range m.funcs() {
		if y.idx < x.idx && x.recv != "" && y.recv == x.recv {
			prev = y.idx
		}
	}
	if prev >= 0 && prev != x.idx-1 {
		return fmt.Sprintf(
			"method %q should be grouped with the other methods of %q", x.name, x.recv,
		)
	}
	head, ok := m.heads[x.recv]
	if prev >= 0 || !ok || head[0] > x.idx || m.placed(x.recv) {
		return ""
	}
	return fmt.Sprintf("type %q should be directly above its methods", x.recv)
}

func (m *model) order() []int {
	out := m.arrange(m.heads)
	if m.keepsInitOrder(out) {
		return out
	}
	return m.arrange(m.settledHeads())
}

func (m *model) arrange(heads map[string][]int) []int {
	done := map[int]bool{}
	var out []int
	mark := func(i int) {
		done[i] = true
		out = append(out, i)
	}
	unitReady := func(u []int) bool { return m.unitReady(u, done) }
	fnReady := func(i int) bool { return m.ready(m.fns[i], done) }
	emitUnit := func(u []int) { kahn(u, fnReady, mark) }
	units := m.units(heads)
	kahn(units, unitReady, emitUnit)
	return m.placeTypes(units, out)
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

func (m *model) units(heads map[string][]int) [][]int {
	var units [][]int
	group := map[string]int{}
	glued := map[int]bool{}
	for _, head := range heads {
		for _, i := range head {
			glued[i] = true
		}
	}
	for i, x := range m.fns {
		switch {
		case glued[i] || m.inPlain(i):
		case x == nil || x.recv == "":
			units = append(units, []int{i})
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

func (m *model) unitReady(u []int, done map[int]bool) bool {
	x := m.fns[u[0]]
	if x == nil || x.recv == "" {
		return m.ready(x, done)
	}
	t, ok := m.typeDecl[x.recv]
	return !ok || done[t]
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
	start := boundary(tf, f, src, f.Name.End(), 0)
	out := slices.Clone(src[:start])
	chunks := make([][]byte, len(f.Decls))
	for i, d := range f.Decls {
		end := boundary(tf, f, src, d.End(), i+1)
		chunks[i] = trimChunk(src[start:end])
		start = end
	}
	for _, i := range order {
		out = append(out, '\n')
		out = append(out, chunks[i]...)
	}
	return append(out, src[start:]...)
}

func boundary(tf *token.File, f *ast.File, src []byte, end token.Pos, next int) int {
	end = extendEnd(tf, f, end)
	if next < len(f.Decls) && tf.Line(f.Decls[next].Pos()) == tf.Line(end) {
		return tf.Offset(end)
	}
	return lineAfter(src, tf.Offset(end))
}

func extendEnd(tf *token.File, f *ast.File, end token.Pos) token.Pos {
	for _, cg := range f.Comments {
		if cg.Pos() >= end && tf.Line(cg.Pos()) == tf.Line(end) {
			end = cg.End()
		}
	}
	return end
}

func lineAfter(src []byte, off int) int {
	return min(analysis.LineEndOffset(src, off)+1, len(src))
}

func trimChunk(chunk []byte) []byte {
	chunk = bytes.TrimLeft(chunk, " \t\r\n;")
	if len(chunk) == 0 || chunk[len(chunk)-1] != '\n' {
		chunk = append(slices.Clip(chunk), '\n')
	}
	return chunk
}
