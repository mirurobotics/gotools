package commentedcode

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"unicode/utf8"
)

func isCode(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	return isDecls(s) || isStmts(s)
}

func isDecls(s string) bool {
	src := "package p\n" + s + "\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil || len(f.Decls) == 0 || isProse(fset, src, f) {
		return false
	}
	return len(f.Decls) > 1 || plausibleDecl(f.Decls[0])
}

func isStmts(s string) bool {
	src := "package p\nfunc _() {\n" + s + "\n}\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil || isProse(fset, src, f) {
		return false
	}
	body := firstBody(f)
	ok := true
	exempt := map[ast.Node]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		if ts, isSwitch := n.(*ast.TypeSwitchStmt); isSwitch {
			exempt[ts.Assign] = true
		}
		if st, isStmt := n.(ast.Stmt); isStmt && !exempt[n] && !plausible(st) {
			ok = false
		}
		return ok
	})
	return ok && !loneBareDecl(body) && hasCodeNode(body)
}

func isProse(fset *token.FileSet, src string, f *ast.File) bool {
	return spacedDot(fset, src, f) || hasNonASCIIIdent(f)
}

func hasNonASCIIIdent(f *ast.File) bool {
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && !isASCII(id.Name) {
			found = true
		}
		return !found
	})
	return found
}

func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func spacedDot(fset *token.FileSet, src string, f *ast.File) bool {
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			from := fset.Position(sel.X.End()).Offset
			gap := src[from:fset.Position(sel.Sel.Pos()).Offset]
			if gap != "." && !strings.Contains(gap, "\n") {
				found = true
			}
		}
		return !found
	})
	return found
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

func plausible(st ast.Stmt) bool {
	switch x := st.(type) {
	case *ast.LabeledStmt:
		return false
	case *ast.ExprStmt:
		return isCallOrRecv(x.X)
	case *ast.IncDecStmt:
		return isAssignable(x.X)
	case *ast.AssignStmt:
		return allAssignable(x.Lhs)
	}
	return true
}

func allAssignable(lhs []ast.Expr) bool {
	for _, e := range lhs {
		if !isAssignable(e) {
			return false
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
