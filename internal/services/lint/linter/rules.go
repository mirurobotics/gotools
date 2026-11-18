package linter

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/bgctx"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/collapse"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/ctxpos"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/errfmt"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/filelen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcinline"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funclen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcsig"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/importalias"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/linelen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/mutableglobal"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/nestdepth"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/nofmt"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/paramcount"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/pkgname"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/rcvrname"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/tempdir"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/todoissue"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/typeassert"
)

// Rule identifies a linter rule.
type Rule string

const (
	RulePkgName       Rule = "pkgname"
	RuleLineLen       Rule = "linelen"
	RuleFuncLen       Rule = "funclen"
	RuleFileLen       Rule = "filelen"
	RuleNestDepth     Rule = "nestdepth"
	RuleErrFmt        Rule = "errfmt"
	RuleCtxPos        Rule = "ctxpos"
	RuleNoFmt         Rule = "nofmt"
	RuleRcvrName      Rule = "rcvrname"
	RuleTypeAssert    Rule = "typeassert"
	RuleParamCount    Rule = "paramcount"
	RuleImports       Rule = "imports"
	RuleCollapse      Rule = "collapse"
	RuleFuncSig       Rule = "funcsig"
	RuleFuncInline    Rule = "funcinline"
	RuleMutableGlobal Rule = "mutableglobal"
	RuleImportAlias   Rule = "importalias"
	RuleBgCtx         Rule = "bgctx"
	RuleTempDir       Rule = "tempdir"
	RuleTodoIssue     Rule = "todoissue"
)

// AllRules returns every valid rule name.
func AllRules() []Rule {
	return []Rule{
		RulePkgName, RuleLineLen, RuleFuncLen, RuleFileLen, RuleNestDepth,
		RuleErrFmt, RuleCtxPos, RuleNoFmt, RuleRcvrName,
		RuleTypeAssert, RuleParamCount,
		RuleMutableGlobal, RuleImportAlias, RuleBgCtx, RuleTempDir,
		RuleTodoIssue,
		RuleImports, RuleCollapse, RuleFuncSig, RuleFuncInline,
	}
}

// Config holds linter configuration.
type Config struct {
	MaxLineWidth  int
	TabWidth      int
	MaxFuncLen    int
	MaxFileLen    int
	MaxNestDepth  int
	MaxParamCount int
	Exclude       map[Rule]bool
	Out           io.Writer
	Err           io.Writer
}

func (c Config) runRule(r Rule) bool { return !c.Exclude[r] }

// ValidateExclusions returns an error if any key in exclude is not a known rule name.
func ValidateExclusions(exclude map[Rule]bool) error {
	all := AllRules()
	valid := make(map[Rule]bool, len(all))
	for _, r := range all {
		valid[r] = true
	}
	for r := range exclude {
		if !valid[r] {
			return fmt.Errorf("unknown lint rule: %q", r)
		}
	}
	return nil
}

type checkInput struct {
	fset *token.FileSet
	path string
	f    *ast.File
	src  []byte
}

type ruleEntry struct {
	rule    Rule
	fixable bool
	check   func(checkInput) []analysis.Diagnostic
}

// ruleCheckers returns the dispatch table for all rules except
// imports (which requires import blocks, handled separately).
//
//nolint:funclen // dispatch table grows with each new rule
func ruleCheckers(cfg Config) []ruleEntry {
	w, tw := cfg.MaxLineWidth, cfg.TabWidth
	return []ruleEntry{
		{RulePkgName, false, func(in checkInput) []analysis.Diagnostic {
			return pkgname.Check(in.fset, in.path, in.f)
		}},
		{RuleLineLen, false, func(in checkInput) []analysis.Diagnostic {
			return linelen.Check(in.fset, in.path, in.f, in.src, w, tw)
		}},
		{RuleFuncLen, false, func(in checkInput) []analysis.Diagnostic {
			return funclen.Check(in.fset, in.path, in.f, in.src, cfg.MaxFuncLen)
		}},
		{RuleFileLen, false, func(in checkInput) []analysis.Diagnostic {
			return filelen.Check(in.fset, in.path, in.f, in.src, cfg.MaxFileLen)
		}},
		{RuleNestDepth, false, func(in checkInput) []analysis.Diagnostic {
			return nestdepth.Check(in.fset, in.path, in.f, cfg.MaxNestDepth)
		}},
		{RuleErrFmt, false, func(in checkInput) []analysis.Diagnostic {
			return errfmt.Check(in.fset, in.path, in.f)
		}},
		{RuleCtxPos, false, func(in checkInput) []analysis.Diagnostic {
			return ctxpos.Check(in.fset, in.path, in.f)
		}},
		{RuleNoFmt, false, func(in checkInput) []analysis.Diagnostic {
			return nofmt.Check(in.fset, in.path, in.f)
		}},
		{RuleRcvrName, false, func(in checkInput) []analysis.Diagnostic {
			return rcvrname.Check(in.fset, in.path, in.f)
		}},
		{RuleTypeAssert, false, func(in checkInput) []analysis.Diagnostic {
			return typeassert.Check(in.fset, in.path, in.f)
		}},
		{RuleParamCount, false, func(in checkInput) []analysis.Diagnostic {
			return paramcount.Check(in.fset, in.path, in.f, cfg.MaxParamCount)
		}},
		{RuleMutableGlobal, false, func(in checkInput) []analysis.Diagnostic {
			return mutableglobal.Check(in.fset, in.path, in.f)
		}},
		{RuleImportAlias, false, func(in checkInput) []analysis.Diagnostic {
			return importalias.Check(in.fset, in.path, in.f)
		}},
		{RuleBgCtx, false, func(in checkInput) []analysis.Diagnostic {
			return bgctx.Check(in.fset, in.path, in.f)
		}},
		{RuleTempDir, false, func(in checkInput) []analysis.Diagnostic {
			return tempdir.Check(in.fset, in.path, in.f)
		}},
		{RuleTodoIssue, false, func(in checkInput) []analysis.Diagnostic {
			return todoissue.Check(in.fset, in.path, in.f)
		}},
		{RuleCollapse, true, func(in checkInput) []analysis.Diagnostic {
			return collapse.Check(in.fset, in.path, in.f, in.src, w, tw)
		}},
		{RuleFuncSig, true, func(in checkInput) []analysis.Diagnostic {
			return funcsig.Check(in.fset, in.path, in.f, in.src, w, tw)
		}},
		{RuleFuncInline, true, func(in checkInput) []analysis.Diagnostic {
			return funcinline.Check(in.fset, in.path, in.f, in.src, w, tw)
		}},
	}
}

// runChecks dispatches all rules in the table. When
// includeFixable is false, fixable rules are skipped.
func runChecks(in checkInput, cfg Config, includeFixable bool) []analysis.Diagnostic {
	var diags []analysis.Diagnostic
	for _, rc := range ruleCheckers(cfg) {
		if !includeFixable && rc.fixable {
			continue
		}
		if !cfg.runRule(rc.rule) {
			continue
		}
		diags = append(diags, rc.check(in)...)
	}
	return diags
}

func applyCandidateFixers(
	path string,
	src []byte,
	fset *token.FileSet,
	f *ast.File,
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
	for _, fx := range fixers {
		if !cfg.runRule(fx.rule) {
			continue
		}
		candidates := fx.find(fset, f, src, cfg.MaxLineWidth, cfg.TabWidth)
		src = analysis.Fix(fset, src, candidates)
		if len(candidates) > 0 {
			var err error
			fset, f, err = reparse(path, src)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}
	return src, fset, f, nil
}
