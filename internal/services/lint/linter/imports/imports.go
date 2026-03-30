package imports

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// ImportGroup represents the category of an import path.
type ImportGroup int

const (
	GroupStd      ImportGroup = 0
	GroupInternal ImportGroup = 1
	GroupExternal ImportGroup = 2
)

const internalPrefix = "github.com/mirurobotics/"

// Import represents a single import statement.
type Import struct {
	Path  string
	Name  string // alias, or "" if none; "." for dot imports
	Pos   token.Pos
	End   token.Pos
	Group ImportGroup
}

// ImportBlock represents one import declaration (single or grouped).
type ImportBlock struct {
	Imports   []Import
	Pos       token.Pos // position of "import" keyword
	End       token.Pos // position after closing paren (or end of single import)
	IsGrouped bool      // true if import (...) form
}

// ExtractBlocks walks a parsed file and returns all import blocks.
func ExtractBlocks(f *ast.File) []ImportBlock {
	var blocks []ImportBlock

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}

		block := ImportBlock{
			Imports:   nil,
			Pos:       gd.Pos(),
			End:       gd.End(),
			IsGrouped: gd.Lparen.IsValid(),
		}

		for _, spec := range gd.Specs {
			is, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			path, _ := strconv.Unquote(is.Path.Value)
			imp := Import{
				Path:  path,
				Name:  "",
				Pos:   is.Pos(),
				End:   is.End(),
				Group: classify(path),
			}
			if is.Name != nil {
				imp.Name = is.Name.Name
			}
			block.Imports = append(block.Imports, imp)
		}

		blocks = append(blocks, block)
	}

	return blocks
}

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
