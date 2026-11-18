package covratchet

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/gocover"
)

// Opts holds the options for the covratchet service.
type Opts struct {
	Packages    string
	SrcPrefix   string
	TestDir     string
	Parallelism int
	Out         io.Writer
}

type runner struct {
	goModule       func() (string, error)
	goListPackages func(string) ([]string, error)
	measure        func(pkg string, testPaths []string) (float64, []byte, error)
	emitProgress   bool
}

// Run ratchets up .covgate thresholds.
func Run(opts Opts) error {
	r := runner{
		goModule:       gocover.GoModule,
		goListPackages: gocover.GoListPackages,
		measure:        gocover.Measure,
		emitProgress:   opts.Parallelism == 0,
	}
	return r.run(opts)
}

func (r *runner) run(opts Opts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	w := opts.Out

	parallelism := gocover.Parallelism(opts.Parallelism)

	_, _ = fmt.Fprintln(w, "Updating .covgate files (ratchet up only)...")
	_, _ = fmt.Fprintln(w)

	module, err := r.goModule()
	if err != nil {
		return err
	}

	pkgs, err := r.goListPackages(opts.Packages)
	if err != nil {
		return err
	}

	r.progress().WriteHeader(w, len(pkgs), parallelism)
	ctx := ratchetCtx{
		module:    module,
		srcPrefix: opts.SrcPrefix,
		testDir:   opts.TestDir,
	}
	results := r.runPackages(pkgs, ctx, parallelism, w)

	updated, unchanged, failed := 0, 0, 0
	for _, res := range results {
		if !r.emitProgress {
			_, _ = fmt.Fprint(w, res.output)
		}
		updated += res.updated
		unchanged += res.unchanged
		failed += res.failed
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(
		w, "Done. Updated: %d, Unchanged: %d, Failed: %d\n",
		updated, unchanged, failed,
	)
	if failed > 0 {
		return fmt.Errorf("%d package(s) failed", failed)
	}
	return nil
}

type ratchetCtx struct {
	module    string
	srcPrefix string
	testDir   string
}

func (r *runner) runPackages(
	pkgs []string, ctx ratchetCtx, parallelism int, w io.Writer,
) []ratchetResult {
	run := func(i int) (ratchetResult, string) {
		res := r.ratchetPackage(pkgs[i], ctx.module, ctx.srcPrefix, ctx.testDir)
		return res, res.output
	}
	return gocover.Run(r.progress(), w, len(pkgs), parallelism, run)
}

func (r *runner) progress() gocover.Progress {
	return gocover.Progress{Enabled: r.emitProgress, Table: table()}
}

func table() gocover.Table {
	return gocover.Table{
		Format: "%-6s  %8s  %8s  %s\n",
		Names:  []string{"STATUS", "PREVIOUS", "CURRENT", "PACKAGE"},
		Dashes: []string{"------", "--------", "-------", "-------"},
	}
}

type ratchetResult struct {
	output    string
	updated   int
	unchanged int
	failed    int
}

func (r *runner) ratchetPackage(pkg, module, srcPrefix, testDir string) ratchetResult {
	relPkg := gocover.RelPkg(pkg, module)
	pkgDir := "./" + relPkg
	covgateFile := pkgDir + "/.covgate"

	current := readCovgate(covgateFile)
	testPaths := gocover.BuildTestPaths(pkg, relPkg, srcPrefix, testDir)
	actual, _, err := r.measure(pkg, testPaths)
	if err != nil {
		line := fmt.Sprintf("%-6s  %8s  %8s  %s\n", "FAIL", "\u2014", "\u2014", relPkg)
		return ratchetResult{output: line, updated: 0, unchanged: 0, failed: 1}
	}

	if current == "" {
		return writeNewCovgate(covgateFile, actual, relPkg)
	}

	currentVal, err := strconv.ParseFloat(current, 64)
	if err != nil {
		line := fmt.Sprintf(
			"%-6s  %7s%%  %7.1f%%  %s (parse .covgate: %v)\n",
			"FAIL", current, actual, relPkg, err,
		)
		return ratchetResult{output: line, updated: 0, unchanged: 0, failed: 1}
	}
	if actual > currentVal {
		if err := writeCovgate(covgateFile, actual); err != nil {
			line := fmt.Sprintf(
				"%-6s  %7s%%  %7.1f%%  %s (%v)\n",
				"FAIL", current, actual, relPkg, err,
			)
			return ratchetResult{output: line, updated: 0, unchanged: 0, failed: 1}
		}
		line := fmt.Sprintf("%-6s  %7s%%  %7.1f%%  %s\n", "UP", current, actual, relPkg)
		return ratchetResult{output: line, updated: 1, unchanged: 0, failed: 0}
	}

	line := fmt.Sprintf("%-6s  %7s%%  %7.1f%%  %s\n", "OK", current, actual, relPkg)
	return ratchetResult{output: line, updated: 0, unchanged: 1, failed: 0}
}

func writeNewCovgate(covgateFile string, actual float64, relPkg string) ratchetResult {
	if err := writeCovgate(covgateFile, actual); err != nil {
		line := fmt.Sprintf(
			"%-6s  %8s  %7.1f%%  %s (%v)\n",
			"FAIL", "\u2014", actual, relPkg, err,
		)
		return ratchetResult{output: line, updated: 0, unchanged: 0, failed: 1}
	}
	line := fmt.Sprintf("%-6s  %8s  %7.1f%%  %s\n", "NEW", "\u2014", actual, relPkg)
	return ratchetResult{output: line, updated: 1, unchanged: 0, failed: 0}
}

func readCovgate(path string) string {
	//nolint:gosec // G304: trusted file path
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(data), "\n")[0])
}

func writeCovgate(path string, coverage float64) error {
	content := fmt.Sprintf("%.1f\n", coverage)
	//nolint:gosec // G306: intentional 0644
	return os.WriteFile(path, []byte(content), 0o644)
}
