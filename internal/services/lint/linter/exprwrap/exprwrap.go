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
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	lines := map[int]int{}
	concat := map[*ast.BinaryExpr]bool{}
	typeParams := map[*ast.FieldList]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.InterfaceType:
			return false
		case *ast.FieldList:
			return !typeParams[node]
		case *ast.FuncType:
			typeParams[node.TypeParams] = true
		case *ast.TypeSpec:
			typeParams[node.TypeParams] = true
		case *ast.BinaryExpr:
			markStringChain(node, concat)
			if !concat[node] && breaks(fset, node.X.End(), node.Y.Pos()) {
				flag(fset, lines, node.Pos(), node.X.End())
			}
		case *ast.SelectorExpr:
			if breaks(fset, node.X.End(), node.Sel.Pos()) {
				flag(fset, lines, node.Pos(), node.X.End())
			}
		}
		return true
	})
	return diagnostics(fset, file, filename, lines)
}

func markStringChain(bin *ast.BinaryExpr, concat map[*ast.BinaryExpr]bool) {
	if _, decided := concat[bin]; bin.Op != token.ADD || decided {
		return
	}
	var nodes []*ast.BinaryExpr
	hasLit := false
	var walk func(ast.Expr)
	walk = func(expr ast.Expr) {
		switch expr := expr.(type) {
		case *ast.ParenExpr:
			walk(expr.X)
		case *ast.BinaryExpr:
			if expr.Op != token.ADD {
				return
			}
			nodes = append(nodes, expr)
			walk(expr.X)
			walk(expr.Y)
		case *ast.BasicLit:
			hasLit = hasLit || expr.Kind == token.STRING
		}
	}
	walk(bin)
	for _, binExpr := range nodes {
		concat[binExpr] = hasLit
	}
}

func breaks(fset *token.FileSet, a, b token.Pos) bool {
	return fset.Position(a).Line != fset.Position(b).Line
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
	file *ast.File,
	filename string,
	lines map[int]int,
) []analysis.Diagnostic {
	diags := make([]analysis.Diagnostic, 0, len(lines))
	for line, start := range lines {
		if analysis.NolintOnLines(fset, file, ruleName, start-1, start) {
			continue
		}
		if analysis.NolintOnLines(fset, file, ruleName, line, line) {
			continue
		}
		diag := analysis.Diagnostic{File: filename, Line: line, Message: message}
		diags = append(diags, diag)
	}
	sort.Slice(diags, func(a, b int) bool { return diags[a].Line < diags[b].Line })
	return diags
}
