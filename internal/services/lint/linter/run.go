package linter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter/analysis"
	"github.com/mirurobotics/gotools/internal/services/lint/linter/imports"
)

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

// FindGoFiles returns .go files under root, skipping hidden, vendor, testdata, codegen.
func FindGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if skipDir(info.Name()) {
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
	if cfg.runRule(RuleImports) {
		fixed = imports.Fix(fset, src, blocks)
	}
	var err error
	if string(fixed) != string(src) {
		fset, f, err = reparse(path, fixed)
		if err != nil {
			return 0, false, err
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

func reparse(path string, src []byte) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	return fset, f, err
}

func skipDir(name string) bool {
	switch name {
	case "vendor", "testdata", "codegen":
		return true
	}
	return strings.HasPrefix(name, ".")
}
