package analysis

import (
	"go/ast"
	"strings"
)

// PkgImportName returns the local name of file's import of path, or "".
func PkgImportName(file *ast.File, path string) string {
	for _, imp := range file.Imports {
		if ImportPath(imp) == path {
			return ImportName(imp)
		}
	}
	return ""
}

// ImportName returns the name a file uses for imp, guessing it when unaliased.
func ImportName(imp *ast.ImportSpec) string {
	if imp.Name != nil {
		return imp.Name.Name
	}
	elems := strings.Split(ImportPath(imp), "/")
	name := elems[len(elems)-1]
	if isVersionElem(name) && len(elems) > 1 {
		name = elems[len(elems)-2]
	}
	name, _, _ = strings.Cut(name, ".")
	return strings.TrimPrefix(name, "go-")
}

// ImportPath returns imp's path without quotes.
func ImportPath(imp *ast.ImportSpec) string {
	return strings.Trim(imp.Path.Value, "\"`")
}

func isVersionElem(elem string) bool {
	digits, ok := strings.CutPrefix(elem, "v")
	return ok && digits != "" && strings.Trim(digits, "0123456789") == ""
}
