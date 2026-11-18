package commentedcode

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const message = "commented-out code; delete it instead"

// Check reports comment groups that contain Go code which has been commented out.
func Check(
	fset *token.FileSet, filename string, f *ast.File, src []byte,
) []analysis.Diagnostic {
	outputs := analysis.ExampleOutputs(filename, f)
	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		if outputs[cg] || isTrailing(fset, src, cg) {
			continue
		}
		if c := findCode(cg); c != nil {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    fset.Position(c.Pos()).Line,
				Message: message,
			})
		}
	}
	return diags
}

func isTrailing(fset *token.FileSet, src []byte, cg *ast.CommentGroup) bool {
	off := fset.Position(cg.Pos()).Offset
	start := analysis.LineStartOffset(src, off)
	return strings.TrimSpace(string(src[start:off])) != ""
}

func findCode(cg *ast.CommentGroup) *ast.Comment {
	for _, run := range runs(cg) {
		if isCode(joinText(run)) {
			return run[0]
		}
		if len(run) == 1 {
			continue
		}
		for _, c := range run {
			if isCode(text(c)) {
				return c
			}
		}
	}
	return nil
}

// runs splits a group into line-comment runs, breaking at directives and indents.
func runs(cg *ast.CommentGroup) [][]*ast.Comment {
	var out [][]*ast.Comment
	var cur []*ast.Comment
	flush := func() {
		if len(cur) > 0 {
			out = append(out, cur)
			cur = nil
		}
	}
	for _, c := range cg.List {
		switch {
		case analysis.IsDirective(c.Text) || isIndented(c.Text):
			flush()
		case strings.HasPrefix(c.Text, "/*"):
			flush()
			out = append(out, []*ast.Comment{c})
		default:
			cur = append(cur, c)
		}
	}
	flush()
	return out
}

func isIndented(t string) bool {
	return strings.HasPrefix(t, "//\t") || strings.HasPrefix(t, "//  ")
}

func text(c *ast.Comment) string {
	if strings.HasPrefix(c.Text, "/*") {
		return strings.TrimSuffix(strings.TrimPrefix(c.Text, "/*"), "*/")
	}
	return strings.TrimPrefix(strings.TrimPrefix(c.Text, "//"), " ")
}

func joinText(run []*ast.Comment) string {
	lines := make([]string, len(run))
	for i, c := range run {
		lines[i] = text(c)
	}
	return strings.Join(lines, "\n")
}

func isCode(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	return isDecls(s) || isStmts(s)
}

func isDecls(s string) bool {
	f, err := parser.ParseFile(token.NewFileSet(), "", "package p\n"+s+"\n", 0)
	return err == nil && len(f.Decls) > 0
}

func isStmts(s string) bool {
	src := "package p\nfunc _() {\n" + s + "\n}\n"
	f, err := parser.ParseFile(token.NewFileSet(), "", src, 0)
	if err != nil {
		return false
	}
	fn, ok := f.Decls[0].(*ast.FuncDecl)
	if !ok {
		return false
	}
	for _, st := range fn.Body.List {
		if !plausible(st) {
			return false
		}
	}
	return hasCodeNode(fn.Body)
}

// plausible rejects statements that parse but cannot compile, as prose often does.
func plausible(st ast.Stmt) bool {
	switch x := st.(type) {
	case *ast.LabeledStmt:
		return false
	case *ast.ExprStmt:
		return isCallOrRecv(x.X)
	case *ast.AssignStmt:
		for _, e := range x.Lhs {
			if _, ok := e.(*ast.CallExpr); ok {
				return false
			}
		}
	}
	return true
}

func isCallOrRecv(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.CallExpr:
		return true
	case *ast.UnaryExpr:
		return x.Op == token.ARROW
	case *ast.ParenExpr:
		return isCallOrRecv(x.X)
	}
	return false
}

// hasCodeNode reports whether body holds a node that prose rarely parses as.
func hasCodeNode(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.CallExpr, *ast.AssignStmt, *ast.IncDecStmt,
			*ast.ReturnStmt, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt,
			*ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt,
			*ast.GoStmt, *ast.DeferStmt, *ast.DeclStmt, *ast.SendStmt, *ast.ExprStmt:
			found = true
		}
		return !found
	})
	return found
}
