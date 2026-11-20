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
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	prose := proseComments(file)
	diags := make([]analysis.Diagnostic, 0, len(prose))
	for _, comment := range prose {
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    fset.Position(comment.Pos()).Line,
			Message: message,
		})
	}
	return diags
}

func proseComments(file *ast.File) []*ast.Comment {
	ok := allowedGroups(file)
	groups := append(analysis.DocGroups(file), analysis.LineComments(file)...)
	var prose []*ast.Comment
	for _, group := range groups {
		if group == nil || ok[group] {
			continue
		}
		if comment := firstProse(group); comment != nil {
			prose = append(prose, comment)
		}
	}
	slices.SortFunc(prose, func(a, b *ast.Comment) int {
		return int(a.Pos() - b.Pos())
	})
	return prose
}

func allowedGroups(file *ast.File) allowed {
	ok := allowed{}
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			public := funcPublic(decl)
			ok.add(public, decl.Doc)
			ok.walk(decl.Type, public)
		case *ast.GenDecl:
			ok.genDecl(decl)
		}
	}
	return ok
}

func funcPublic(decl *ast.FuncDecl) bool {
	if !decl.Name.IsExported() {
		return false
	}
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return true
	}
	name := analysis.BaseIdent(decl.Recv.List[0].Type)
	return name == nil || name.IsExported()
}

func firstProse(group *ast.CommentGroup) *ast.Comment {
	if analysis.HasNolint(group, "privdoc") {
		return nil
	}
	for _, comment := range group.List {
		single := &ast.CommentGroup{List: []*ast.Comment{comment}}
		if analysis.ProseLineCount(single) > 0 {
			return comment
		}
	}
	return nil
}

type allowed map[*ast.CommentGroup]bool

func (ok allowed) genDecl(decl *ast.GenDecl) {
	anyPublic := decl.Tok == token.IMPORT
	for _, spec := range decl.Specs {
		anyPublic = ok.spec(spec) || anyPublic
	}
	ok.add(anyPublic, decl.Doc)
}

func (ok allowed) spec(spec ast.Spec) bool {
	switch spec := spec.(type) {
	case *ast.ImportSpec:
		ok.add(true, spec.Doc, spec.Comment)
	case *ast.TypeSpec:
		public := spec.Name.IsExported()
		ok.add(public, spec.Doc, spec.Comment)
		if spec.TypeParams != nil {
			ok.walk(spec.TypeParams, public)
		}
		ok.walk(spec.Type, public)
		return public
	case *ast.ValueSpec:
		public := anyExported(spec.Names)
		ok.add(public, spec.Doc, spec.Comment)
		if spec.Type != nil {
			ok.walk(spec.Type, public)
		}
		for _, expr := range spec.Values {
			ok.walk(expr, public)
		}
		return public
	}
	return false
}

func (ok allowed) walk(node ast.Node, public bool) {
	ast.Inspect(node, func(child ast.Node) bool {
		switch child := child.(type) {
		case *ast.StructType:
			ok.fields(child.Fields, public)
			return false
		case *ast.InterfaceType:
			ok.fields(child.Methods, public)
			return false
		}
		return true
	})
}

func (ok allowed) fields(list *ast.FieldList, public bool) {
	for _, field := range list.List {
		isPublic := public && fieldPublic(field)
		ok.add(isPublic, field.Doc, field.Comment)
		ok.walk(field.Type, isPublic)
	}
}

func (ok allowed) add(public bool, groups ...*ast.CommentGroup) {
	if !public {
		return
	}
	for _, group := range groups {
		ok[group] = true
	}
}

func fieldPublic(field *ast.Field) bool {
	if len(field.Names) > 0 {
		return anyExported(field.Names)
	}
	name := analysis.BaseIdent(field.Type)
	return name == nil || name.IsExported()
}

func anyExported(names []*ast.Ident) bool {
	for _, ident := range names {
		if ident.IsExported() {
			return true
		}
	}
	return false
}
