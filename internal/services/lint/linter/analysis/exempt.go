package analysis

import (
	"bytes"
	"strings"
)

// IsTestFile returns true if the filename ends with _test.go.
func IsTestFile(filename string) bool { return strings.HasSuffix(filename, "_test.go") }

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

// IsTestHelper returns true if the file lives under a tests/ directory.
func IsTestHelper(filename string) bool { return strings.Contains(filename, "tests/") }
