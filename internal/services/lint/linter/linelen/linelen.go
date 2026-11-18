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
	f *ast.File,
	src []byte,
	maxLineWidth, tabWidth int,
) []analysis.Diagnostic {
	importRanges := importLineRanges(fset, f)
	lines := bytes.Split(src, []byte("\n"))

	var diags []analysis.Diagnostic
	for i, line := range lines {
		lineNum := i + 1
		w := analysis.VisualWidth(string(line), tabWidth)
		if w <= maxLineWidth {
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
				w, maxLineWidth,
			),
		})
	}
	return diags
}

// lineRange represents a contiguous range of line numbers [start, end].
type lineRange struct {
	start, end int
}

// importLineRanges returns the line ranges of all import declarations in f.
func importLineRanges(fset *token.FileSet, f *ast.File) []lineRange {
	var ranges []lineRange
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		start := fset.Position(gd.Pos()).Line
		end := fset.Position(gd.End()).Line
		ranges = append(ranges, lineRange{start: start, end: end})
	}
	return ranges
}

// isExempt returns true if the line should be skipped from length checking.
func isExempt(line string, lineNum int, importRanges []lineRange) bool {
	if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
		return true
	}

	for _, r := range importRanges {
		if lineNum >= r.start && lineNum <= r.end {
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

// isStructTagLine reports whether the line contains a struct field tag.
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
	return strings.Contains(tag, `json:"`) ||
		strings.Contains(tag, `gorm:"`) ||
		strings.Contains(tag, `yaml:"`) ||
		strings.Contains(tag, `xml:"`) ||
		strings.Contains(tag, `db:"`) ||
		strings.Contains(tag, `validate:"`)
}

// paramlessFuncSigRe matches a func name followed by an empty parameter list.
func paramlessFuncSigRe() *regexp.Regexp {
	return sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?\w+\(\)`)
	})()
}

// isParamlessFuncSig reports whether the line opens a function with no parameters.
func isParamlessFuncSig(trimmed string) bool {
	if !paramlessFuncSigRe().MatchString(trimmed) {
		return false
	}
	return strings.Count(trimmed, "{") != strings.Count(trimmed, "}")
}

// isStringLiteralLine reports whether the line is mostly a string literal.
func isStringLiteralLine(trimmed string) bool {
	if len(trimmed) == 0 {
		return false
	}
	bare := strings.TrimRight(trimmed, ",")
	if (strings.HasPrefix(bare, `"`) && strings.HasSuffix(bare, `"`)) ||
		(strings.HasPrefix(bare, "`") && strings.HasSuffix(bare, "`")) {
		return true
	}
	if _, rhs, ok := strings.Cut(trimmed, "= "); ok {
		rhs = strings.TrimSpace(rhs)
		rhs = strings.TrimRight(rhs, ",")
		if (strings.HasPrefix(rhs, `"`) && strings.HasSuffix(rhs, `"`)) ||
			(strings.HasPrefix(rhs, "`") && strings.HasSuffix(rhs, "`")) {
			return true
		}
	}
	return false
}
