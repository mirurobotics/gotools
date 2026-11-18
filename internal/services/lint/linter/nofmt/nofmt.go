package nofmt

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

func banned() map[string]bool {
	return map[string]bool{
		"fmt.Print":   true,
		"fmt.Println": true,
		"fmt.Printf":  true,
		"log.Print":   true,
		"log.Println": true,
		"log.Printf":  true,
		"log.Fatal":   true,
		"log.Fatalf":  true,
		"log.Fatalln": true,
	}
}

func builtinBanned() map[string]bool {
	return map[string]bool{"print": true, "println": true}
}

// Check reports print and log calls outside main, tests, and cmd/.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	if f.Name.Name == "main" {
		return nil
	}
	if analysis.IsTestFile(filename) {
		return nil
	}
	if analysis.IsLintSource(filename) {
		return nil
	}
	if analysis.IsCmdSource(filename) {
		return nil
	}

	var diags []analysis.Diagnostic

	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		var name string
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			ident, ok := fn.X.(*ast.Ident)
			if !ok {
				return true
			}
			name = ident.Name + "." + fn.Sel.Name
		case *ast.Ident:
			name = fn.Name
		default:
			return true
		}

		if banned()[name] || builtinBanned()[name] {
			if hasNolint(fset, f, call.Pos()) {
				return true
			}
			pos := fset.Position(call.Pos())
			diags = append(diags, analysis.Diagnostic{
				File: filename,
				Line: pos.Line,
				Message: name +
					" should not be used outside" +
					" main/test; use structured logging",
			})
		}

		return true
	})

	return diags
}

func hasNolint(fset *token.FileSet, f *ast.File, pos token.Pos) bool {
	line := fset.Position(pos).Line
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if fset.Position(c.Pos()).Line != line {
				continue
			}
			if strings.Contains(c.Text, "nolint:nofmt") {
				return true
			}
		}
	}
	return false
}
