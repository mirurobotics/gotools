package todoissue

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"unicode"

	"github.com/mirurobotics/tools/go/internal/services/lint/linter/analysis"
)

const message = "TODO comment must include an issue link (an http or https URL) " +
	"in the same comment"

const linkPattern = `(?i)https?://[\w-]+(\.[\w-]+)+`

type line struct {
	num  int
	text string
}

// Check reports TODO comments with no issue link in the TODO or its continuation lines.
func Check(fset *token.FileSet, filename string, f *ast.File) []analysis.Diagnostic {
	link := regexp.MustCompile(linkPattern)
	var diags []analysis.Diagnostic
	for _, cg := range f.Comments {
		if analysis.HasNolint(cg, "todoissue") {
			continue
		}
		for _, n := range unlinkedTODOs(groupLines(fset, cg), link) {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    n,
				Message: message,
			})
		}
	}
	return diags
}

func groupLines(fset *token.FileSet, cg *ast.CommentGroup) []line {
	var lines []line
	for _, c := range cg.List {
		text, ok := checkedText(c.Text)
		if !ok {
			continue
		}
		start := fset.Position(c.Pos()).Line
		for i, text := range strings.Split(text, "\n") {
			lines = append(lines, line{start + i, stripMarkers(text)})
		}
	}
	return lines
}

func checkedText(text string) (string, bool) {
	if !analysis.IsDirective(text) {
		return text, true
	}
	note, ok := directiveNote(strings.TrimSpace(strings.TrimPrefix(text, "//")))
	return "//" + note, ok
}

func directiveNote(body string) (string, bool) {
	switch {
	case strings.HasPrefix(body, "nolint"):
		_, note, ok := strings.Cut(body, " //")
		return note, ok
	case strings.HasPrefix(body, "#nosec"):
		_, note, ok := strings.Cut(body, " -- ")
		return note, ok
	case strings.HasPrefix(body, "lint:ignore "),
		strings.HasPrefix(body, "lint:file-ignore "):
		fields := strings.Fields(body)
		return strings.Join(fields[min(2, len(fields)):], " "), len(fields) > 2
	}
	return "", false
}

func stripMarkers(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	text = strings.TrimSpace(text)
	text = strings.TrimSpace(strings.TrimPrefix(text, "*"))
	return strings.TrimSpace(strings.TrimPrefix(text, "- "))
}

func unlinkedTODOs(lines []line, link *regexp.Regexp) []int {
	var out []int
	open, linked := 0, false
	closeScope := func() {
		if open != 0 && !linked {
			out = append(out, open)
		}
		open = 0
	}
	for _, l := range lines {
		if l.text == "" || isTODO(l.text) {
			closeScope()
		}
		if isTODO(l.text) {
			open, linked = l.num, false
		}
		if open != 0 && link.MatchString(l.text) {
			linked = true
		}
	}
	closeScope()
	return out
}

func isTODO(text string) bool {
	rest, ok := strings.CutPrefix(text, "TODO")
	if !ok || rest == "" {
		return ok
	}
	r := []rune(rest)[0]
	return r == ':' || r == '(' || unicode.IsSpace(r)
}
