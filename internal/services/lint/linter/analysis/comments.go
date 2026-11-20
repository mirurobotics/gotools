package analysis

import (
	"go/ast"
	"go/token"
	"strings"
)

// NamedDoc pairs a doc comment group with the single name it documents, if any.
type NamedDoc struct {
	Name *ast.Ident
	Doc  *ast.CommentGroup
}

// DocGroups returns the doc comment groups outside function bodies.
func DocGroups(f *ast.File) []*ast.CommentGroup {
	named := NamedDocs(f)
	docs := make([]*ast.CommentGroup, len(named))
	for i, d := range named {
		docs[i] = d.Doc
	}
	return docs
}

// NamedDocs pairs doc groups outside function bodies with their single name, if any.
func NamedDocs(f *ast.File) []NamedDoc {
	var docs []NamedDoc
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BlockStmt:
			return false
		case *ast.FuncDecl:
			docs = append(docs, NamedDoc{x.Name, x.Doc})
		case *ast.GenDecl:
			docs = append(docs, NamedDoc{genDeclName(x), x.Doc})
		case *ast.TypeSpec:
			docs = append(docs, NamedDoc{x.Name, x.Doc})
		case *ast.ValueSpec:
			docs = append(docs, NamedDoc{soleName(x.Names), x.Doc})
		case *ast.ImportSpec:
			docs = append(docs, NamedDoc{nil, x.Doc})
		case *ast.Field:
			docs = append(docs, NamedDoc{soleName(x.Names), x.Doc})
		}
		return true
	})
	return docs
}

// LineComments returns trailing comments of specs and fields outside function bodies.
func LineComments(f *ast.File) []*ast.CommentGroup {
	var groups []*ast.CommentGroup
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BlockStmt:
			return false
		case *ast.TypeSpec:
			groups = append(groups, x.Comment)
		case *ast.ValueSpec:
			groups = append(groups, x.Comment)
		case *ast.ImportSpec:
			groups = append(groups, x.Comment)
		case *ast.Field:
			groups = append(groups, x.Comment)
		}
		return true
	})
	return groups
}

// FuncBodies returns the bodies of all function declarations and literals.
func FuncBodies(f *ast.File) []*ast.BlockStmt {
	var bodies []*ast.BlockStmt
	ast.Inspect(f, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body != nil {
				bodies = append(bodies, fn.Body)
			}
		case *ast.FuncLit:
			bodies = append(bodies, fn.Body)
		}
		return true
	})
	return bodies
}

// InAnyBody reports whether pos lies strictly inside one of bodies.
func InAnyBody(pos token.Pos, bodies []*ast.BlockStmt) bool {
	for _, b := range bodies {
		if pos > b.Lbrace && pos < b.Rbrace {
			return true
		}
	}
	return false
}

// ProseLineCount counts cg lines, ignoring directives and leading/trailing blanks.
func ProseLineCount(cg *ast.CommentGroup) int {
	lines := ProseLines(cg)
	for len(lines) > 0 && isBlank(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && isBlank(lines[0]) {
		lines = lines[1:]
	}
	return len(lines)
}

// ProseLines returns the prose lines of cg's non-directive comments, without markers.
func ProseLines(cg *ast.CommentGroup) []string {
	var lines []string
	for _, c := range cg.List {
		if IsDirective(c.Text) {
			continue
		}
		if text, ok := strings.CutPrefix(c.Text, "//"); ok {
			lines = append(lines, text)
			continue
		}
		lines = append(lines, blockLines(c.Text)...)
	}
	return lines
}

// Comments returns every comment in f, flattened across groups, in source order.
func Comments(f *ast.File) []*ast.Comment {
	var out []*ast.Comment
	for _, cg := range f.Comments {
		out = append(out, cg.List...)
	}
	return out
}

func genDeclName(d *ast.GenDecl) *ast.Ident {
	if d.Lparen != token.NoPos || len(d.Specs) != 1 {
		return nil
	}
	switch s := d.Specs[0].(type) {
	case *ast.TypeSpec:
		return s.Name
	case *ast.ValueSpec:
		return soleName(s.Names)
	}
	return nil
}

func soleName(names []*ast.Ident) *ast.Ident {
	if len(names) != 1 {
		return nil
	}
	return names[0]
}

func blockLines(text string) []string {
	text = strings.TrimSuffix(strings.TrimPrefix(text, "/*"), "*/")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "*"); ok {
			lines[i] = rest
		}
	}
	return lines
}

func isBlank(line string) bool { return strings.TrimSpace(line) == "" }
