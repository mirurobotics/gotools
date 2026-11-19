package exprwrap

import (
	"go/ast"
	"go/token"
	"sort"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const (
	ruleName = "exprwrap"
	message  = "expression wraps across lines — extract it into a function"
)

// Check flags expressions that wrap at a binary operator or selector dot.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	lines := map[int]int{}
	concat := map[*ast.BinaryExpr]bool{}
	typeParams := map[*ast.FieldList]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.InterfaceType:
			return false
		case *ast.FieldList:
			return !typeParams[x]
		case *ast.FuncType:
			typeParams[x.TypeParams] = true
		case *ast.TypeSpec:
			typeParams[x.TypeParams] = true
		case *ast.BinaryExpr:
			markStringChain(x, concat)
			if !concat[x] && breaks(fset, x.X.End(), x.Y.Pos()) {
				flag(fset, lines, x.Pos(), x.X.End())
			}
		case *ast.SelectorExpr:
			if breaks(fset, x.X.End(), x.Sel.Pos()) {
				flag(fset, lines, x.Pos(), x.X.End())
			}
		}
		return true
	})
	return diagnostics(fset, f, filename, lines)
}

func breaks(fset *token.FileSet, a, b token.Pos) bool {
	return fset.Position(a).Line != fset.Position(b).Line
}

func markStringChain(e *ast.BinaryExpr, concat map[*ast.BinaryExpr]bool) {
	if _, decided := concat[e]; e.Op != token.ADD || decided {
		return
	}
	var nodes []*ast.BinaryExpr
	hasLit := false
	var walk func(ast.Expr)
	walk = func(x ast.Expr) {
		switch v := x.(type) {
		case *ast.ParenExpr:
			walk(v.X)
		case *ast.BinaryExpr:
			if v.Op != token.ADD {
				return
			}
			nodes = append(nodes, v)
			walk(v.X)
			walk(v.Y)
		case *ast.BasicLit:
			hasLit = hasLit || v.Kind == token.STRING
		}
	}
	walk(e)
	for _, n := range nodes {
		concat[n] = hasLit
	}
}

func flag(fset *token.FileSet, lines map[int]int, start, brk token.Pos) {
	line := fset.Position(brk).Line
	from := fset.Position(start).Line
	if prev, ok := lines[line]; !ok || from < prev {
		lines[line] = from
	}
}

func diagnostics(
	fset *token.FileSet,
	f *ast.File,
	filename string,
	lines map[int]int,
) []analysis.Diagnostic {
	diags := make([]analysis.Diagnostic, 0, len(lines))
	for line, start := range lines {
		if analysis.NolintOnLines(fset, f, ruleName, start-1, line) {
			continue
		}
		diag := analysis.Diagnostic{File: filename, Line: line, Message: message}
		diags = append(diags, diag)
	}
	sort.Slice(diags, func(i, j int) bool { return diags[i].Line < diags[j].Line })
	return diags
}
