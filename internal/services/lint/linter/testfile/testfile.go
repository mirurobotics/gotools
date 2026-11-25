package testfile

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

const ruleName = "testfile"

func Check(
	fset *token.FileSet, filename string, file *ast.File, srcPrefix, testDir string,
) []analysis.Diagnostic {
	name := filepath.Base(filename)
	testStem, isTest := strings.CutSuffix(name, "_test.go")
	if !isTest {
		return nil
	}
	if analysis.HasPackageNolint(fset, file, ruleName) {
		return nil
	}
	srcDir := sourceDir(filepath.Dir(filename), srcPrefix, testDir)
	entries, err := os.ReadDir(srcDir)
	if hasSource(entries, testStem) {
		return nil
	}
	message := fmt.Sprintf("no source file in %s matches %s", srcDir, name)
	if err != nil {
		message = fmt.Sprintf("source package %s for %s is missing", srcDir, name)
	}
	line := fset.Position(file.Package).Line
	return []analysis.Diagnostic{{File: filename, Line: line, Message: message}}
}

func sourceDir(testFileDir, srcPrefix, testDir string) string {
	if testDir == "" {
		return testFileDir
	}
	root := filepath.ToSlash(filepath.Clean(testDir)) + "/"
	sub, inTestDir := strings.CutPrefix(filepath.ToSlash(testFileDir)+"/", root)
	if !inTestDir {
		return testFileDir
	}
	return path.Join(srcPrefix, sub)
}

func hasSource(entries []os.DirEntry, testStem string) bool {
	for _, entry := range entries {
		srcStem, isGo := strings.CutSuffix(entry.Name(), ".go")
		isSource := isGo && !strings.HasSuffix(srcStem, "_test")
		if isSource && isNamedAfter(testStem, srcStem) {
			return true
		}
	}
	return false
}

func isNamedAfter(testStem, srcStem string) bool {
	return testStem == srcStem || strings.HasPrefix(testStem, srcStem+"_")
}
