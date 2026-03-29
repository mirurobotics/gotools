package funcsig

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
)

// FindCandidates walks the AST and returns all multi-line
// function signatures that can safely be collapsed to a
// single line.
func FindCandidates(
	fset *token.FileSet,
	f *ast.File,
	src []byte,
	maxLineWidth, tabWidth int,
) []analysis.Candidate {
	var candidates []analysis.Candidate

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		c, ok := tryCollapseSig(
			fset, f, fn.Type.Func, fn.Recv,
			fn.Name.Name, fn.Type, fn.Body,
			src, maxLineWidth, tabWidth,
		)
		if ok {
			candidates = append(candidates, c)
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		c, ok := tryCollapseSig(
			fset, f, lit.Type.Func, nil,
			"", lit.Type, lit.Body,
			src, maxLineWidth, tabWidth,
		)
		if ok {
			candidates = append(candidates, c)
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
	// No body (external/assembly declaration).
	if body == nil {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	// Already single-line.
	if fset.Position(funcPos).Line == fset.Position(body.Lbrace).Line {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	// Comments between func keyword and opening brace.
	if analysis.HasCommentsBetween(fset, f, funcPos, body.Lbrace) {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	// Any field in recv/type-params/params/results spans multiple lines.
	if hasMultiLineField(fset, recv) ||
		hasMultiLineField(fset, funcType.TypeParams) ||
		hasMultiLineField(fset, funcType.Params) ||
		hasMultiLineField(fset, funcType.Results) {
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

	if name != "" {
		b.WriteString("func ")
		if recv != nil && len(recv.List) > 0 {
			b.WriteString("(")
			b.WriteString(fieldListText(fset, recv, src))
			b.WriteString(") ")
		}
		b.WriteString(name)
	} else {
		b.WriteString("func")
	}

	if funcType.TypeParams != nil && len(funcType.TypeParams.List) > 0 {
		b.WriteString("[")
		b.WriteString(fieldListText(fset, funcType.TypeParams, src))
		b.WriteString("]")
	}

	b.WriteString("(")
	b.WriteString(fieldListText(fset, funcType.Params, src))
	b.WriteString(")")

	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		if funcType.Results.Opening.IsValid() {
			b.WriteString(" (")
			b.WriteString(fieldListText(fset, funcType.Results, src))
			b.WriteString(")")
		} else {
			b.WriteString(" ")
			b.WriteString(fieldListText(fset, funcType.Results, src))
		}
	}

	b.WriteString(" {")
	return b.String()
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

// Check reports diagnostics for multi-line function signatures
// that could be collapsed.
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
