package covratchet

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

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
	parallelism    int
	emitProgress   bool
}

// effectiveParallelism is intentionally duplicated in covgate;
// keep them in sync.
func effectiveParallelism(opts Opts) int {
	if opts.Parallelism > 0 {
		return opts.Parallelism
	}
	return runtime.NumCPU()
}

// Run ratchets up .covgate thresholds.
func Run(opts Opts) error {
	r := runner{
		goModule:       gocover.GoModule,
		goListPackages: gocover.GoListPackages,
		measure:        gocover.Measure,
		parallelism:    effectiveParallelism(opts),
		emitProgress:   opts.Parallelism == 0,
	}
	return r.run(opts)
}

func (r *runner) run(opts Opts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	w := opts.Out

	parallelism := r.parallelism
	if parallelism <= 0 {
		parallelism = effectiveParallelism(opts)
	}

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

	r.writeRunHeader(w, len(pkgs), parallelism)
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

// writeRunHeader writes the announce line and the progress table
// header when progress is enabled; otherwise it writes the
// standard table header.
func (r *runner) writeRunHeader(w io.Writer, total, parallelism int) {
	if !r.emitProgress {
		printHeader(w)
		return
	}
	_, _ = fmt.Fprintf(
		w, "Running %d packages with parallelism=%d; progress:\n",
		total, parallelism,
	)
	printProgressHeader(w, total)
}

func printHeader(w io.Writer) {
	_, _ = fmt.Fprintf(
		w, "%-6s  %8s  %8s  %s\n",
		"STATUS", "PREVIOUS", "CURRENT", "PACKAGE",
	)
	_, _ = fmt.Fprintf(
		w, "%-6s  %8s  %8s  %s\n",
		"------", "--------", "-------", "-------",
	)
}

// printProgressHeader prints the table header used during the live
// progress stream. It adds a leading COUNT column ahead of the
// standard columns so each progress line carries an [N/total] tag
// aligned under "COUNT".
func printProgressHeader(w io.Writer, total int) {
	cw := progressColWidth(total)
	dashes := strings.Repeat("-", cw)
	_, _ = fmt.Fprintf(
		w, "%-*s  %-6s  %8s  %8s  %s\n",
		cw, "COUNT", "STATUS", "PREVIOUS", "CURRENT", "PACKAGE",
	)
	_, _ = fmt.Fprintf(
		w, "%-*s  %-6s  %8s  %8s  %s\n",
		cw, dashes, "------", "--------", "-------", "-------",
	)
}

// progressColWidth returns the width of the COUNT column for total
// packages. The value 3 + 2*ndigits is always >= 5 ("COUNT") for
// total >= 0, so no clamp is needed.
func progressColWidth(total int) int { return 3 + 2*len(strconv.Itoa(total)) }

// firstLine returns the first line of s, including the trailing
// newline. s must contain at least one '\n' — every ratchetResult's
// output is built from a "%...\n" format string, so this holds.
func firstLine(s string) string { return s[:strings.IndexByte(s, '\n')+1] }

// ratchetCtx holds the per-run constants threaded into ratchetPackage.
type ratchetCtx struct {
	module    string
	srcPrefix string
	testDir   string
}

func (r *runner) runPackages(
	pkgs []string, ctx ratchetCtx, parallelism int, w io.Writer,
) []ratchetResult {
	total := len(pkgs)
	results := make([]ratchetResult, total)
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
	var progressMu sync.Mutex
	countWidth := len(strconv.Itoa(total))
	colWidth := progressColWidth(total)

	for i, pkg := range pkgs {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, p string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = r.ratchetPackage(p, ctx.module, ctx.srcPrefix, ctx.testDir)
			if r.emitProgress {
				label := fmt.Sprintf("[%*d/%d]", countWidth, idx+1, total)
				progressMu.Lock()
				_, _ = fmt.Fprintf(
					w, "%-*s  %s",
					colWidth, label, firstLine(results[idx].output),
				)
				progressMu.Unlock()
			}
		}(i, pkg)
	}
	wg.Wait()
	return results
}

// ratchetResult holds the output and counts for a single package ratchet.
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
