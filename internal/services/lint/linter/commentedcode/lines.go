package commentedcode

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

type source struct {
	tf  *token.File
	src []byte
}

func (s source) before(pos token.Pos) string {
	off := s.tf.Offset(pos)
	start := analysis.LineStartOffset(s.src, off)
	return strings.TrimPrefix(string(s.src[start:off]), "\uFEFF")
}

type line struct {
	pos      token.Pos
	text     string
	indented bool
}

func runs(code source, group *ast.CommentGroup) [][]line {
	var out [][]line
	var cur []line
	for _, comment := range group.List {
		if analysis.IsDirective(comment.Text) {
			out, cur = appendRun(out, cur), nil
			continue
		}
		cur = append(cur, lines(code, comment)...)
	}
	return appendRun(out, cur)
}

func appendRun(out [][]line, run []line) [][]line {
	if len(run) == 0 {
		return out
	}
	return append(out, run)
}

func lines(code source, comment *ast.Comment) []line {
	if body, ok := strings.CutPrefix(comment.Text, "/*"); ok {
		return blockLines(code, comment.Pos(), strings.TrimSuffix(body, "*/"))
	}
	body := strings.TrimPrefix(comment.Text, "//")
	text := strings.TrimPrefix(body, " ")
	return []line{{pos: comment.Pos(), text: text, indented: isIndented(body)}}
}

func blockLines(code source, opener token.Pos, body string) []line {
	texts := strings.Split(body, "\n")
	start := code.tf.PositionFor(opener, false)
	prefix := commonIndent(texts[1:])
	prefix = prefix[:min(len(prefix), len(code.before(opener)))]
	out := make([]line, len(texts))
	for i, raw := range texts {
		text := strings.TrimPrefix(raw, prefix)
		pos := opener
		if i > 0 {
			pos = code.tf.LineStart(start.Line + i)
		}
		out[i] = line{pos: pos, text: text, indented: isIndented(text)}
	}
	return out
}

func commonIndent(texts []string) string {
	prefix, first := "", true
	for _, text := range texts {
		if strings.TrimSpace(text) == "" {
			continue
		}
		indent := text[:len(text)-len(strings.TrimLeft(text, " \t"))]
		if first {
			prefix, first = indent, false
		}
		for !strings.HasPrefix(indent, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func isIndented(text string) bool {
	return strings.HasPrefix(text, "\t") || strings.HasPrefix(text, "  ")
}

func joinText(run []line) string {
	texts := make([]string, len(run))
	for i, entry := range run {
		texts[i] = entry.text
	}
	return strings.Join(texts, "\n")
}
