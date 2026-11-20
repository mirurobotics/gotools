package todoissue

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"unicode"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const message = "TODO comment must include an issue link (an http or https URL) " +
	"in the same comment"

const linkPattern = `(?i)https?://[\w-]+(\.[\w-]+)+`

// Check reports TODO comments with no issue link in the TODO or its continuation lines.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	link := regexp.MustCompile(linkPattern)
	var diags []analysis.Diagnostic
	for _, group := range file.Comments {
		if analysis.HasNolint(group, "todoissue") {
			continue
		}
		for _, lineNum := range unlinkedTODOs(groupLines(fset, group), link) {
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    lineNum,
				Message: message,
			})
		}
	}
	return diags
}

type line struct {
	num  int
	text string
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
	for _, entry := range lines {
		if entry.text == "" || isTODO(entry.text) {
			closeScope()
		}
		if isTODO(entry.text) {
			open, linked = entry.num, false
		}
		if open != 0 && link.MatchString(entry.text) {
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
	char := []rune(rest)[0]
	return char == ':' || char == '(' || unicode.IsSpace(char)
}

func groupLines(fset *token.FileSet, group *ast.CommentGroup) []line {
	var lines []line
	for _, comment := range group.List {
		text, ok := checkedText(comment.Text)
		if !ok {
			continue
		}
		start := fset.Position(comment.Pos()).Line
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
