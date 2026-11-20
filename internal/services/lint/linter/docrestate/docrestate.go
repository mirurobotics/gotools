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
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	var diags []analysis.Diagnostic
	for _, doc := range analysis.NamedDocs(file) {
		if doc.Doc == nil || doc.Name == nil || doc.Name.Name == "_" {
			continue
		}
		if analysis.HasNolint(doc.Doc, "docrestate") {
			continue
		}
		if !restates(doc.Name.Name, doc.Doc) {
			continue
		}
		diags = append(diags, analysis.Diagnostic{
			File: filename,
			Line: fset.Position(doc.Doc.Pos()).Line,
			Message: fmt.Sprintf(
				"doc comment for %s only restates its name; "+
					"say what it does or remove it", doc.Name.Name),
		})
	}
	return diags
}

func restates(name string, group *ast.CommentGroup) bool {
	text := strings.Join(analysis.ProseLines(group), " ")
	if hasSymbols(text) {
		return false
	}
	words := strings.FieldsFunc(strings.ToLower(text), func(char rune) bool {
		return !isWordRune(char)
	})
	if len(words) == 0 || words[0] != strings.ToLower(name) {
		return false
	}
	nameWords := append(splitName(name), strings.ToLower(name))
	for _, word := range words[1:] {
		if !isFiller(word) && !matchesAny(word, nameWords) {
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
	runes := []rune(text)
	for i, char := range runes {
		if char != '\'' || (i > 0 && isWordRune(runes[i-1])) {
			continue
		}
		if end := slices.Index(runes[i+1:], '\''); end > 0 {
			return true
		}
	}
	return false
}

func isWordRune(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_'
}

func splitName(name string) []string {
	var words []string
	runes := []rune(name)
	start := 0
	for i := 1; i <= len(runes); i++ {
		if i < len(runes) && !isBoundary(runes, i) {
			continue
		}
		if word := strings.Trim(string(runes[start:i]), "_"); word != "" {
			words = append(words, strings.ToLower(word))
		}
		start = i
	}
	return words
}

func isBoundary(runes []rune, i int) bool {
	prev, cur := runes[i-1], runes[i]
	switch {
	case cur == '_' || prev == '_':
		return true
	case unicode.IsLower(prev) && unicode.IsUpper(cur):
		return true
	case unicode.IsDigit(prev) != unicode.IsDigit(cur):
		return true
	}
	if !unicode.IsUpper(prev) || !unicode.IsUpper(cur) || i+1 >= len(runes) {
		return false
	}
	return unicode.IsLower(runes[i+1]) && !isPluralS(runes, i+1)
}

func isPluralS(runes []rune, i int) bool {
	return runes[i] == 's' && (i+1 == len(runes) || !unicode.IsLower(runes[i+1]))
}

func isFiller(word string) bool {
	switch word {
	case "a", "an", "the", "this", "that", "it", "its", "is", "are", "be",
		"new", "returns", "creates", "constructs", "makes", "builds",
		"type", "struct", "interface", "func", "function", "method",
		"value", "instance", "object", "given", "for", "of":
		return true
	}
	return false
}

func matchesAny(word string, nameWords []string) bool {
	for _, nameWord := range nameWords {
		if inflects(word, nameWord) || inflects(nameWord, word) {
			return true
		}
	}
	return false
}

const minStemLen = 4

func inflects(word, base string) bool {
	suffix, ok := strings.CutPrefix(word, base)
	if ok && takesShortSuffix(base, suffix) {
		return true
	}
	if len(base) < minStemLen {
		return false
	}
	if ok && (suffix == "ed" || suffix == "ing") {
		return true
	}
	if stem, ok := strings.CutSuffix(base, "e"); ok && word == stem+"ing" {
		return true
	}
	if stem, ok := strings.CutSuffix(base, "y"); ok {
		return word == stem+"ies" || word == stem+"ied"
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
