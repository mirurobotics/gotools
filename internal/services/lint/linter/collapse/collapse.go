package collapse

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports diagnostics for multi-line expressions that could be collapsed.
func Check(
	fset *token.FileSet,
	filename string,
	file *ast.File,
	src []byte,
	maxLineWidth, tabWidth int,
) []analysis.Diagnostic {
	candidates := FindCandidates(fset, file, src, maxLineWidth, tabWidth)
	return analysis.CheckCandidates(
		fset, filename, candidates,
		"multi-line expression can be collapsed to single line",
	)
}

type collapseInput struct {
	exprStart token.Pos
	openPos   token.Pos
	closePos  token.Pos
	items     []ast.Expr
	openStr   string
	closeStr  string
	ellipsis  bool
}

// FindCandidates returns multi-line calls and literals that fit on one line.
func FindCandidates(
	fset *token.FileSet,
	file *ast.File,
	src []byte,
	maxLineWidth, tabWidth int,
) []analysis.Candidate {
	var candidates []analysis.Candidate
	var stack []ast.Node

	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}

		var input collapseInput
		switch expr := node.(type) {
		case *ast.CallExpr:
			input = collapseInputFromCall(expr)
		case *ast.CompositeLit:
			input = collapseInputFromLit(expr)
		default:
			stack = append(stack, node)
			return true
		}

		delta := kvAlignDelta(fset, stack, node, src, tabWidth)
		if cand, ok := tryCollapse(
			fset, input, src, maxLineWidth, tabWidth, file, delta,
		); ok {
			candidates = append(candidates, cand)
		}
		stack = append(stack, node)
		return true
	})

	return candidates
}

func collapseInputFromCall(call *ast.CallExpr) collapseInput {
	return collapseInput{
		exprStart: call.Fun.Pos(),
		openPos:   call.Lparen,
		closePos:  call.Rparen,
		items:     call.Args,
		openStr:   "(",
		closeStr:  ")",
		ellipsis:  call.Ellipsis.IsValid(),
	}
}

func collapseInputFromLit(lit *ast.CompositeLit) collapseInput {
	start := lit.Lbrace
	if lit.Type != nil {
		start = lit.Type.Pos()
	}
	return collapseInput{
		exprStart: start,
		openPos:   lit.Lbrace,
		closePos:  lit.Rbrace,
		items:     lit.Elts,
		openStr:   "{",
		closeStr:  "}",
		ellipsis:  false,
	}
}

func kvAlignDelta(
	fset *token.FileSet,
	stack []ast.Node,
	node ast.Node,
	src []byte,
	tabWidth int,
) int {
	nLine := fset.Position(node.Pos()).Line
	for i := len(stack) - 1; i >= 1; i-- {
		pair, ok := stack[i].(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if node.Pos() < pair.Value.Pos() {
			continue
		}
		if fset.Position(pair.Key.Pos()).Line != nLine {
			continue
		}
		parent, ok := stack[i-1].(*ast.CompositeLit)
		if !ok {
			continue
		}
		return keyAlignDelta(fset, pair, parent, src, tabWidth)
	}
	return 0
}

func keyAlignDelta(
	fset *token.FileSet,
	pair *ast.KeyValueExpr,
	parent *ast.CompositeLit,
	src []byte,
	tabWidth int,
) int {
	currentW := nodeWidth(fset, pair.Key, src, tabWidth)
	maxW := currentW
	for _, elt := range parent.Elts {
		sib, ok := elt.(*ast.KeyValueExpr)
		if !ok || sib == pair {
			continue
		}
		if width := nodeWidth(fset, sib.Key, src, tabWidth); width > maxW {
			maxW = width
		}
	}
	return maxW - currentW
}

func nodeWidth(fset *token.FileSet, node ast.Node, src []byte, tabWidth int) int {
	start := fset.Position(node.Pos()).Offset
	end := fset.Position(node.End()).Offset
	return analysis.VisualWidth(string(src[start:end]), tabWidth)
}

func tryCollapse(
	fset *token.FileSet,
	input collapseInput,
	src []byte,
	maxLineWidth, tabWidth int,
	file *ast.File,
	alignDelta int,
) (analysis.Candidate, bool) {
	if shouldExclude(fset, file, input) {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}
	return buildCollapsed(fset, input, src, maxLineWidth, tabWidth, alignDelta)
}

func shouldExclude(fset *token.FileSet, file *ast.File, input collapseInput) bool {
	if fset.Position(input.openPos).Line == fset.Position(input.closePos).Line {
		return true
	}
	if len(input.items) == 0 {
		return true
	}
	for _, item := range input.items {
		if analysis.ContainsFuncLit(item) {
			return true
		}
		if fset.Position(item.Pos()).Line != fset.Position(item.End()).Line {
			return true
		}
	}
	if analysis.HasCommentsBetween(fset, file, input.openPos, input.closePos) {
		return true
	}
	return hasRawStringWithNewlines(input.items)
}

func hasRawStringWithNewlines(exprs []ast.Expr) bool {
	for _, expr := range exprs {
		found := false
		ast.Inspect(expr, func(node ast.Node) bool {
			if found {
				return false
			}
			lit, ok := node.(*ast.BasicLit)
			if ok && isMultiLineRawString(lit) {
				found = true
				return false
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

func isMultiLineRawString(lit *ast.BasicLit) bool {
	if lit.Kind != token.STRING || !strings.HasPrefix(lit.Value, "`") {
		return false
	}
	return strings.Contains(lit.Value, "\n")
}

func buildCollapsed(
	fset *token.FileSet,
	input collapseInput,
	src []byte,
	maxLineWidth, tabWidth, alignDelta int,
) (analysis.Candidate, bool) {
	openOff := fset.Position(input.openPos).Offset
	exprStartOff := fset.Position(input.exprStart).Offset
	prefix := string(src[exprStartOff:openOff])

	texts := make([]string, len(input.items))
	for i, item := range input.items {
		startOff := fset.Position(item.Pos()).Offset
		endOff := fset.Position(item.End()).Offset
		texts[i] = string(src[startOff:endOff])
	}
	if input.ellipsis {
		texts[len(texts)-1] += "..."
	}

	collapsed := prefix + input.openStr + strings.Join(texts, ", ") + input.closeStr

	endOff := fset.Position(input.closePos).Offset + 1
	total := analysis.CollapsedLineWidth(
		src, exprStartOff, endOff, tabWidth, collapsed,
	) + alignDelta
	if total > maxLineWidth {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}

	return analysis.Candidate{
		Pos:       input.exprStart,
		End:       input.closePos + 1,
		Collapsed: collapsed,
	}, true
}
