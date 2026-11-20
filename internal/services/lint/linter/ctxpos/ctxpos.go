package ctxpos

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
)

// Check reports functions whose context.Context parameter is not first.
func Check(fset *token.FileSet, filename string, file *ast.File) []analysis.Diagnostic {
	ctxPkg := analysis.ContextPkgName(file)
	if ctxPkg == "" {
		return nil
	}
	testingPkg := analysis.TestingPkgName(file)

	var diags []analysis.Diagnostic

	ast.Inspect(file, func(node ast.Node) bool {
		params, pos, ok := funcParams(node)
		if !ok {
			return true
		}
		if ctxParamIndex(params, ctxPkg) <= 0 {
			return true
		}
		if !analysis.IsTestingType(params.List[0].Type, testingPkg) {
			position := fset.Position(pos)
			diags = append(diags, analysis.Diagnostic{
				File:    filename,
				Line:    position.Line,
				Message: "context.Context should be the first parameter",
			})
		}
		return true
	})

	return diags
}

func funcParams(node ast.Node) (*ast.FieldList, token.Pos, bool) {
	var funcType *ast.FuncType
	switch node := node.(type) {
	case *ast.FuncDecl:
		funcType = node.Type
	case *ast.FuncLit:
		funcType = node.Type
	default:
		return nil, token.NoPos, false
	}
	if funcType.Params == nil {
		return nil, token.NoPos, false
	}
	return funcType.Params, node.Pos(), true
}

func ctxParamIndex(params *ast.FieldList, ctxPkg string) int {
	paramIdx := 0
	for _, field := range params.List {
		if analysis.IsContextType(field.Type, ctxPkg) {
			return paramIdx
		}
		names := len(field.Names)
		if names == 0 {
			names = 1
		}
		paramIdx += names
	}
	return -1
}
