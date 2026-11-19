package analysis

import (
	"go/ast"
	"go/token"
	"strings"
)

// DocGroups returns the doc comment groups of declarations, specs, and fields.
func DocGroups(f *ast.File) []*ast.CommentGroup {
	var docs []*ast.CommentGroup
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			docs = append(docs, x.Doc)
		case *ast.GenDecl:
			docs = append(docs, x.Doc)
		case *ast.TypeSpec:
			docs = append(docs, x.Doc)
		case *ast.ValueSpec:
			docs = append(docs, x.Doc)
		case *ast.ImportSpec:
			docs = append(docs, x.Doc)
		case *ast.Field:
			docs = append(docs, x.Doc)
		}
		return true
	})
	return docs
}

// LineComments returns the trailing line comment groups of specs and fields.
func LineComments(f *ast.File) []*ast.CommentGroup {
	var groups []*ast.CommentGroup
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
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

// ProseLines counts cg lines, ignoring directives and leading/trailing blanks.
func ProseLines(cg *ast.CommentGroup) int {
	var lines []string
	for _, c := range cg.List {
		if IsDirective(c.Text) {
			continue
		}
		text := strings.TrimSuffix(strings.TrimPrefix(c.Text, "/*"), "*/")
		lines = append(lines, strings.Split(text, "\n")...)
	}
	for len(lines) > 0 && isBlank(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && isBlank(lines[0]) {
		lines = lines[1:]
	}
	return len(lines)
}

func isBlank(line string) bool {
	return strings.TrimSpace(strings.TrimPrefix(line, "//")) == ""
}

// Comments returns every comment in f, flattened across groups, in source order.
func Comments(f *ast.File) []*ast.Comment {
	var out []*ast.Comment
	for _, cg := range f.Comments {
		out = append(out, cg.List...)
	}
	return out
}
