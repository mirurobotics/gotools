package collapse

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// collapseInput captures the AST fields needed to evaluate
// and build a collapse candidate.
type collapseInput struct {
	exprStart token.Pos  // Fun.Pos() or Type.Pos() or Lbrace
	openPos   token.Pos  // ( or {
	closePos  token.Pos  // ) or }
	items     []ast.Expr // Args or Elts
	openStr   string     // "(" or "{"
	closeStr  string     // ")" or "}"
	ellipsis  bool       // variadic call?
}

// FindCandidates walks the AST and returns all multi-line
// call expressions and composite literals that can safely be
// collapsed to a single line.
func FindCandidates(
	fset *token.FileSet,
	f *ast.File,
	src []byte,
	maxLineWidth, tabWidth int,
) []analysis.Candidate {
	var candidates []analysis.Candidate
	var stack []ast.Node

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return false
		}

		var input collapseInput
		switch expr := n.(type) {
		case *ast.CallExpr:
			input = collapseInputFromCall(expr)
		case *ast.CompositeLit:
			input = collapseInputFromLit(expr)
		default:
			stack = append(stack, n)
			return true
		}

		delta := kvAlignDelta(fset, stack, n, src, tabWidth)
		if c, ok := tryCollapse(
			fset, input, src, maxLineWidth, tabWidth, f, delta,
		); ok {
			candidates = append(candidates, c)
		}
		stack = append(stack, n)
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

func tryCollapse(
	fset *token.FileSet,
	input collapseInput,
	src []byte,
	maxLineWidth, tabWidth int,
	f *ast.File,
	alignDelta int,
) (analysis.Candidate, bool) {
	if shouldExclude(fset, f, input) {
		return analysis.Candidate{}, false //nolint:exhaustruct
	}
	return buildCollapsed(fset, input, src, maxLineWidth, tabWidth, alignDelta)
}

func shouldExclude(fset *token.FileSet, f *ast.File, input collapseInput) bool {
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
	if analysis.HasCommentsBetween(fset, f, input.openPos, input.closePos) {
		return true
	}
	return hasRawStringWithNewlines(input.items)
}

// kvAlignDelta computes the extra indent that gofmt alignment
// will add when the expression being collapsed is a value in a
// key-value pair inside a struct or map literal.
func kvAlignDelta(
	fset *token.FileSet,
	stack []ast.Node,
	n ast.Node,
	src []byte,
	tabWidth int,
) int {
	nLine := fset.Position(n.Pos()).Line
	for i := len(stack) - 1; i >= 1; i-- {
		kv, ok := stack[i].(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if n.Pos() < kv.Value.Pos() {
			continue
		}
		if fset.Position(kv.Key.Pos()).Line != nLine {
			continue
		}
		parent, ok := stack[i-1].(*ast.CompositeLit)
		if !ok {
			continue
		}
		return keyAlignDelta(fset, kv, parent, src, tabWidth)
	}
	return 0
}

func keyAlignDelta(
	fset *token.FileSet,
	kv *ast.KeyValueExpr,
	parent *ast.CompositeLit,
	src []byte,
	tabWidth int,
) int {
	currentW := nodeWidth(fset, kv.Key, src, tabWidth)
	maxW := currentW
	for _, elt := range parent.Elts {
		sib, ok := elt.(*ast.KeyValueExpr)
		if !ok || sib == kv {
			continue
		}
		if w := nodeWidth(fset, sib.Key, src, tabWidth); w > maxW {
			maxW = w
		}
	}
	return maxW - currentW
}

func nodeWidth(fset *token.FileSet, n ast.Node, src []byte, tabWidth int) int {
	start := fset.Position(n.Pos()).Offset
	end := fset.Position(n.End()).Offset
	return analysis.VisualWidth(string(src[start:end]), tabWidth)
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

// hasRawStringWithNewlines returns true if any expression in the list contains
// a raw string literal (backtick-delimited) with embedded newlines.
func hasRawStringWithNewlines(exprs []ast.Expr) bool {
	for _, expr := range exprs {
		found := false
		ast.Inspect(expr, func(n ast.Node) bool {
			if found {
				return false
			}
			lit, ok := n.(*ast.BasicLit)
			if ok && lit.Kind == token.STRING {
				if strings.HasPrefix(lit.Value, "`") &&
					strings.Contains(lit.Value, "\n") {
					found = true
					return false
				}
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

// Check reports diagnostics for multi-line expressions that could be collapsed.
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
		"multi-line expression can be collapsed to single line",
	)
}
