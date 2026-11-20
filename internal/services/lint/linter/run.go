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

	for _, filePath := range files {
		diags, fixed, err := ProcessFile(filePath, doFix, cfg)
		if err != nil {
			_, _ = fmt.Fprintf(cfg.Err, "warning: %s: %v\n", filePath, err)
			continue
		}
		totalDiags += diags
		if fixed {
			filesFixed++
		}
	}

	return totalDiags, filesFixed, nil
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

func skipDir(name string) bool {
	switch name {
	case "vendor", "testdata", "codegen":
		return true
	}
	return strings.HasPrefix(name, ".")
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

	fset, file, err := reparse(path, src)
	if err != nil {
		return 0, false, err
	}

	blocks := imports.ExtractBlocks(file)

	if doFix {
		return processFixMode(path, src, fset, file, blocks, info, cfg)
	}
	return processCheckMode(path, src, fset, file, blocks, cfg)
}

func processFixMode(
	path string,
	src []byte,
	fset *token.FileSet,
	file *ast.File,
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
		fset, file, err = reparse(path, fixed)
		if err != nil {
			return 0, false, err
		}
	}

	fixed, fset, file, err = applyCandidateFixers(path, fixed, fset, file, cfg)
	if err != nil {
		return 0, false, err
	}
	fixed, fset, file, err = applyOrderFixer(path, fixed, fset, file, cfg)
	skipped := warnFixSkipped(cfg.Err, path, err)

	didFix := string(fixed) != string(src)
	if didFix {
		//nolint:gosec // path is from FindGoFiles, not user input
		if err := os.WriteFile(path, fixed, info.Mode()); err != nil {
			return 0, false, err
		}
		_, _ = fmt.Fprintf(cfg.Out, "fixed: %s\n", path)
	}

	input := checkInput{fset, path, file, fixed}
	diags := runChecks(input, cfg, false)
	for _, diag := range diags {
		_, _ = fmt.Fprintln(cfg.Out, diag)
	}
	return len(diags) + skipped, didFix, nil
}

func reparse(path string, src []byte) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	return fset, file, err
}

func warnFixSkipped(out io.Writer, path string, err error) int {
	if err == nil {
		return 0
	}
	const msg = "warning: %s: funcorder fix produced invalid Go: %v\n"
	_, _ = fmt.Fprintf(out, msg, path, err)
	return 1
}

func processCheckMode(
	path string,
	src []byte,
	fset *token.FileSet,
	file *ast.File,
	blocks []imports.ImportBlock,
	cfg Config,
) (int, bool, error) {
	input := checkInput{fset, path, file, src}
	diags := runChecks(input, cfg, true)
	if cfg.runRule(RuleImports) {
		diags = append(diags, imports.Check(fset, path, src, blocks)...)
	}
	for _, diag := range diags {
		_, _ = fmt.Fprintln(cfg.Out, diag)
	}
	return len(diags), false, nil
}
