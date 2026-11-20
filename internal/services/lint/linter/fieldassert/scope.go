package fieldassert

import (
	"go/ast"
	"go/token"
	"go/types"
	"maps"
)

func (c *checker) closeUsed(open map[string]*fieldGroup, stmt ast.Stmt) {
	used := c.usedKeys(stmt)
	maps.DeleteFunc(open, func(_ string, group *fieldGroup) bool {
		for key := range group.deps {
			if used[key] {
				return true
			}
		}
		return false
	})
}

func (c *checker) usedKeys(stmt ast.Stmt) map[any]bool {
	call, _, args := c.testifyCall(stmt)
	if call == nil {
		return identSet(stmt)
	}
	used := make(map[any]bool)
	for _, arg := range args {
		ast.Inspect(arg, func(node ast.Node) bool {
			if !mayChange(node) {
				return true
			}
			maps.Copy(used, identSet(node))
			return false
		})
	}
	return used
}

func (c *checker) testifyCall(stmt ast.Stmt) (*ast.CallExpr, string, []ast.Expr) {
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return nil, "", nil
	}
	call, ok := exprStmt.X.(*ast.CallExpr)
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
	switch typed := id.Obj.Decl.(type) {
	case *ast.AssignStmt:
		return c.isNewValue(id, typed.Lhs, typed.Rhs)
	case *ast.ValueSpec:
		return c.isNewValue(id, identExprs(typed.Names), typed.Values)
	case *ast.Field:
		return c.isAssertionsType(typed.Type)
	}
	return false
}

func (c *checker) isNewValue(id *ast.Ident, lhs, rhs []ast.Expr) bool {
	i := 0
	for ; i < len(lhs); i++ {
		if name, ok := lhs[i].(*ast.Ident); ok && name.Obj == id.Obj {
			break
		}
	}
	return len(lhs) == len(rhs) && i < len(rhs) && c.isNewCall(rhs[i])
}

func (c *checker) isNewCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	return ok && c.isTestifySel(call.Fun, "New")
}

func (c *checker) isAssertionsType(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	return ok && c.isTestifySel(star.X, "Assertions")
}

func (c *checker) isTestifySel(expr ast.Expr, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
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

func identSet(node ast.Node) map[any]bool {
	keys := make(map[any]bool)
	ast.Inspect(node, func(child ast.Node) bool {
		switch child := child.(type) {
		case *ast.SelectorExpr:
			maps.Copy(keys, identSet(child.X))
			return false
		case *ast.Ident:
			keys[identKey(child)] = true
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

func mayChange(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.CallExpr:
		return !isPureUniverseCall(node.Fun)
	case *ast.FuncLit:
		return true
	case *ast.UnaryExpr:
		return node.Op == token.AND
	}
	return false
}

func isPureUniverseCall(fun ast.Expr) bool {
	id, ok := fun.(*ast.Ident)
	if !ok || id.Obj != nil {
		return false
	}
	switch obj := types.Universe.Lookup(id.Name).(type) {
	case *types.TypeName:
		return true
	case *types.Builtin:
		switch obj.Name() {
		case "len", "cap", "min", "max", "real", "imag", "complex":
			return true
		}
	}
	return false
}

func identExprs(ids []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, len(ids))
	for i, id := range ids {
		exprs[i] = id
	}
	return exprs
}

func rootIdent(expr ast.Expr) *ast.Ident {
	for {
		switch inner := expr.(type) {
		case *ast.Ident:
			return inner
		case *ast.SelectorExpr:
			expr = inner.X
		case *ast.IndexExpr:
			expr = inner.X
		case *ast.StarExpr:
			expr = inner.X
		case *ast.ParenExpr:
			expr = inner.X
		case *ast.CallExpr:
			expr = methodRecv(inner)
		default:
			return nil
		}
	}
}

func methodRecv(call *ast.CallExpr) ast.Expr {
	fun := call.Fun
	switch expr := fun.(type) {
	case *ast.IndexExpr:
		fun = expr.X
	case *ast.IndexListExpr:
		fun = expr.X
	}
	switch fun := fun.(type) {
	case *ast.Ident:
		return nil
	case *ast.SelectorExpr:
		if id, ok := fun.X.(*ast.Ident); ok && id.Obj == nil {
			return nil
		}
	}
	return fun
}

func resultFields(file *ast.File) map[*ast.Field]bool {
	fields := make(map[*ast.Field]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		if funcType, ok := node.(*ast.FuncType); ok && funcType.Results != nil {
			for _, field := range funcType.Results.List {
				fields[field] = true
			}
		}
		return true
	})
	return fields
}
