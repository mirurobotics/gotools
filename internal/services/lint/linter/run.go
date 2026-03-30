package linter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/collapse"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/ctxpos"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/errfmt"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcinline"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funclen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/funcsig"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/imports"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/linelen"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/nestdepth"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/nofmt"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/paramcount"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/pkgname"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/rcvrname"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/typeassert"
)

// Rule identifies a linter rule.
type Rule string

const (
	RulePkgName    Rule = "pkgname"
	RuleLineLen    Rule = "linelen"
	RuleFuncLen    Rule = "funclen"
	RuleNestDepth  Rule = "nestdepth"
	RuleErrFmt     Rule = "errfmt"
	RuleCtxPos     Rule = "ctxpos"
	RuleNoFmt      Rule = "nofmt"
	RuleRcvrName   Rule = "rcvrname"
	RuleTypeAssert Rule = "typeassert"
	RuleParamCount Rule = "paramcount"
	RuleImports    Rule = "imports"
	RuleCollapse   Rule = "collapse"
	RuleFuncSig    Rule = "funcsig"
	RuleFuncInline Rule = "funcinline"
)

// AllRules contains every valid rule name.
var AllRules = []Rule{
	RulePkgName, RuleLineLen, RuleFuncLen, RuleNestDepth,
	RuleErrFmt, RuleCtxPos, RuleNoFmt, RuleRcvrName,
	RuleTypeAssert, RuleParamCount,
	RuleImports, RuleCollapse, RuleFuncSig, RuleFuncInline,
}

// Config holds linter configuration.
type Config struct {
	MaxLineWidth  int
	TabWidth      int
	MaxFuncLen    int
	MaxNestDepth  int
	MaxParamCount int
	Exclude       map[Rule]bool
	Out           io.Writer
	Err           io.Writer
}

func (c Config) runRule(r Rule) bool { return !c.Exclude[r] }

// ValidateExclusions returns an error if any key in exclude is not a known rule name.
func ValidateExclusions(exclude map[Rule]bool) error {
	valid := make(map[Rule]bool, len(AllRules))
	for _, r := range AllRules {
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

// Run processes all Go files under path. Returns (diagnosticCount, filesFixed, error).
func Run(path string, doFix bool, cfg Config) (int, int, error) {
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.Err == nil {
		cfg.Err = os.Stderr
	}

	files, err := FindGoFiles(path)
	if err != nil {
		return 0, 0, err
	}

	totalDiags := 0
	filesFixed := 0

	for _, f := range files {
		diags, fixed, err := ProcessFile(f, doFix, cfg)
		if err != nil {
			_, _ = fmt.Fprintf(cfg.Err, "warning: %s: %v\n", f, err)
			continue
		}
		totalDiags += diags
		if fixed {
			filesFixed++
		}
	}

	return totalDiags, filesFixed, nil
}

// ProcessFile lints or fixes a single file.
// Returns (diagnosticCount, wasFixed, error).
func ProcessFile(path string, doFix bool, cfg Config) (int, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, false, err
	}
	// #nosec G304 -- this tool intentionally reads the caller-specified file path.
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, false, err
	}

	if analysis.IsGeneratedFile(src) {
		return 0, false, nil
	}

	fset, f, err := reparse(path, src)
	if err != nil {
		return 0, false, err
	}

	blocks := imports.ExtractBlocks(f)

	if doFix {
		return processFixMode(path, src, fset, f, blocks, info, cfg)
	}
	return processCheckMode(path, src, fset, f, blocks, cfg)
}

func processFixMode(
	path string,
	src []byte,
	fset *token.FileSet,
	f *ast.File,
	blocks []imports.ImportBlock,
	info os.FileInfo,
	cfg Config,
) (int, bool, error) {
	fixed := src
	var err error
	if cfg.runRule(RuleImports) {
		fixed = imports.Fix(fset, src, blocks)
		if string(fixed) != string(src) {
			fset, f, err = reparse(path, fixed)
			if err != nil {
				return 0, false, err
			}
		}
	}

	fixed, fset, f, err = applyCandidateFixers(path, fixed, fset, f, cfg)
	if err != nil {
		return 0, false, err
	}

	didFix := string(fixed) != string(src)
	if didFix {
		//nolint:gosec // path is from FindGoFiles, not user input
		if err := os.WriteFile(path, fixed, info.Mode()); err != nil {
			return 0, false, err
		}
		_, _ = fmt.Fprintf(cfg.Out, "fixed: %s\n", path)
	}

	in := checkInput{fset, path, f, fixed}
	diags := runChecks(in, cfg, false)
	for _, d := range diags {
		_, _ = fmt.Fprintln(cfg.Out, d)
	}
	return len(diags), didFix, nil
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

func processCheckMode(
	path string,
	src []byte,
	fset *token.FileSet,
	f *ast.File,
	blocks []imports.ImportBlock,
	cfg Config,
) (int, bool, error) {
	in := checkInput{fset, path, f, src}
	diags := runChecks(in, cfg, true)
	if cfg.runRule(RuleImports) {
		diags = append(diags, imports.Check(fset, path, src, blocks)...)
	}
	for _, d := range diags {
		_, _ = fmt.Fprintln(cfg.Out, d)
	}
	return len(diags), false, nil
}

// reparse creates a fresh FileSet and re-parses src. Used after each fixer
// to keep the AST in sync with the modified source bytes.
func reparse(path string, src []byte) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	return fset, f, err
}

// FindGoFiles walks a directory tree and returns all .go files,
// skipping hidden directories, vendor, and testdata.
func FindGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			skip := strings.HasPrefix(name, ".") ||
				name == "vendor" ||
				name == "testdata" ||
				name == "codegen"
			if skip {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
