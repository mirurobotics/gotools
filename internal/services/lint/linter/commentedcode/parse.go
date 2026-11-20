package commentedcode

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"unicode/utf8"
)

func isCode(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	return isDecls(text) || isStmts(text)
}

func isDecls(text string) bool {
	src := "package p\n" + text + "\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil || len(file.Decls) == 0 || isProse(fset, src, file) {
		return false
	}
	return anyPlausibleDecl(file.Decls)
}

func isStmts(text string) bool {
	src := "package p\nfunc _() {\n" + text + "\n}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil || isProse(fset, src, file) {
		return false
	}
	body := firstBody(file)
	ok := true
	exempt := map[ast.Node]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		if typeSwitch, isSwitch := node.(*ast.TypeSwitchStmt); isSwitch {
			exempt[typeSwitch.Assign] = true
		}
		stmt, isStmt := node.(ast.Stmt)
		if isStmt && !exempt[node] && !plausible(stmt) {
			ok = false
		}
		return ok
	})
	return ok && hasCodeNode(body)
}

func isProse(fset *token.FileSet, src string, file *ast.File) bool {
	return spacedDot(fset, src, file) || hasNonASCIIIdent(file)
}

func spacedDot(fset *token.FileSet, src string, file *ast.File) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		if sel, ok := node.(*ast.SelectorExpr); ok {
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

func hasNonASCIIIdent(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && !isASCII(id.Name) {
			found = true
		}
		return !found
	})
	return found
}

func isASCII(text string) bool {
	for i := range len(text) {
		if text[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func firstBody(file *ast.File) *ast.BlockStmt {
	var body *ast.BlockStmt
	ast.Inspect(file, func(node ast.Node) bool {
		if decl, ok := node.(*ast.FuncDecl); ok && body == nil {
			body = decl.Body
		}
		return body == nil
	})
	return body
}

func plausible(stmt ast.Stmt) bool {
	switch stmt := stmt.(type) {
	case *ast.LabeledStmt:
		return false
	case *ast.ExprStmt:
		return isCallOrRecv(stmt.X)
	case *ast.IncDecStmt:
		return isAssignable(stmt.X)
	case *ast.AssignStmt:
		return allAssignable(stmt.Lhs)
	}
	return true
}

func isCallOrRecv(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case *ast.CallExpr:
		return true
	case *ast.UnaryExpr:
		return expr.Op == token.ARROW
	case *ast.ParenExpr:
		return isCallOrRecv(expr.X)
	}
	return false
}

func allAssignable(lhs []ast.Expr) bool {
	for _, expr := range lhs {
		if !isAssignable(expr) {
			return false
		}
	}
	return true
}

func isAssignable(expr ast.Expr) bool {
	switch expr := expr.(type) {
	case *ast.Ident, *ast.SelectorExpr, *ast.IndexExpr, *ast.StarExpr:
		return true
	case *ast.ParenExpr:
		return isAssignable(expr.X)
	}
	return false
}

func hasCodeNode(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		switch node.(type) {
		case *ast.CallExpr, *ast.AssignStmt, *ast.IncDecStmt,
			*ast.ReturnStmt, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt,
			*ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt,
			*ast.GoStmt, *ast.DeferStmt, *ast.SendStmt, *ast.ExprStmt:
			found = true
		}
		return !found
	})
	return found
}
