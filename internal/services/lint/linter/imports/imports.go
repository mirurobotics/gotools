package imports

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// ImportBlock represents one import declaration (single or grouped).
type ImportBlock struct {
	Imports   []Import
	Pos       token.Pos // position of "import" keyword
	End       token.Pos // position after closing paren (or end of single import)
	IsGrouped bool      // true if import (...) form
}

// ExtractBlocks walks a parsed file and returns all import blocks.
func ExtractBlocks(file *ast.File) []ImportBlock {
	var blocks []ImportBlock

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}

		blocks = append(blocks, newImportBlock(genDecl))
	}

	return blocks
}

func newImportBlock(decl *ast.GenDecl) ImportBlock {
	block := ImportBlock{
		Imports:   nil,
		Pos:       decl.Pos(),
		End:       decl.End(),
		IsGrouped: decl.Lparen.IsValid(),
	}
	for _, spec := range decl.Specs {
		if imp, ok := spec.(*ast.ImportSpec); ok {
			block.Imports = append(block.Imports, newImport(imp))
		}
	}
	return block
}

// Import represents a single import statement.
type Import struct {
	Path  string
	Name  string // alias, or "" if none; "." for dot imports
	Pos   token.Pos
	End   token.Pos
	Group ImportGroup
}

func newImport(spec *ast.ImportSpec) Import {
	path, _ := strconv.Unquote(spec.Path.Value)
	imp := Import{
		Path:  path,
		Name:  "",
		Pos:   spec.Pos(),
		End:   spec.End(),
		Group: classify(path),
	}
	if spec.Name != nil {
		imp.Name = spec.Name.Name
	}
	return imp
}

const internalPrefix = "github.com/mirurobotics/"

// ImportGroup represents the category of an import path.
type ImportGroup int

const (
	GroupStd      ImportGroup = 0
	GroupInternal ImportGroup = 1
	GroupExternal ImportGroup = 2
)

func classify(path string) ImportGroup {
	first := path
	if i := strings.IndexByte(path, '/'); i >= 0 {
		first = path[:i]
	}
	if !strings.Contains(first, ".") {
		return GroupStd
	}
	if strings.HasPrefix(path, internalPrefix) {
		return GroupInternal
	}
	return GroupExternal
}
