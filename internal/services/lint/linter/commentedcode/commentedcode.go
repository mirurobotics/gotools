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
	preambles := cgoPreambles(f)
	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		if outputs[cg] || preambles[cg] || isTrailing(fset, src, cg) {
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

// line is one line of comment text with the comment it came from.
type line struct {
	c        *ast.Comment
	text     string
	indented bool
}

// findCode tries each run whole, then its unindented lines one at a time.
func findCode(cg *ast.CommentGroup) *ast.Comment {
	for _, run := range runs(cg) {
		if isCode(joinText(run)) {
			return run[0].c
		}
		if len(run) == 1 {
			continue
		}
		for _, l := range run {
			if !l.indented && isCode(l.text) {
				return l.c
			}
		}
	}
	return nil
}

// runs splits a group into runs of comment lines, breaking at directives.
func runs(cg *ast.CommentGroup) [][]line {
	var out [][]line
	var cur []line
	for _, c := range cg.List {
		if analysis.IsDirective(c.Text) {
			if len(cur) > 0 {
				out = append(out, cur)
			}
			cur = nil
			continue
		}
		cur = append(cur, lines(c)...)
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func lines(c *ast.Comment) []line {
	if body, ok := strings.CutPrefix(c.Text, "/*"); ok {
		texts := strings.Split(strings.TrimSuffix(body, "*/"), "\n")
		prefix := commonIndent(texts[1:])
		out := make([]line, len(texts))
		for i, t := range texts {
			t = strings.TrimPrefix(t, prefix)
			out[i] = line{c: c, text: t, indented: isIndented(t)}
		}
		return out
	}
	body := strings.TrimPrefix(c.Text, "//")
	text := strings.TrimPrefix(body, " ")
	return []line{{c: c, text: text, indented: isIndented(body)}}
}

func isIndented(s string) bool {
	return strings.HasPrefix(s, "\t") || strings.HasPrefix(s, "  ")
}

// commonIndent returns the leading whitespace shared by every non-blank line.
func commonIndent(texts []string) string {
	prefix, first := "", true
	for _, t := range texts {
		if strings.TrimSpace(t) == "" {
			continue
		}
		ws := t[:len(t)-len(strings.TrimLeft(t, " \t"))]
		if first {
			prefix, first = ws, false
		}
		for !strings.HasPrefix(ws, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

// cgoPreambles returns the comment groups that hold cgo preambles.
func cgoPreambles(f *ast.File) map[*ast.CommentGroup]bool {
	m := map[*ast.CommentGroup]bool{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		for _, spec := range gd.Specs {
			is, ok := spec.(*ast.ImportSpec)
			if ok && is.Path.Value == `"C"` {
				m[gd.Doc], m[is.Doc] = true, true
			}
		}
	}
	return m
}

func joinText(run []line) string {
	texts := make([]string, len(run))
	for i, l := range run {
		texts[i] = l.text
	}
	return strings.Join(texts, "\n")
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
	body := firstBody(f)
	for _, st := range body.List {
		if !plausible(st) {
			return false
		}
	}
	return hasCodeNode(body)
}

func firstBody(f *ast.File) *ast.BlockStmt {
	var body *ast.BlockStmt
	ast.Inspect(f, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && body == nil {
			body = fd.Body
		}
		return body == nil
	})
	return body
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
			if !isAssignable(e) {
				return false
			}
		}
	}
	return true
}

func isAssignable(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident, *ast.SelectorExpr, *ast.IndexExpr, *ast.StarExpr:
		return true
	case *ast.ParenExpr:
		return isAssignable(x.X)
	}
	return false
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
