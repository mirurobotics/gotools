package fieldassert

import (
	"go/ast"
	"maps"
)

func closeAssigned(open map[groupKey]*fieldGroup, s ast.Stmt) {
	for _, key := range assignedKeys(s) {
		maps.DeleteFunc(open, func(_ groupKey, g *fieldGroup) bool {
			return g.deps[key]
		})
	}
}

func rootKey(id *ast.Ident) any {
	if id == nil {
		return nil
	}
	return identKey(id)
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
		return c.isNewValue(id, d.Lhs, d.Rhs)
	case *ast.ValueSpec:
		return c.isNewValue(id, identExprs(d.Names), d.Values)
	case *ast.Field:
		return c.isAssertionsType(d.Type)
	}
	return false
}

func (c *checker) isNewValue(id *ast.Ident, lhs, rhs []ast.Expr) bool {
	i := 0
	for ; i < len(lhs); i++ {
		if l, ok := lhs[i].(*ast.Ident); ok && l.Obj == id.Obj {
			break
		}
	}
	return len(lhs) == len(rhs) && i < len(rhs) && c.isNewCall(rhs[i])
}

func (c *checker) isAssertionsType(e ast.Expr) bool {
	star, ok := e.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Assertions" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && c.testify[pkg.Name]
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

func identExprs(ids []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, len(ids))
	for i, id := range ids {
		exprs[i] = id
	}
	return exprs
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
