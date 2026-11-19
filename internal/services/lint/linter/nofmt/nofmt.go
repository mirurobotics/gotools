package nofmt

import (
	"go/ast"
	"go/token"

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

// Check reports print and log calls outside main, tests, cmd/, and linter source.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	if isExempt(filename, f) {
		return nil
	}

	var diags []analysis.Diagnostic

	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call)
		if !banned()[name] && !builtinBanned()[name] {
			return true
		}
		pos := fset.Position(call.Pos())
		if analysis.NolintOnLines(fset, f, "nofmt", pos.Line, pos.Line) {
			return true
		}
		diags = append(diags, analysis.Diagnostic{
			File: filename,
			Line: pos.Line,
			Message: name +
				" should not be used outside" +
				" main/test; use structured logging",
		})
		return true
	})

	return diags
}

func isExempt(filename string, f *ast.File) bool {
	return f.Name.Name == "main" ||
		analysis.IsTestFile(filename) ||
		analysis.IsLintSource(filename) ||
		analysis.IsCmdSource(filename)
}

func callName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		ident, ok := fn.X.(*ast.Ident)
		if !ok {
			return ""
		}
		return ident.Name + "." + fn.Sel.Name
	case *ast.Ident:
		return fn.Name
	}
	return ""
}
