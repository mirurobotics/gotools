package linelen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"sync"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports lines wider than maxLineWidth, except URLs, imports, and strings.
func Check(
	fset *token.FileSet,
	filename string,
	file *ast.File,
	src []byte,
	maxLineWidth, tabWidth int,
) []analysis.Diagnostic {
	importRanges := importLineRanges(fset, file)
	lines := bytes.Split(src, []byte("\n"))

	var diags []analysis.Diagnostic
	for i, line := range lines {
		lineNum := i + 1
		width := analysis.VisualWidth(string(line), tabWidth)
		if width <= maxLineWidth {
			continue
		}
		if isExempt(string(line), lineNum, importRanges) {
			continue
		}
		diags = append(diags, analysis.Diagnostic{
			File: filename,
			Line: lineNum,
			Message: fmt.Sprintf(
				"line is %d columns wide, exceeds %d-column limit",
				width, maxLineWidth,
			),
		})
	}
	return diags
}

type lineRange struct {
	start, end int
}

func importLineRanges(fset *token.FileSet, file *ast.File) []lineRange {
	var ranges []lineRange
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		start := fset.Position(genDecl.Pos()).Line
		end := fset.Position(genDecl.End()).Line
		ranges = append(ranges, lineRange{start: start, end: end})
	}
	return ranges
}

func isExempt(line string, lineNum int, importRanges []lineRange) bool {
	if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
		return true
	}

	for _, span := range importRanges {
		if lineNum >= span.start && lineNum <= span.end {
			return true
		}
	}

	trimmed := strings.TrimSpace(line)

	if isStructTagLine(trimmed) {
		return true
	}

	if isParamlessFuncSig(trimmed) {
		return true
	}

	return isStringLiteralLine(trimmed)
}

func isStructTagLine(trimmed string) bool {
	idx := strings.Index(trimmed, "`")
	if idx < 0 {
		return false
	}
	end := strings.LastIndex(trimmed, "`")
	if end <= idx {
		return false
	}
	tag := trimmed[idx+1 : end]
	for _, key := range []string{"json", "gorm", "yaml", "xml", "db", "validate"} {
		if strings.Contains(tag, key+`:"`) {
			return true
		}
	}
	return false
}

func isParamlessFuncSig(trimmed string) bool {
	if !paramlessFuncSigRe().MatchString(trimmed) {
		return false
	}
	return strings.Count(trimmed, "{") != strings.Count(trimmed, "}")
}

func paramlessFuncSigRe() *regexp.Regexp {
	return sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?\w+\(\)`)
	})()
}

func isStringLiteralLine(trimmed string) bool {
	if len(trimmed) == 0 {
		return false
	}
	bare := strings.TrimRight(trimmed, ",")
	if isQuoted(bare) {
		return true
	}
	if _, rhs, ok := strings.Cut(trimmed, "= "); ok {
		rhs = strings.TrimSpace(rhs)
		rhs = strings.TrimRight(rhs, ",")
		if isQuoted(rhs) {
			return true
		}
	}
	return false
}

func isQuoted(text string) bool {
	return isWrappedIn(text, `"`) || isWrappedIn(text, "`")
}

func isWrappedIn(text, quote string) bool {
	return strings.HasPrefix(text, quote) && strings.HasSuffix(text, quote)
}
