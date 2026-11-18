package analysis

import "go/ast"

// Comments returns every comment in f, flattened across comment groups,
// in source order.
func Comments(f *ast.File) []*ast.Comment {
	var out []*ast.Comment
	for _, cg := range f.Comments {
		out = append(out, cg.List...)
	}
	return out
}
