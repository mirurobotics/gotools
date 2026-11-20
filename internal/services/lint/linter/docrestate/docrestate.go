package docrestate

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strings"
	"unicode"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports doc comments that only restate the name they document.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic
	for _, d := range analysis.NamedDocs(f) {
		if d.Doc == nil || d.Name == nil || d.Name.Name == "_" {
			continue
		}
		if analysis.HasNolint(d.Doc, "docrestate") {
			continue
		}
		if !restates(d.Name.Name, d.Doc) {
			continue
		}
		diags = append(diags, analysis.Diagnostic{
			File: filename,
			Line: fset.Position(d.Doc.Pos()).Line,
			Message: fmt.Sprintf(
				"doc comment for %s only restates its name; "+
					"say what it does or remove it", d.Name.Name),
		})
	}
	return diags
}

func restates(name string, cg *ast.CommentGroup) bool {
	text := strings.Join(analysis.ProseLines(cg), " ")
	if hasSymbols(text) {
		return false
	}
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !isWordRune(r)
	})
	if len(words) == 0 || words[0] != strings.ToLower(name) {
		return false
	}
	nameWords := append(splitName(name), strings.ToLower(name))
	for _, w := range words[1:] {
		if !isFiller(w) && !matchesAny(w, nameWords) {
			return false
		}
	}
	return true
}

func hasSymbols(text string) bool {
	if strings.ContainsAny(text, "\"`") || hasSingleQuoted(text) {
		return true
	}
	for _, tok := range strings.Fields(text) {
		bare := strings.Trim(tok, ".,;:!?")
		if bare != "" && !strings.ContainsFunc(bare, isWordRune) {
			return true
		}
	}
	return false
}

func hasSingleQuoted(text string) bool {
	r := []rune(text)
	for i, c := range r {
		if c != '\'' || (i > 0 && isWordRune(r[i-1])) {
			continue
		}
		if end := slices.Index(r[i+1:], '\''); end > 0 {
			return true
		}
	}
	return false
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func splitName(name string) []string {
	var words []string
	runes := []rune(name)
	start := 0
	for i := 1; i <= len(runes); i++ {
		if i < len(runes) && !isBoundary(runes, i) {
			continue
		}
		if w := strings.Trim(string(runes[start:i]), "_"); w != "" {
			words = append(words, strings.ToLower(w))
		}
		start = i
	}
	return words
}

func isBoundary(r []rune, i int) bool {
	prev, cur := r[i-1], r[i]
	switch {
	case cur == '_' || prev == '_':
		return true
	case unicode.IsLower(prev) && unicode.IsUpper(cur):
		return true
	case unicode.IsDigit(prev) != unicode.IsDigit(cur):
		return true
	}
	if !unicode.IsUpper(prev) || !unicode.IsUpper(cur) || i+1 >= len(r) {
		return false
	}
	return unicode.IsLower(r[i+1]) && !isPluralS(r, i+1)
}

func isPluralS(r []rune, i int) bool {
	return r[i] == 's' && (i+1 == len(r) || !unicode.IsLower(r[i+1]))
}

func isFiller(w string) bool {
	switch w {
	case "a", "an", "the", "this", "that", "it", "its", "is", "are", "be",
		"new", "returns", "creates", "constructs", "makes", "builds",
		"type", "struct", "interface", "func", "function", "method",
		"value", "instance", "object", "given", "for", "of":
		return true
	}
	return false
}

func matchesAny(w string, nameWords []string) bool {
	for _, n := range nameWords {
		if inflects(w, n) || inflects(n, w) {
			return true
		}
	}
	return false
}

func inflects(w, base string) bool {
	suffix, ok := strings.CutPrefix(w, base)
	if ok && takesShortSuffix(base, suffix) {
		return true
	}
	if len(base) < minStemLen {
		return false
	}
	if ok && (suffix == "ed" || suffix == "ing") {
		return true
	}
	if stem, ok := strings.CutSuffix(base, "e"); ok && w == stem+"ing" {
		return true
	}
	if stem, ok := strings.CutSuffix(base, "y"); ok {
		return w == stem+"ies" || w == stem+"ied"
	}
	return false
}

func takesShortSuffix(base, suffix string) bool {
	switch suffix {
	case "", "s":
		return true
	case "es":
		ends := []string{"s", "x", "z", "ch", "sh"}
		return slices.ContainsFunc(ends, func(end string) bool {
			return strings.HasSuffix(base, end)
		})
	case "d":
		return strings.HasSuffix(base, "e")
	}
	return false
}

const minStemLen = 4
