package fieldassert

import (
	"go/ast"
	"go/token"
	"maps"
)

func (c *checker) closeUsed(open map[string]*fieldGroup, s ast.Stmt) {
	used := c.usedKeys(s)
	maps.DeleteFunc(open, func(_ string, g *fieldGroup) bool {
		for key := range g.deps {
			if used[key] {
				return true
			}
		}
		return false
	})
}

func (c *checker) usedKeys(s ast.Stmt) map[any]bool {
	call, _, args := c.testifyCall(s)
	if call == nil {
		return identSet(s)
	}
	used := make(map[any]bool)
	for _, arg := range args {
		ast.Inspect(arg, func(n ast.Node) bool {
			if !mayChange(n) {
				return true
			}
			maps.Copy(used, identSet(n))
			return false
		})
	}
	return used
}

func (c *checker) testifyCall(s ast.Stmt) (*ast.CallExpr, string, []ast.Expr) {
	es, ok := s.(*ast.ExprStmt)
	if !ok {
		return nil, "", nil
	}
	call, ok := es.X.(*ast.CallExpr)
	if !ok {
		return nil, "", nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, "", nil
	}
	recv, ok := sel.X.(*ast.Ident)
	switch {
	case !ok:
		return nil, "", nil
	case c.isAssertObject(recv):
		return call, sel.Sel.Name, call.Args
	case c.isTestifyPkg(recv) && len(call.Args) > 0:
		return call, sel.Sel.Name, call.Args[1:]
	}
	return nil, "", nil
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
	return ok && c.isTestifySel(star.X, "Assertions")
}

func (c *checker) isNewCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	return ok && c.isTestifySel(call.Fun, "New")
}

func (c *checker) isTestifySel(e ast.Expr, name string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && c.isTestifyPkg(pkg)
}

func (c *checker) isTestifyPkg(id *ast.Ident) bool {
	return id.Obj == nil && c.testify[id.Name]
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

func identSet(e ast.Node) map[any]bool {
	keys := make(map[any]bool)
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			keys[identKey(id)] = true
		}
		return true
	})
	return keys
}

func identKey(id *ast.Ident) any {
	if id.Obj != nil {
		return id.Obj
	}
	return id.Name
}

func mayChange(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.CallExpr, *ast.FuncLit:
		return true
	case *ast.UnaryExpr:
		return n.Op == token.AND
	}
	return false
}
