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
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic

	ast.Inspect(file, func(node ast.Node) bool {
		lit, val, ok := errorStringArg(node)
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

func errorStringArg(node ast.Node) (*ast.BasicLit, string, bool) {
	call, ok := node.(*ast.CallExpr)
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

func isAcronymStart(text string) bool {
	word := firstWord(text)
	if len(word) < 2 {
		return false
	}
	for _, char := range word {
		if !unicode.IsUpper(char) && !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}

func firstWord(text string) string {
	i := strings.IndexFunc(text, func(char rune) bool {
		return char == ' ' || char == ':' || char == ',' || char == '('
	})
	if i < 0 {
		return text
	}
	return text[:i]
}
