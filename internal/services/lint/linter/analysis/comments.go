package analysis

import (
	"go/ast"
	"go/token"
	"strings"
)

// DocGroups returns the doc comment groups outside function bodies.
func DocGroups(file *ast.File) []*ast.CommentGroup {
	named := NamedDocs(file)
	docs := make([]*ast.CommentGroup, len(named))
	for i, doc := range named {
		docs[i] = doc.Doc
	}
	return docs
}

// NamedDoc pairs a doc comment group with the single name it documents, if any.
type NamedDoc struct {
	Name *ast.Ident
	Doc  *ast.CommentGroup
}

// NamedDocs pairs doc groups outside function bodies with their single name, if any.
func NamedDocs(file *ast.File) []NamedDoc {
	var docs []NamedDoc
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BlockStmt:
			return false
		case *ast.FuncDecl:
			docs = append(docs, NamedDoc{node.Name, node.Doc})
		case *ast.GenDecl:
			docs = append(docs, NamedDoc{genDeclName(node), node.Doc})
		case *ast.TypeSpec:
			docs = append(docs, NamedDoc{node.Name, node.Doc})
		case *ast.ValueSpec:
			docs = append(docs, NamedDoc{soleName(node.Names), node.Doc})
		case *ast.ImportSpec:
			docs = append(docs, NamedDoc{nil, node.Doc})
		case *ast.Field:
			docs = append(docs, NamedDoc{soleName(node.Names), node.Doc})
		}
		return true
	})
	return docs
}

func genDeclName(decl *ast.GenDecl) *ast.Ident {
	if decl.Lparen != token.NoPos || len(decl.Specs) != 1 {
		return nil
	}
	switch spec := decl.Specs[0].(type) {
	case *ast.TypeSpec:
		return spec.Name
	case *ast.ValueSpec:
		return soleName(spec.Names)
	}
	return nil
}

func soleName(names []*ast.Ident) *ast.Ident {
	if len(names) != 1 {
		return nil
	}
	return names[0]
}

// LineComments returns trailing comments of specs and fields outside function bodies.
func LineComments(file *ast.File) []*ast.CommentGroup {
	var groups []*ast.CommentGroup
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.BlockStmt:
			return false
		case *ast.TypeSpec:
			groups = append(groups, node.Comment)
		case *ast.ValueSpec:
			groups = append(groups, node.Comment)
		case *ast.ImportSpec:
			groups = append(groups, node.Comment)
		case *ast.Field:
			groups = append(groups, node.Comment)
		}
		return true
	})
	return groups
}

// FuncBodies returns the bodies of all function declarations and literals.
func FuncBodies(file *ast.File) []*ast.BlockStmt {
	var bodies []*ast.BlockStmt
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl:
			if node.Body != nil {
				bodies = append(bodies, node.Body)
			}
		case *ast.FuncLit:
			bodies = append(bodies, node.Body)
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

// ProseLineCount counts group lines, ignoring directives and leading/trailing blanks.
func ProseLineCount(group *ast.CommentGroup) int {
	lines := ProseLines(group)
	for len(lines) > 0 && isBlank(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && isBlank(lines[0]) {
		lines = lines[1:]
	}
	return len(lines)
}

// ProseLines returns the prose of group's non-directive comments, without markers.
func ProseLines(group *ast.CommentGroup) []string {
	var lines []string
	for _, comment := range group.List {
		if IsDirective(comment.Text) {
			continue
		}
		if text, ok := strings.CutPrefix(comment.Text, "//"); ok {
			lines = append(lines, text)
			continue
		}
		lines = append(lines, blockLines(comment.Text)...)
	}
	return lines
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

// Comments returns every comment in file, flattened across groups, in source order.
func Comments(file *ast.File) []*ast.Comment {
	var out []*ast.Comment
	for _, group := range file.Comments {
		out = append(out, group.List...)
	}
	return out
}
