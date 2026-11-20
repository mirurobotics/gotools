package analysis

import (
	"bytes"
	"go/ast"
	"strings"
	"unicode"
	"unicode/utf8"
)

// IsGeneratedFile reports whether a Code generated marker precedes the package line.
func IsGeneratedFile(src []byte) bool {
	for _, line := range bytes.Split(src, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if bytes.Contains(trimmed, []byte("// Code generated")) {
			return true
		}
		if bytes.HasPrefix(trimmed, []byte("package ")) {
			return false
		}
	}
	return false
}

// IsLintSource returns true if the file lives under the linter's own source tree.
func IsLintSource(filename string) bool {
	return strings.Contains(filename, "services/lint/")
}

// IsCmdSource returns true if the file lives under a cmd/ directory.
func IsCmdSource(filename string) bool { return strings.Contains(filename, "cmd/") }

// IsExampleFunc reports whether fn is a go test Example function in a test file.
func IsExampleFunc(filename string, fn *ast.FuncDecl) bool {
	if fn.Recv != nil || !IsTestFile(filename) {
		return false
	}
	suffix, ok := strings.CutPrefix(fn.Name.Name, "Example")
	if !ok || suffix == "" {
		return ok
	}
	r, _ := utf8.DecodeRuneInString(suffix)
	return !unicode.IsLower(r)
}

// IsTestFile returns true if the filename ends with _test.go.
func IsTestFile(filename string) bool { return strings.HasSuffix(filename, "_test.go") }
