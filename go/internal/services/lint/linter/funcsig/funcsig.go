package funcsig

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for multi-line function signatures that could be collapsed.
func Check(
	fset *token.FileSet,
	filename string,
	f *ast.File,
	src []byte,
	maxLineWidth, tabWidth int,
) []analysis.Diagnostic {
	candidates := FindCandidates(fset, f, src, maxLineWidth, tabWidth)
	return analysis.CheckCandidates(
		fset, filename, candidates,
		"multi-line function signature "+
			"can be collapsed to single line",
	)
}

// FindCandidates returns multi-line function signatures that fit on one line.
func FindCandidates(
	fset *token.FileSet,
	f *ast.File,
	src []byte,
	maxLineWidth, tabWidth int,
) []analysis.Candidate {
	var candidates []analysis.Candidate
	add := func(c analysis.Candidate, ok bool) {
		if ok {
			candidates = append(candidates, c)
		}
	}

	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			add(tryCollapseSig(
				fset, f, fn.Type.Func, fn.Recv,
				fn.Name.Name, fn.Type, fn.Body,
				src, maxLineWidth, tabWidth,
			))
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		if lit, ok := n.(*ast.FuncLit); ok {
			add(tryCollapseSig(
				fset, f, lit.Type.Func, nil,
				"", lit.Type, lit.Body,
				src, maxLineWidth, tabWidth,
			))
		}
		return true
	})

	return candidates
}

func tryCollapseSig(
	fset *token.FileSet,
	f *ast.File,
	funcPos token.Pos,
	recv *ast.FieldList,
	name string,
	funcType *ast.FuncType,
	body *ast.BlockStmt,
	src []byte,
	maxLineWidth, tabWidth int,
) (analysis.Candidate, bool) {
	if body == nil {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	if fset.Position(funcPos).Line == fset.Position(body.Lbrace).Line {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	if analysis.HasCommentsBetween(fset, f, funcPos, body.Lbrace) {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	fields := []*ast.FieldList{
		recv, funcType.TypeParams, funcType.Params, funcType.Results,
	}
	if anyMultiLineField(fset, fields) {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	collapsed := buildCollapsed(fset, recv, name, funcType, src)

	startOff := fset.Position(funcPos).Offset
	endOff := fset.Position(body.Lbrace).Offset + 1
	if analysis.CollapsedLineWidth(
		src, startOff, endOff, tabWidth, collapsed,
	) > maxLineWidth {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	return analysis.Candidate{
		Pos:       funcPos,
		End:       body.Lbrace + 1,
		Collapsed: collapsed,
	}, true
}

func anyMultiLineField(fset *token.FileSet, lists []*ast.FieldList) bool {
	for _, fields := range lists {
		if hasMultiLineField(fset, fields) {
			return true
		}
	}
	return false
}

func hasMultiLineField(fset *token.FileSet, fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}
	for _, field := range fields.List {
		typeStart := fset.Position(field.Type.Pos()).Line
		typeEnd := fset.Position(field.Type.End()).Line
		if typeStart != typeEnd {
			return true
		}
	}
	return false
}

func buildCollapsed(
	fset *token.FileSet,
	recv *ast.FieldList,
	name string,
	funcType *ast.FuncType,
	src []byte,
) string {
	var b strings.Builder

	writeFuncName(&b, fset, recv, name, src)

	if funcType.TypeParams != nil && len(funcType.TypeParams.List) > 0 {
		b.WriteString("[")
		b.WriteString(fieldListText(fset, funcType.TypeParams, src))
		b.WriteString("]")
	}

	b.WriteString("(")
	b.WriteString(fieldListText(fset, funcType.Params, src))
	b.WriteString(")")

	writeResults(&b, fset, funcType.Results, src)

	b.WriteString(" {")
	return b.String()
}

func writeFuncName(
	b *strings.Builder,
	fset *token.FileSet,
	recv *ast.FieldList,
	name string,
	src []byte,
) {
	b.WriteString("func")
	if name == "" {
		return
	}
	b.WriteString(" ")
	if recv != nil && len(recv.List) > 0 {
		b.WriteString("(")
		b.WriteString(fieldListText(fset, recv, src))
		b.WriteString(") ")
	}
	b.WriteString(name)
}

func writeResults(
	b *strings.Builder,
	fset *token.FileSet,
	results *ast.FieldList,
	src []byte,
) {
	if results == nil || len(results.List) == 0 {
		return
	}
	if results.Opening.IsValid() {
		b.WriteString(" (")
		b.WriteString(fieldListText(fset, results, src))
		b.WriteString(")")
		return
	}
	b.WriteString(" ")
	b.WriteString(fieldListText(fset, results, src))
}

func fieldListText(fset *token.FileSet, fields *ast.FieldList, src []byte) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}
	parts := make([]string, len(fields.List))
	for i, field := range fields.List {
		parts[i] = fieldText(fset, field, src)
	}
	return strings.Join(parts, ", ")
}

func fieldText(fset *token.FileSet, field *ast.Field, src []byte) string {
	startOff := fset.Position(field.Type.Pos()).Offset
	endOff := fset.Position(field.Type.End()).Offset
	typeText := string(src[startOff:endOff])
	if len(field.Names) == 0 {
		return typeText
	}
	names := make([]string, len(field.Names))
	for i, n := range field.Names {
		names[i] = n.Name
	}
	return strings.Join(names, ", ") + " " + typeText
}
