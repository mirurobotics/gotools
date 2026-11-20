package fieldassert

import (
	"go/ast"
	"go/token"
	"maps"
)

func closeAssigned(open map[string]*fieldGroup, s ast.Stmt) {
	for _, name := range assignedNames(s) {
		maps.DeleteFunc(open, func(_ string, g *fieldGroup) bool {
			return g.root == name
		})
	}
}

func assignedNames(s ast.Stmt) []string {
	var names []string
	switch s := s.(type) {
	case *ast.AssignStmt:
		for _, lhs := range s.Lhs {
			names = append(names, rootName(lhs))
		}
	case *ast.IncDecStmt:
		names = append(names, rootName(s.X))
	case *ast.DeclStmt:
		ast.Inspect(s, func(n ast.Node) bool {
			if vs, ok := n.(*ast.ValueSpec); ok {
				names = append(names, identNames(vs.Names)...)
			}
			return true
		})
	}
	return names
}

func rootName(e ast.Expr) string {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.SelectorExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		case *ast.ParenExpr:
			e = x.X
		case *ast.CallExpr:
			e = methodRecv(x)
		default:
			return ""
		}
	}
}

func addNames(names map[string]bool, fl *ast.FieldList) {
	if fl == nil {
		return
	}
	for _, field := range fl.List {
		for _, n := range field.Names {
			names[n.Name] = true
		}
	}
}

func identNames(ids []*ast.Ident) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, id.Name)
	}
	return names
}

func (c *checker) assertObjects(body *ast.BlockStmt) []binding {
	var found []binding
	var blocks []*ast.BlockStmt
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case nil:
			blocks = popBlock(blocks)
			return true
		case *ast.FuncLit:
			return false
		case *ast.AssignStmt:
			if name := c.newObjectName(n); name != "" {
				end := blocks[len(blocks)-1].End()
				found = append(found, binding{name: name, from: n.End(), to: end})
			}
		}
		blocks = append(blocks, asBlock(n, blocks))
		return true
	})
	return found
}

func (c *checker) newObjectName(as *ast.AssignStmt) string {
	if len(as.Lhs) != 1 || len(as.Rhs) != 1 {
		return ""
	}
	id, ok := as.Lhs[0].(*ast.Ident)
	if !ok || !c.isNewCall(as.Rhs[0]) {
		return ""
	}
	return id.Name
}

func (c *checker) isNewCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "New" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && c.testify[pkg.Name]
}

func popBlock(blocks []*ast.BlockStmt) []*ast.BlockStmt {
	return blocks[:len(blocks)-1]
}

func asBlock(n ast.Node, blocks []*ast.BlockStmt) *ast.BlockStmt {
	if b, ok := n.(*ast.BlockStmt); ok {
		return b
	}
	return blocks[len(blocks)-1]
}

func (sc scope) isObject(id *ast.Ident) bool {
	for _, b := range sc.objects {
		if b.name == id.Name && b.from < id.Pos() && id.Pos() < b.to {
			return true
		}
	}
	return false
}

func localDefs(body *ast.BlockStmt) []string {
	var names []string
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.AssignStmt:
			names = append(names, definedNames(n.Tok, n.Lhs...)...)
		case *ast.RangeStmt:
			names = append(names, definedNames(n.Tok, n.Key, n.Value)...)
		case *ast.ValueSpec:
			names = append(names, identNames(n.Names)...)
		}
		return true
	})
	return names
}

func definedNames(tok token.Token, exprs ...ast.Expr) []string {
	var names []string
	for _, e := range exprs {
		if id, ok := e.(*ast.Ident); ok && tok == token.DEFINE {
			names = append(names, id.Name)
		}
	}
	return names
}

func methodRecv(call *ast.CallExpr) ast.Expr {
	if _, ok := call.Fun.(*ast.Ident); ok {
		return nil
	}
	return call.Fun
}
