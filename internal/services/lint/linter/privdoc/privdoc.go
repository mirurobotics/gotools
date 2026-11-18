package privdoc

import (
	"go/ast"
	"go/token"
	"slices"

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

type allowed map[*ast.CommentGroup]bool

func proseComments(f *ast.File) []*ast.Comment {
	ok := allowedGroups(f)
	bodies := analysis.FuncBodies(f)
	groups := append(analysis.DocGroups(f), analysis.LineComments(f)...)
	var prose []*ast.Comment
	for _, cg := range groups {
		if cg == nil || ok[cg] || analysis.InAnyBody(cg.Pos(), bodies) {
			continue
		}
		if c := firstProse(cg); c != nil {
			prose = append(prose, c)
		}
	}
	slices.SortFunc(prose, func(a, b *ast.Comment) int {
		return int(a.Pos() - b.Pos())
	})
	return prose
}

func allowedGroups(f *ast.File) allowed {
	ok := allowed{}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			public := funcPublic(d)
			ok.add(public, d.Doc)
			ok.walk(d.Type, public)
		case *ast.GenDecl:
			ok.genDecl(d)
		}
	}
	return ok
}

func (ok allowed) genDecl(d *ast.GenDecl) {
	anyPublic := d.Tok == token.IMPORT
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.ImportSpec:
			ok.add(true, s.Doc, s.Comment)
		case *ast.TypeSpec:
			public := s.Name.IsExported()
			anyPublic = anyPublic || public
			ok.add(public, s.Doc, s.Comment)
			if s.TypeParams != nil {
				ok.walk(s.TypeParams, public)
			}
			ok.walk(s.Type, public)
		case *ast.ValueSpec:
			public := anyExported(s.Names)
			anyPublic = anyPublic || public
			ok.add(public, s.Doc, s.Comment)
			if s.Type != nil {
				ok.walk(s.Type, public)
			}
			for _, v := range s.Values {
				ok.walk(v, public)
			}
		}
	}
	ok.add(anyPublic, d.Doc)
}

func (ok allowed) walk(n ast.Node, public bool) {
	ast.Inspect(n, func(x ast.Node) bool {
		switch t := x.(type) {
		case *ast.StructType:
			ok.fields(t.Fields, public)
			return false
		case *ast.InterfaceType:
			ok.fields(t.Methods, public)
			return false
		}
		return true
	})
}

func (ok allowed) fields(list *ast.FieldList, public bool) {
	for _, field := range list.List {
		p := public && fieldPublic(field)
		ok.add(p, field.Doc, field.Comment)
		ok.walk(field.Type, p)
	}
}

func (ok allowed) add(public bool, groups ...*ast.CommentGroup) {
	if !public {
		return
	}
	for _, cg := range groups {
		ok[cg] = true
	}
}

func fieldPublic(field *ast.Field) bool {
	if len(field.Names) > 0 {
		return anyExported(field.Names)
	}
	name := analysis.BaseIdent(field.Type)
	return name == nil || name.IsExported()
}

func funcPublic(d *ast.FuncDecl) bool {
	if !d.Name.IsExported() {
		return false
	}
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return true
	}
	name := analysis.BaseIdent(d.Recv.List[0].Type)
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
	if analysis.HasNolint(cg, "privdoc") {
		return nil
	}
	for _, c := range cg.List {
		if analysis.ProseLines(&ast.CommentGroup{List: []*ast.Comment{c}}) > 0 {
			return c
		}
	}
	return nil
}
