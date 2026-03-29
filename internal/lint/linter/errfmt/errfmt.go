package errfmt

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unicode"

	"github.com/mirurobotics/gotools/internal/lint/linter/analysis"
)

// Check reports diagnostics for error strings passed to errors.New and
// fmt.Errorf that are capitalized or end with punctuation.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic

	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if !isErrorCall(call) {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		val, err := strconv.Unquote(lit.Value)
		if err != nil || len(val) == 0 {
			return true
		}

		pos := fset.Position(lit.Pos())

		// Check capitalization.
		firstRune := rune(val[0])
		isUpper := unicode.IsUpper(firstRune)
		if isUpper && !isAcronymStart(val) && !startsWithFormatVerb(val) {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    pos.Line,
				Message: fmt.Sprintf("error string %q should not be capitalized", val),
			})
		}

		// Check trailing punctuation.
		trimmed := strings.TrimRight(val, " ")
		if len(trimmed) > 0 {
			last := trimmed[len(trimmed)-1]
			if last == '.' || last == '!' {
				diags = append(diags, analysis.Diagnostic{
					File: filename,
					Line: pos.Line,
					Message: fmt.Sprintf(
						"error string %q should not end with punctuation",
						val,
					),
				})
			}
		}

		return true
	})

	return diags
}

// isErrorCall checks if a call expression is errors.New or fmt.Errorf.
func isErrorCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return (ident.Name == "errors" && sel.Sel.Name == "New") ||
		(ident.Name == "fmt" && sel.Sel.Name == "Errorf")
}

// isAcronymStart returns true if the string starts with an all-caps word
// (e.g., "HTTP request failed", "JSON parse error").
func isAcronymStart(s string) bool {
	word := firstWord(s)
	if len(word) < 2 {
		return false
	}
	for _, r := range word {
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// startsWithFormatVerb returns true if the string starts with a % format verb.
func startsWithFormatVerb(s string) bool { return len(s) > 0 && s[0] == '%' }

func firstWord(s string) string {
	i := strings.IndexFunc(s, func(r rune) bool {
		return r == ' ' || r == ':' || r == ',' || r == '('
	})
	if i < 0 {
		return s
	}
	return s[:i]
}
