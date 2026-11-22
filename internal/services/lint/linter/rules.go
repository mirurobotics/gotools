package linter

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/bgctx"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/collapse"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/commentedcode"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/ctxpos"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/doclen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/docrestate"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/errfmt"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/exprwrap"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/fieldassert"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/filelen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funccomment"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcinline"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funclen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcorder"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcsig"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/importalias"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/linelen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/mutableglobal"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/namelen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/nestdepth"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/nofmt"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/paramcount"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/pkgname"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/privdoc"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/rcvrname"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/straycomment"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/tempdir"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/todoissue"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/typeassert"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/varname"
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
	RuleFuncComment   Rule = "funccomment"
	RuleDocLen        Rule = "doclen"
	RuleStrayComment  Rule = "straycomment"
	RulePrivDoc       Rule = "privdoc"
	RuleExprWrap      Rule = "exprwrap"
	RuleDocRestate    Rule = "docrestate"
	RuleCommentedCode Rule = "commentedcode"
	RuleFuncOrder     Rule = "funcorder"
	RuleFieldAssert   Rule = "fieldassert"
	RuleVarName       Rule = "varname"
	RuleNameLen       Rule = "namelen"
)

// Config holds linter configuration.
type Config struct {
	MaxLineWidth    int
	TabWidth        int
	MaxFuncLen      int
	MaxFileLen      int
	MaxNestDepth    int
	MaxParamCount   int
	MaxDocLines     int
	MaxDocFileLines int
	Exclude         map[Rule]bool
	Out             io.Writer
	Err             io.Writer
}

func (c Config) runRule(rule Rule) bool { return !c.Exclude[rule] }

// ValidateExclusions returns an error if any key in exclude is not a known rule name.
func ValidateExclusions(exclude map[Rule]bool) error {
	all := AllRules()
	valid := make(map[Rule]bool, len(all))
	for _, rule := range all {
		valid[rule] = true
	}
	for rule := range exclude {
		if !valid[rule] {
			return fmt.Errorf("unknown lint rule: %q", rule)
		}
	}
	return nil
}

// AllRules returns every valid rule name.
func AllRules() []Rule {
	return []Rule{
		RulePkgName, RuleLineLen, RuleFuncLen, RuleFileLen, RuleNestDepth,
		RuleErrFmt, RuleCtxPos, RuleNoFmt, RuleRcvrName,
		RuleTypeAssert, RuleParamCount,
		RuleMutableGlobal, RuleImportAlias, RuleBgCtx, RuleTempDir,
		RuleTodoIssue, RuleExprWrap, RuleFieldAssert, RuleVarName, RuleNameLen,
		RuleFuncComment, RuleDocLen, RuleStrayComment, RulePrivDoc,
		RuleDocRestate, RuleCommentedCode,
		RuleImports, RuleCollapse, RuleFuncSig, RuleFuncInline, RuleFuncOrder,
	}
}

type checkInput struct {
	fset *token.FileSet
	path string
	f    *ast.File
	src  []byte
}

func runChecks(
	input checkInput, cfg Config, includeFixable bool,
) []analysis.Diagnostic {
	var diags []analysis.Diagnostic
	for _, entry := range ruleCheckers(cfg) {
		if !includeFixable && entry.fixable {
			continue
		}
		if !cfg.runRule(entry.rule) {
			continue
		}
		diags = append(diags, entry.check(input)...)
	}
	return diags
}

type ruleEntry struct {
	rule    Rule
	fixable bool
	check   func(checkInput) []analysis.Diagnostic
}

func ruleCheckers(cfg Config) []ruleEntry {
	width, tabWidth := cfg.MaxLineWidth, cfg.TabWidth
	return []ruleEntry{
		{RulePkgName, false, func(input checkInput) []analysis.Diagnostic {
			return pkgname.Check(input.fset, input.path, input.f)
		}},
		{RuleLineLen, false, func(input checkInput) []analysis.Diagnostic {
			return linelen.Check(
				input.fset, input.path, input.f, input.src, width, tabWidth,
			)
		}},
		{RuleFuncLen, false, func(input checkInput) []analysis.Diagnostic {
			return funclen.Check(
				input.fset, input.path, input.f, input.src, cfg.MaxFuncLen,
			)
		}},
		{RuleFileLen, false, func(input checkInput) []analysis.Diagnostic {
			return filelen.Check(
				input.fset, input.path, input.f, input.src, cfg.MaxFileLen,
			)
		}},
		{RuleNestDepth, false, func(input checkInput) []analysis.Diagnostic {
			return nestdepth.Check(input.fset, input.path, input.f, cfg.MaxNestDepth)
		}},
		{RuleErrFmt, false, func(input checkInput) []analysis.Diagnostic {
			return errfmt.Check(input.fset, input.path, input.f)
		}},
		{RuleCtxPos, false, func(input checkInput) []analysis.Diagnostic {
			return ctxpos.Check(input.fset, input.path, input.f)
		}},
		{RuleNoFmt, false, func(input checkInput) []analysis.Diagnostic {
			return nofmt.Check(input.fset, input.path, input.f)
		}},
		{RuleRcvrName, false, func(input checkInput) []analysis.Diagnostic {
			return rcvrname.Check(input.fset, input.path, input.f)
		}},
		{RuleTypeAssert, false, func(input checkInput) []analysis.Diagnostic {
			return typeassert.Check(input.fset, input.path, input.f)
		}},
		{RuleParamCount, false, func(input checkInput) []analysis.Diagnostic {
			return paramcount.Check(input.fset, input.path, input.f, cfg.MaxParamCount)
		}},
		{RuleMutableGlobal, false, func(input checkInput) []analysis.Diagnostic {
			return mutableglobal.Check(input.fset, input.path, input.f)
		}},
		{RuleImportAlias, false, func(input checkInput) []analysis.Diagnostic {
			return importalias.Check(input.fset, input.path, input.f)
		}},
		{RuleBgCtx, false, func(input checkInput) []analysis.Diagnostic {
			return bgctx.Check(input.fset, input.path, input.f)
		}},
		{RuleTempDir, false, func(input checkInput) []analysis.Diagnostic {
			return tempdir.Check(input.fset, input.path, input.f)
		}},
		{RuleTodoIssue, false, func(input checkInput) []analysis.Diagnostic {
			return todoissue.Check(input.fset, input.path, input.f)
		}},
		{RuleFieldAssert, false, func(input checkInput) []analysis.Diagnostic {
			return fieldassert.Check(input.fset, input.path, input.f)
		}},
		{RuleVarName, false, func(input checkInput) []analysis.Diagnostic {
			return varname.Check(input.fset, input.path, input.f)
		}},
		{RuleNameLen, false, func(input checkInput) []analysis.Diagnostic {
			return namelen.Check(input.fset, input.path, input.f)
		}},
		{RuleFuncComment, false, func(input checkInput) []analysis.Diagnostic {
			return funccomment.Check(input.fset, input.path, input.f)
		}},
		{RuleDocLen, false, func(input checkInput) []analysis.Diagnostic {
			lim := doclen.Limits{Doc: cfg.MaxDocLines, DocGoPkg: cfg.MaxDocFileLines}
			return doclen.Check(input.fset, input.path, input.f, lim)
		}},
		{RuleStrayComment, false, func(input checkInput) []analysis.Diagnostic {
			return straycomment.Check(input.fset, input.path, input.f)
		}},
		{RulePrivDoc, false, func(input checkInput) []analysis.Diagnostic {
			return privdoc.Check(input.fset, input.path, input.f)
		}},
		{RuleExprWrap, false, func(input checkInput) []analysis.Diagnostic {
			return exprwrap.Check(input.fset, input.path, input.f)
		}},
		{RuleDocRestate, false, func(input checkInput) []analysis.Diagnostic {
			return docrestate.Check(input.fset, input.path, input.f)
		}},
		{RuleCommentedCode, false, func(input checkInput) []analysis.Diagnostic {
			return commentedcode.Check(input.fset, input.path, input.f, input.src)
		}},
		{RuleCollapse, true, func(input checkInput) []analysis.Diagnostic {
			return collapse.Check(
				input.fset, input.path, input.f, input.src, width, tabWidth,
			)
		}},
		{RuleFuncSig, true, func(input checkInput) []analysis.Diagnostic {
			return funcsig.Check(
				input.fset, input.path, input.f, input.src, width, tabWidth,
			)
		}},
		{RuleFuncInline, true, func(input checkInput) []analysis.Diagnostic {
			return funcinline.Check(
				input.fset, input.path, input.f, input.src, width, tabWidth,
			)
		}},
		{RuleFuncOrder, true, func(input checkInput) []analysis.Diagnostic {
			return funcorder.Check(input.fset, input.path, input.f)
		}},
	}
}
