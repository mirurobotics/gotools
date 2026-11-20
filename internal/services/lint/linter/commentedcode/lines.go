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

func runs(s source, cg *ast.CommentGroup) [][]line {
	var out [][]line
	var cur []line
	for _, c := range cg.List {
		if analysis.IsDirective(c.Text) {
			out, cur = appendRun(out, cur), nil
			continue
		}
		cur = append(cur, lines(s, c)...)
	}
	return appendRun(out, cur)
}

func appendRun(out [][]line, run []line) [][]line {
	if len(run) == 0 {
		return out
	}
	return append(out, run)
}

func lines(s source, c *ast.Comment) []line {
	if body, ok := strings.CutPrefix(c.Text, "/*"); ok {
		return blockLines(s, c.Pos(), strings.TrimSuffix(body, "*/"))
	}
	body := strings.TrimPrefix(c.Text, "//")
	text := strings.TrimPrefix(body, " ")
	return []line{{pos: c.Pos(), text: text, indented: isIndented(body)}}
}

func blockLines(s source, opener token.Pos, body string) []line {
	texts := strings.Split(body, "\n")
	start := s.tf.PositionFor(opener, false)
	prefix := commonIndent(texts[1:])
	prefix = prefix[:min(len(prefix), len(s.before(opener)))]
	out := make([]line, len(texts))
	for i, t := range texts {
		text := strings.TrimPrefix(t, prefix)
		pos := opener
		if i > 0 {
			pos = s.tf.LineStart(start.Line + i)
		}
		out[i] = line{pos: pos, text: text, indented: isIndented(text)}
	}
	return out
}

func isIndented(s string) bool {
	return strings.HasPrefix(s, "\t") || strings.HasPrefix(s, "  ")
}

func commonIndent(texts []string) string {
	prefix, first := "", true
	for _, t := range texts {
		if strings.TrimSpace(t) == "" {
			continue
		}
		ws := t[:len(t)-len(strings.TrimLeft(t, " \t"))]
		if first {
			prefix, first = ws, false
		}
		for !strings.HasPrefix(ws, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

func joinText(run []line) string {
	texts := make([]string, len(run))
	for i, l := range run {
		texts[i] = l.text
	}
	return strings.Join(texts, "\n")
}
