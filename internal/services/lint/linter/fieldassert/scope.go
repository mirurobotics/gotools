package fieldassert

import (
	"go/ast"
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
