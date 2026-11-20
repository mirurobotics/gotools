package fieldassert

import (
	"go/ast"
	"maps"
)

func closeAssigned(open map[string]*fieldGroup, s ast.Stmt) {
	for _, name := range assignedNames(s) {
		maps.DeleteFunc(open, func(_ string, g *fieldGroup) bool {
			return g.deps[name]
		})
	}
}

func assignedNames(s ast.Stmt) []string {
	var names []string
	ast.Inspect(s, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range n.Lhs {
				names = append(names, rootName(lhs))
			}
		case *ast.IncDecStmt:
			names = append(names, rootName(n.X))
		case *ast.ValueSpec:
			names = append(names, identNames(n.Names)...)
		}
		return true
	})
	return names
}

func identNames(ids []*ast.Ident) []string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		names = append(names, id.Name)
	}
	return names
}

func (c *checker) isAssertObject(id *ast.Ident) bool {
	if id.Obj == nil {
		return false
	}
	as, ok := id.Obj.Decl.(*ast.AssignStmt)
	return ok && len(as.Rhs) == 1 && c.isNewCall(as.Rhs[0])
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

func (c *checker) isParam(id *ast.Ident) bool {
	if id == nil || id.Obj == nil {
		return false
	}
	field, ok := id.Obj.Decl.(*ast.Field)
	return ok && !c.results[field]
}

func (c *checker) isPkg(id *ast.Ident) bool {
	return id.Obj == nil && c.imports[id.Name]
}

func rootName(e ast.Expr) string {
	if id := rootIdent(e); id != nil {
		return id.Name
	}
	return ""
}

func rootIdent(e ast.Expr) *ast.Ident {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return x
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
			return nil
		}
	}
}

func methodRecv(call *ast.CallExpr) ast.Expr {
	fun := call.Fun
	switch x := fun.(type) {
	case *ast.IndexExpr:
		fun = x.X
	case *ast.IndexListExpr:
		fun = x.X
	}
	switch x := fun.(type) {
	case *ast.Ident:
		return nil
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok && id.Obj == nil {
			return nil
		}
	}
	return fun
}

func resultFields(f *ast.File) map[*ast.Field]bool {
	fields := make(map[*ast.Field]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		if ft, ok := n.(*ast.FuncType); ok && ft.Results != nil {
			for _, field := range ft.Results.List {
				fields[field] = true
			}
		}
		return true
	})
	return fields
}

func identSet(e ast.Expr) map[string]bool {
	names := make(map[string]bool)
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			names[id.Name] = true
		}
		return true
	})
	return names
}
