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

// Check reports a diagnostic for each expression that breaks across lines
// at a binary operator (`a ||\n b`) or at a selector dot (`x.\n Y()`): if
// an expression cannot fit on one line, it belongs in its own function.
//
// Only those two kinds of break count, so these are fine:
//   - call arguments spread over lines between ( and ), though an
//     argument that itself wraps at an operator is still flagged
//   - composite literals such as table tests and struct literals, though
//     a key or value that itself wraps is still flagged
//   - function literals; their bodies are checked like any other code
//   - raw string literals
//   - the + in a + chain that has a string literal operand (string
//     building, not logic); breaks inside an operand are still flagged
//
// Interface types and type-parameter lists are skipped, so type-set
// unions may wrap.
//
// A //nolint:exprwrap comment on the flagged line, or on the line above
// it, suppresses the diagnostic.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	lines := map[int]bool{}
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
				lines[fset.Position(x.Pos()).Line] = true
			}
		case *ast.SelectorExpr:
			if breaks(fset, x.X.End(), x.Sel.Pos()) {
				lines[fset.Position(x.Pos()).Line] = true
			}
		}
		return true
	})
	return diagnostics(filename, lines, nolintLines(fset, f))
}

// breaks reports whether a and b sit on different lines.
func breaks(fset *token.FileSet, a, b token.Pos) bool {
	return fset.Position(a).Line != fset.Position(b).Line
}

// markStringChain records, for every + node of the chain rooted at e,
// whether any operand of the chain is a string literal. Inspect visits
// the root first, so the whole chain is decided at once.
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

// diagnostics builds one diagnostic per flagged line, skipping lines a
// nolint comment covers, in line order.
func diagnostics(filename string, lines, nolint map[int]bool) []analysis.Diagnostic {
	diags := make([]analysis.Diagnostic, 0, len(lines))
	for line := range lines {
		if nolint[line] {
			continue
		}
		diag := analysis.Diagnostic{File: filename, Line: line, Message: message}
		diags = append(diags, diag)
	}
	sort.Slice(diags, func(i, j int) bool { return diags[i].Line < diags[j].Line })
	return diags
}

// nolintLines returns the lines a //nolint:exprwrap comment covers: its
// own line and the line below it.
func nolintLines(fset *token.FileSet, f *ast.File) map[int]bool {
	lines := map[int]bool{}
	for _, cg := range f.Comments {
		for _, cm := range cg.List {
			if !analysis.NolintsRule(cm.Text, ruleName) {
				continue
			}
			line := fset.Position(cm.Pos()).Line
			lines[line] = true
			lines[line+1] = true
		}
	}
	return lines
}
