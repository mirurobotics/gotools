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
}

// effectiveParallelism and childGOMAXPROCS are intentionally
// duplicated in covgate; keep them in sync.
func effectiveParallelism(opts Opts) int {
	if opts.Parallelism > 0 {
		return opts.Parallelism
	}
	return runtime.GOMAXPROCS(0)
}

func childGOMAXPROCS(parallelism int) int {
	n := runtime.GOMAXPROCS(0) / parallelism
	if n < 1 {
		return 1
	}
	return n
}

// Run ratchets up .covgate thresholds.
func Run(opts Opts) error {
	parallelism := effectiveParallelism(opts)
	extraEnv := []string{fmt.Sprintf("GOMAXPROCS=%d", childGOMAXPROCS(parallelism))}
	r := runner{
		goModule:       gocover.GoModule,
		goListPackages: gocover.GoListPackages,
		measure: func(pkg string, testPaths []string) (float64, []byte, error) {
			return gocover.MeasureWithEnv(pkg, testPaths, extraEnv)
		},
		parallelism: parallelism,
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

	printHeader(w)

	results := make([]ratchetResult, len(pkgs))
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup

	for i, pkg := range pkgs {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, p string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = r.ratchetPackage(p, module, opts.SrcPrefix, opts.TestDir)
		}(i, pkg)
	}
	wg.Wait()

	updated := 0
	unchanged := 0
	failed := 0
	for _, res := range results {
		_, _ = fmt.Fprint(w, res.output)
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
