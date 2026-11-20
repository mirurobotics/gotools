package fieldassert

import (
	"go/ast"
	"maps"
)

func closeAssigned(open map[string]*fieldGroup, s ast.Stmt) {
	for _, key := range assignedKeys(s) {
		maps.DeleteFunc(open, func(_ string, g *fieldGroup) bool { return g.deps[key] })
	}
}

func assignedKeys(s ast.Stmt) []any {
	var keys []any
	ast.Inspect(s, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range n.Lhs {
				keys = appendKey(keys, rootIdent(lhs))
			}
		case *ast.IncDecStmt:
			keys = appendKey(keys, rootIdent(n.X))
		}
		return true
	})
	return keys
}

func appendKey(keys []any, id *ast.Ident) []any {
	if id == nil {
		return keys
	}
	return append(keys, identKey(id))
}

func identKey(id *ast.Ident) any {
	if id.Obj != nil {
		return id.Obj
	}
	return id.Name
}

func (c *checker) isAssertObject(id *ast.Ident) bool {
	if id.Obj == nil {
		return false
	}
	switch d := id.Obj.Decl.(type) {
	case *ast.AssignStmt:
		return len(d.Rhs) == 1 && c.isNewCall(d.Rhs[0])
	case *ast.ValueSpec:
		return len(d.Values) == 1 && c.isNewCall(d.Values[0])
	}
	return false
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

func identSet(e ast.Expr) map[any]bool {
	keys := make(map[any]bool)
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			keys[identKey(id)] = true
		}
		return true
	})
	return keys
}
