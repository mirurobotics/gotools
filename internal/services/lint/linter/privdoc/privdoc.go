package privdoc

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const message = "comment on identifier outside the exported API; " +
	"only the exported API may be documented"

// Check reports doc and trailing comments on identifiers outside the exported API.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	prose := proseComments(f)
	diags := make([]analysis.Diagnostic, 0, len(prose))
	for _, c := range prose {
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    fset.Position(c.Pos()).Line,
			Message: message,
		})
	}
	return diags
}

func proseComments(f *ast.File) []*ast.Comment {
	var groups []*ast.Comment
	add := func(cgs ...*ast.CommentGroup) {
		for _, cg := range cgs {
			if c := firstProse(cg); c != nil {
				groups = append(groups, c)
			}
		}
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !funcPublic(d) {
				add(d.Doc)
			}
		case *ast.GenDecl:
			add(genDeclGroups(d)...)
		}
	}
	slices.SortFunc(groups, func(a, b *ast.Comment) int {
		return int(a.Pos() - b.Pos())
	})
	return groups
}

func genDeclGroups(d *ast.GenDecl) []*ast.CommentGroup {
	if d.Tok == token.IMPORT {
		return nil
	}
	var groups []*ast.CommentGroup
	anyPublic := false
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			public := s.Name.IsExported()
			anyPublic = anyPublic || public
			if !public {
				groups = append(groups, s.Doc, s.Comment)
			}
			groups = append(groups, fieldGroups(s.Type, public)...)
		case *ast.ValueSpec:
			public := anyExported(s.Names)
			anyPublic = anyPublic || public
			if !public {
				groups = append(groups, s.Doc, s.Comment)
			}
			if s.Type != nil {
				groups = append(groups, fieldGroups(s.Type, public)...)
			}
		}
	}
	if !anyPublic {
		groups = append(groups, d.Doc)
	}
	return groups
}

func fieldGroups(expr ast.Expr, parentPublic bool) []*ast.CommentGroup {
	var list *ast.FieldList
	switch t := expr.(type) {
	case *ast.StructType:
		list = t.Fields
	case *ast.InterfaceType:
		list = t.Methods
	case *ast.StarExpr:
		return fieldGroups(t.X, parentPublic)
	case *ast.ArrayType:
		return fieldGroups(t.Elt, parentPublic)
	case *ast.ChanType:
		return fieldGroups(t.Value, parentPublic)
	case *ast.MapType:
		keys := fieldGroups(t.Key, parentPublic)
		return append(keys, fieldGroups(t.Value, parentPublic)...)
	default:
		return nil
	}
	var groups []*ast.CommentGroup
	for _, field := range list.List {
		public := parentPublic && fieldPublic(field)
		if !public {
			groups = append(groups, field.Doc, field.Comment)
		}
		groups = append(groups, fieldGroups(field.Type, public)...)
	}
	return groups
}

func fieldPublic(field *ast.Field) bool {
	if len(field.Names) > 0 {
		return anyExported(field.Names)
	}
	name := embeddedName(field.Type)
	return name == nil || name.IsExported()
}

func embeddedName(expr ast.Expr) *ast.Ident {
	switch t := expr.(type) {
	case *ast.Ident:
		return t
	case *ast.StarExpr:
		return embeddedName(t.X)
	case *ast.SelectorExpr:
		return t.Sel
	case *ast.IndexExpr:
		return embeddedName(t.X)
	case *ast.IndexListExpr:
		return embeddedName(t.X)
	}
	return nil
}

func funcPublic(d *ast.FuncDecl) bool {
	if !d.Name.IsExported() {
		return false
	}
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return true
	}
	name := embeddedName(d.Recv.List[0].Type)
	return name == nil || name.IsExported()
}

func anyExported(names []*ast.Ident) bool {
	for _, n := range names {
		if n.IsExported() {
			return true
		}
	}
	return false
}

func firstProse(cg *ast.CommentGroup) *ast.Comment {
	if cg == nil {
		return nil
	}
	for _, c := range cg.List {
		blank := strings.TrimSpace(strings.TrimPrefix(c.Text, "//")) == ""
		if !blank && !analysis.IsDirective(c.Text) {
			return c
		}
	}
	return nil
}
