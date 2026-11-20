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
	order, below := m.order()
	var diags []analysis.Diagnostic
	add := func(d ast.Decl, msg string) {
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    fset.Position(d.Pos()).Line,
			Message: msg,
		})
	}
	for _, x := range m.funcs() {
		for _, msg := range m.violations(x, order, below) {
			add(x.decl, msg)
		}
	}
	uses := m.firstUses(m.units(false), func(i int) int { return i })
	for _, t := range slices.Sorted(maps.Keys(uses)) {
		if !m.typePlaced(t, uses) {
			add(m.decls[t], m.typeMessage(t, uses[t]))
		}
	}
	return diags
}

// Fix returns src with its declarations reordered, or nil when already ordered.
func Fix(fset *token.FileSet, f *ast.File, src []byte) []byte {
	order, _ := build(f).order()
	if slices.IsSorted(order) {
		return nil
	}
	return render(fset, f, src, order)
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

func (m *model) violations(x *fn, order []int, below map[int]bool) []string {
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
	if len(msgs) > 0 || !below[x.idx] {
		return msgs
	}
	return m.directlyBelow(x, order)
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

func (m *model) directlyBelow(x *fn, order []int) []string {
	k := slices.Index(order, x.idx)
	for m.fns[order[k-1]] == nil {
		k--
	}
	above := order[k-1]
	if above == order[k]-1 {
		return nil
	}
	msg := fmt.Sprintf("%q should be directly below %q", x.name, m.declName(above))
	return []string{msg}
}
