package funcinline

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// FindCandidates returns functions whose body fits on one line.
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
		c, ok := tryInline(fset, f, fn.Pos(), fn.Body, src, maxLineWidth, tabWidth)
		if ok {
			candidates = append(candidates, c)
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		c, ok := tryInline(fset, f, lit.Pos(), lit.Body, src, maxLineWidth, tabWidth)
		if ok {
			candidates = append(candidates, c)
		}
		return true
	})

	return candidates
}

func tryInline(
	fset *token.FileSet,
	f *ast.File,
	funcPos token.Pos,
	body *ast.BlockStmt,
	src []byte,
	maxLineWidth, tabWidth int,
) (analysis.Candidate, bool) {
	stmt, ok := inlineableStmt(fset, f, body)
	if !ok {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	sigEnd := fset.Position(body.Lbrace).Offset
	signature := string(src[fset.Position(funcPos).Offset:sigEnd])

	stmtStart := fset.Position(stmt.Pos()).Offset
	stmtEnd := fset.Position(stmt.End()).Offset
	stmtText := string(src[stmtStart:stmtEnd])

	collapsed := signature + "{ " + stmtText + " }"

	startOff := fset.Position(funcPos).Offset
	endOff := fset.Position(body.Rbrace).Offset + 1
	if analysis.CollapsedLineWidth(
		src, startOff, endOff, tabWidth, collapsed,
	) > maxLineWidth {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	return analysis.Candidate{
		Pos:       funcPos,
		End:       body.Rbrace + 1,
		Collapsed: collapsed,
	}, true
}

func inlineableStmt(
	fset *token.FileSet,
	f *ast.File,
	body *ast.BlockStmt,
) (ast.Stmt, bool) {
	if body == nil || len(body.List) != 1 {
		return nil, false
	}
	if fset.Position(body.Lbrace).Line == fset.Position(body.Rbrace).Line {
		return nil, false
	}
	stmt := body.List[0]
	if fset.Position(stmt.Pos()).Line != fset.Position(stmt.End()).Line {
		return nil, false
	}
	if analysis.HasCommentsBetween(fset, f, body.Lbrace, body.Rbrace) {
		return nil, false
	}
	if analysis.ContainsFuncLit(stmt) {
		return nil, false
	}
	return stmt, true
}

// Check reports diagnostics for function declarations whose body can be inlined.
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
		"function body can be inlined to single line",
	)
}
