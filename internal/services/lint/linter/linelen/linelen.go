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

// Check reports a diagnostic for each line in src that exceeds maxLineWidth
// visual columns (tabs expanded to tabWidth). Lines containing URLs, import
// paths, or string literals are exempt.
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
	// URL exemption.
	if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
		return true
	}

	// Import path exemption.
	for _, r := range importRanges {
		if lineNum >= r.start && lineNum <= r.end {
			return true
		}
	}

	trimmed := strings.TrimSpace(line)

	// Struct tag exemption: Go does not allow struct tags to span
	// multiple lines, so lines containing a struct tag are exempt.
	if isStructTagLine(trimmed) {
		return true
	}

	// Parameterless function signature exemption: func names and return
	// types are not splittable, so exempt signatures with no parameters.
	if isParamlessFuncSig(trimmed) {
		return true
	}

	// String literal exemption: the trimmed line is predominantly a string.
	return isStringLiteralLine(trimmed)
}

// isStructTagLine returns true if the trimmed line contains a Go struct
// field tag (a backtick-delimited string with key:"value" pairs).
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

// paramlessFuncSigRe returns a compiled regexp that matches a function
// signature whose own parameter list is empty. It skips the optional
// receiver and matches the func name followed by "()" — avoiding false
// positives from func() callback types that appear in parameter lists.
func paramlessFuncSigRe() *regexp.Regexp {
	return sync.OnceValue(func() *regexp.Regexp {
		return regexp.MustCompile(`^func\s+(?:\([^)]*\)\s+)?\w+\(\)`)
	})()
}

// isParamlessFuncSig returns true if the trimmed line is a function
// signature with no parameters (e.g. "func (r *Repo) LongName() error {").
// Lines containing a complete function body (both { and }) are not exempt.
func isParamlessFuncSig(trimmed string) bool {
	if !paramlessFuncSigRe().MatchString(trimmed) {
		return false
	}
	// If braces are balanced the line contains a complete single-line
	// function body — the body can be moved to a new line.
	// We count rather than using first/last index so that type braces
	// (e.g. struct{}, interface{}) don't cause false positives.
	return strings.Count(trimmed, "{") != strings.Count(trimmed, "}")
}

// isStringLiteralLine returns true if the trimmed line looks like it's
// predominantly a string literal (starts/ends with quote or backtick,
// or is an assignment to a string).
func isStringLiteralLine(trimmed string) bool {
	if len(trimmed) == 0 {
		return false
	}
	// Entire line is a quoted string or backtick string (possibly with trailing comma).
	bare := strings.TrimRight(trimmed, ",")
	if (strings.HasPrefix(bare, `"`) && strings.HasSuffix(bare, `"`)) ||
		(strings.HasPrefix(bare, "`") && strings.HasSuffix(bare, "`")) {
		return true
	}
	// Assignment: something = "..." or something = `...`
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
