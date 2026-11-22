package linter

import (
	"go/ast"
	"go/token"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/collapse"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcinline"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcorder"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcsig"
)

func applyCandidateFixers(
	path string,
	src []byte,
	fset *token.FileSet,
	file *ast.File,
	cfg Config,
) ([]byte, *token.FileSet, *ast.File, error) {
	fixers := []struct {
		rule Rule
		find func(*token.FileSet, *ast.File, []byte, int, int) []analysis.Candidate
	}{
		{RuleCollapse, collapse.FindCandidates},
		{RuleFuncSig, funcsig.FindCandidates},
		{RuleFuncInline, funcinline.FindCandidates},
	}
	for _, fixer := range fixers {
		if !cfg.runRule(fixer.rule) {
			continue
		}
		candidates := fixer.find(fset, file, src, cfg.MaxLineWidth, cfg.TabWidth)
		if len(candidates) == 0 {
			continue
		}
		src = analysis.Fix(fset, src, candidates)
		var err error
		fset, file, err = reparse(path, src)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return src, fset, file, nil
}

func applyOrderFixer(
	path string,
	src []byte,
	fset *token.FileSet,
	file *ast.File,
	cfg Config,
) ([]byte, *token.FileSet, *ast.File, error) {
	if !cfg.runRule(RuleFuncOrder) {
		return src, fset, file, nil
	}
	input := checkInput{fset, path, file, src}
	return adoptOrderFix(input, funcorder.Fix(fset, file, src))
}

func adoptOrderFix(
	input checkInput, fixed []byte,
) ([]byte, *token.FileSet, *ast.File, error) {
	if fixed == nil {
		return input.src, input.fset, input.f, nil
	}
	fset, file, err := reparse(input.path, fixed)
	if err != nil {
		return input.src, input.fset, input.f, err
	}
	return fixed, fset, file, nil
}
