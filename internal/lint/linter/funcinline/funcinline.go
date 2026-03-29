package funcinline

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
)

// FindCandidates walks the AST and returns all function
// declarations and function literals whose body can be
// inlined to a single line.
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
	if body == nil {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}
	if len(body.List) != 1 {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	// Already single-line.
	if fset.Position(body.Lbrace).Line == fset.Position(body.Rbrace).Line {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	stmt := body.List[0]

	// Statement spans multiple lines.
	if fset.Position(stmt.Pos()).Line != fset.Position(stmt.End()).Line {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	// Comments inside the body.
	if analysis.HasCommentsBetween(fset, f, body.Lbrace, body.Rbrace) {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	// Statement contains a function literal.
	if analysis.ContainsFuncLit(stmt) {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	// Build collapsed form: signature + "{ " + statement + " }"
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

// Check reports diagnostics for function declarations whose body
// can be inlined.
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
