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
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	graph := build(file)
	order, below := graph.order()
	var diags []analysis.Diagnostic
	add := func(decl ast.Decl, msg string) {
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    fset.Position(decl.Pos()).Line,
			Message: msg,
		})
	}
	for _, fnc := range graph.funcs() {
		for _, msg := range graph.violations(fnc, order, below) {
			add(fnc.decl, msg)
		}
	}
	uses := graph.firstUses(graph.units(false), func(i int) int { return i })
	for _, typeIdx := range slices.Sorted(maps.Keys(uses)) {
		if msg := graph.typeViolation(typeIdx, uses); msg != "" {
			add(graph.decls[typeIdx], msg)
		}
	}
	return diags
}

// Fix returns src with its declarations reordered, or nil when already ordered.
func Fix(fset *token.FileSet, file *ast.File, src []byte) []byte {
	order, _ := build(file).order()
	if slices.IsSorted(order) {
		return nil
	}
	return render(fset, file, src, order)
}

func render(fset *token.FileSet, file *ast.File, src []byte, order []int) []byte {
	tfile := fset.File(file.Pos())
	start := boundary(tfile, file, src, file.Name.End(), 0)
	out := slices.Clone(src[:start])
	chunks := make([][]byte, len(file.Decls))
	for i, decl := range file.Decls {
		end := boundary(tfile, file, src, decl.End(), i+1)
		chunks[i] = trimChunk(src[start:end])
		start = end
	}
	for _, i := range order {
		out = append(out, '\n')
		out = append(out, chunks[i]...)
	}
	return append(out, src[start:]...)
}

func boundary(
	tfile *token.File, file *ast.File, src []byte, end token.Pos, next int,
) int {
	end = extendEnd(tfile, file, end)
	if next < len(file.Decls) && tfile.Line(file.Decls[next].Pos()) == tfile.Line(end) {
		return tfile.Offset(end)
	}
	return lineAfter(src, tfile.Offset(end))
}

func extendEnd(tfile *token.File, file *ast.File, end token.Pos) token.Pos {
	for _, group := range file.Comments {
		if group.Pos() >= end && tfile.Line(group.Pos()) == tfile.Line(end) {
			end = group.End()
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

func (m *model) violations(fnc *fn, order []int, below map[int]bool) []string {
	checks := []func(*fn) string{m.stepdown, m.rankOrder, m.belowType, m.grouped}
	if fnc.exempt {
		checks = checks[2:]
	}
	var msgs []string
	for _, check := range checks {
		if msg := check(fnc); msg != "" {
			msgs = append(msgs, msg)
		}
	}
	if len(msgs) > 0 || !below[fnc.idx] {
		return msgs
	}
	return m.directlyBelow(fnc, order)
}

func (m *model) stepdown(fnc *fn) string {
	if len(fnc.callers) == 0 || slices.Max(fnc.callers) < fnc.idx {
		return ""
	}
	last := m.fns[slices.Max(fnc.callers)].name
	return fmt.Sprintf("%q should be below its last caller %q", fnc.name, last)
}

func (m *model) rankOrder(fnc *fn) string {
	if fnc.rank != 0 {
		return ""
	}
	for _, other := range m.funcs() {
		if other.idx < fnc.idx && mustPrecede(fnc, other) {
			return fmt.Sprintf("%q should be above unexported %q", fnc.name, other.name)
		}
	}
	return ""
}

func (m *model) belowType(fnc *fn) string {
	if typeIdx, ok := m.typeDecl[fnc.recv]; !ok || typeIdx < fnc.idx {
		return ""
	}
	return fmt.Sprintf("method %q should be below type %q", fnc.name, fnc.recv)
}

func (m *model) grouped(fnc *fn) string {
	prev := -1
	for _, other := range m.funcs() {
		if other.idx < fnc.idx && fnc.recv != "" && other.recv == fnc.recv {
			prev = other.idx
		}
	}
	if prev >= 0 && prev != fnc.idx-1 {
		return fmt.Sprintf(
			"method %q should be grouped with the other methods of %q",
			fnc.name, fnc.recv,
		)
	}
	typeIdx, ok := m.typeDecl[fnc.recv]
	headed := ok && typeIdx < fnc.idx && owns(m.typeOwner, fnc.idx, typeIdx)
	if prev >= 0 || !headed || owns(m.typeOwner, fnc.idx, fnc.idx-1) {
		return ""
	}
	return fmt.Sprintf("type %q should be directly above its methods", fnc.recv)
}

func (m *model) directlyBelow(fnc *fn, order []int) []string {
	pos, first := slices.Index(order, fnc.idx), fnc.idx
	for m.fns[order[pos-1]] == nil {
		pos--
		if !m.inPlain(order[pos]) {
			first = order[pos]
		}
	}
	prev := first - 1
	for prev >= 0 && m.inPlain(prev) {
		prev--
	}
	above := order[pos-1]
	if above == prev {
		return nil
	}
	msg := fmt.Sprintf("%q should be directly below %q", fnc.name, m.declName(above))
	return []string{msg}
}
