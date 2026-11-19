package errfmt

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports error strings that are capitalized or end with punctuation.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic

	ast.Inspect(f, func(n ast.Node) bool {
		lit, val, ok := errorStringArg(n)
		if !ok {
			return true
		}
		line := fset.Position(lit.Pos()).Line
		for _, msg := range checkErrorString(val) {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    line,
				Message: msg,
			})
		}
		return true
	})

	return diags
}

func errorStringArg(n ast.Node) (*ast.BasicLit, string, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok || !isErrorCall(call) || len(call.Args) == 0 {
		return nil, "", false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return nil, "", false
	}
	val, err := strconv.Unquote(lit.Value)
	if err != nil || len(val) == 0 {
		return nil, "", false
	}
	return lit, val, true
}

func checkErrorString(val string) []string {
	var msgs []string
	firstRune, _ := utf8.DecodeRuneInString(val)
	if unicode.IsUpper(firstRune) && !isAcronymStart(val) {
		msgs = append(msgs,
			fmt.Sprintf("error string %q should not be capitalized", val))
	}
	trimmed := strings.TrimRight(val, " ")
	if len(trimmed) > 0 {
		last := trimmed[len(trimmed)-1]
		if last == '.' || last == '!' {
			msgs = append(msgs, fmt.Sprintf(
				"error string %q should not end with punctuation", val))
		}
	}
	return msgs
}

func isErrorCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	name := ident.Name + "." + sel.Sel.Name
	return name == "errors.New" || name == "fmt.Errorf"
}

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

func firstWord(s string) string {
	i := strings.IndexFunc(s, func(r rune) bool {
		return r == ' ' || r == ':' || r == ',' || r == '('
	})
	if i < 0 {
		return s
	}
	return s[:i]
}
