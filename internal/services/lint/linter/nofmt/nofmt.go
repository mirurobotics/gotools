package nofmt

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const messageSuffix = " should not be used outside main; use structured logging or t.Log"

// Check reports print and log calls outside main, cmd/, linter source, and examples.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	if isExempt(filename, file) {
		return nil
	}

	var diags []analysis.Diagnostic

	ast.Inspect(file, func(node ast.Node) bool {
		decl, isFunc := node.(*ast.FuncDecl)
		if isFunc && analysis.IsExampleFunc(filename, decl) {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call)
		if !banned()[name] && !builtinBanned()[name] {
			return true
		}
		pos := fset.Position(call.Pos())
		if analysis.NolintOnLines(fset, file, "nofmt", pos.Line, pos.Line) {
			return true
		}
		diags = append(diags, analysis.Diagnostic{
			File:    filename,
			Line:    pos.Line,
			Message: name + messageSuffix,
		})
		return true
	})

	return diags
}

func isExempt(filename string, file *ast.File) bool {
	if file.Name.Name == "main" {
		return true
	}
	return analysis.IsLintSource(filename) || analysis.IsCmdSource(filename)
}

func callName(call *ast.CallExpr) string {
	switch expr := call.Fun.(type) {
	case *ast.SelectorExpr:
		ident, ok := expr.X.(*ast.Ident)
		if !ok {
			return ""
		}
		return ident.Name + "." + expr.Sel.Name
	case *ast.Ident:
		return expr.Name
	}
	return ""
}

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
