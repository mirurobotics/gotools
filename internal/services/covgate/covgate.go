package covgate

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mirurobotics/gotools/internal/services/gocover"
)

// Opts holds the options for the covgate service.
type Opts struct {
	Packages string
	// Exclude is a comma-separated list of Go list patterns whose
	// matched import paths are removed from the measurement set
	// before tests run. Empty means no exclusion.
	Exclude            string
	SrcPrefix          string
	TestDir            string
	DefaultThreshold   float64
	Parallelism        int
	TightnessEnabled   bool
	TightnessTolerance float64
	Out                io.Writer
}

type runner struct {
	goModule       func() (string, error)
	goListPackages func(string) ([]string, error)
	measure        func(pkg string, testPaths []string) (float64, []byte, error)
	prewarm        func(testPaths []string) error
	parallelism    int
	emitProgress   bool
}

// effectiveParallelism is intentionally duplicated in covratchet;
// keep them in sync.
func effectiveParallelism(opts Opts) int {
	if opts.Parallelism > 0 {
		return opts.Parallelism
	}
	return runtime.NumCPU()
}

// Run checks per-package coverage against thresholds.
func Run(opts Opts) error {
	r := runner{
		goModule:       gocover.GoModule,
		goListPackages: gocover.GoListPackages,
		measure:        gocover.Measure,
		prewarm:        gocover.PrewarmBuild,
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

	_, _ = fmt.Fprintf(
		w, "Checking per-package coverage "+
			"(default minimum: %.1f%%)...\n\n",
		opts.DefaultThreshold,
	)

	module, err := r.goModule()
	if err != nil {
		return err
	}

	pkgs, err := r.goListPackages(opts.Packages)
	if err != nil {
		return err
	}

	pkgs, excluded, err := r.applyExclude(pkgs, opts.Exclude, w)
	if err != nil {
		return err
	}

	ctx := checkPackageCtx{
		module:             module,
		srcPrefix:          opts.SrcPrefix,
		testDir:            opts.TestDir,
		threshold:          opts.DefaultThreshold,
		tightnessEnabled:   opts.TightnessEnabled,
		tightnessTolerance: opts.TightnessTolerance,
	}

	if parallelism > 1 && len(pkgs) > 1 {
		if err := r.prewarmCache(w, pkgs, ctx); err != nil {
			return err
		}
	}

	r.writeRunHeader(w, len(pkgs), parallelism)

	start := time.Now()
	results := r.runPackages(pkgs, ctx, parallelism, w)
	wallTime := time.Since(start)
	return r.printResults(w, results, excluded, module, wallTime)
}

// prewarmCache warms the build cache in one coherent compile pass before the
// parallel per-package runs, so they don't stampede on shared dependency
// compiles. It times the pass and reports its duration, which is otherwise
// invisible — the warm pass is excluded from "Total time", which covers only
// the parallel coverage phase.
func (r *runner) prewarmCache(w io.Writer, pkgs []string, ctx checkPackageCtx) error {
	warmStart := time.Now()
	_, _ = fmt.Fprintf(w, "Pre-warming build cache (%d packages)...\n", len(pkgs))
	if err := r.prewarm(collectWarmPaths(pkgs, ctx)); err != nil {
		return err
	}
	elapsed := fmtDuration(time.Since(warmStart))
	_, _ = fmt.Fprintf(w, "Pre-warm complete in %s\n\n", elapsed)
	return nil
}

// collectWarmPaths returns the de-duplicated union of the test
// paths covgate will build for every package, using the same
// RelPkg + BuildTestPaths logic as checkPackage so the warm
// pass compiles exactly the set the real runs need.
func collectWarmPaths(pkgs []string, ctx checkPackageCtx) []string {
	seen := make(map[string]struct{})
	var paths []string
	for _, pkg := range pkgs {
		relPkg := gocover.RelPkg(pkg, ctx.module)
		for _, p := range gocover.BuildTestPaths(
			pkg, relPkg, ctx.srcPrefix, ctx.testDir,
		) {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			paths = append(paths, p)
		}
	}
	return paths
}

// applyExclude removes packages matched by the comma-separated
// exclude patterns from pkgs. It preserves the original order of
// pkgs in both returned slices and prints a one-line notice to w
// when any package is actually removed. An empty (or
// whitespace-only) exclude string returns pkgs unchanged, a nil
// excluded slice, and no output.
func (r *runner) applyExclude(
	pkgs []string, exclude string, w io.Writer,
) (kept []string, excluded []string, err error) {
	if strings.TrimSpace(exclude) == "" {
		return pkgs, nil, nil
	}

	excludedSet := make(map[string]struct{})
	for raw := range strings.SplitSeq(exclude, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		matched, listErr := r.goListPackages(entry)
		if listErr != nil {
			return nil, nil, fmt.Errorf("resolve exclude %q: %w", entry, listErr)
		}
		for _, p := range matched {
			excludedSet[p] = struct{}{}
		}
	}

	kept = make([]string, 0, len(pkgs))
	excluded = make([]string, 0, len(excludedSet))
	for _, p := range pkgs {
		if _, drop := excludedSet[p]; drop {
			excluded = append(excluded, p)
			continue
		}
		kept = append(kept, p)
	}

	if len(excluded) > 0 {
		_, _ = fmt.Fprintf(
			w, "Excluded %d package(s) from coverage measurement\n",
			len(excluded),
		)
	}
	return kept, excluded, nil
}

func (r *runner) runPackages(
	pkgs []string, ctx checkPackageCtx, parallelism int, w io.Writer,
) []checkResult {
	total := len(pkgs)
	results := make([]checkResult, total)
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
			results[idx] = r.checkPackage(p, ctx)
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

// progressColWidth returns the width of the COUNT column for total
// packages. The value 3 + 2*ndigits is always >= 5 ("COUNT") for
// total >= 0, so no clamp is needed.
func progressColWidth(total int) int { return 3 + 2*len(strconv.Itoa(total)) }

// firstLine returns the first line of s, including the trailing
// newline. s must contain at least one '\n' — every checkResult's
// output is built from a "%...\n" format string, so this holds.
func firstLine(s string) string { return s[:strings.IndexByte(s, '\n')+1] }

// restOfOutput returns everything after the first '\n' in s. Used
// in progress mode to preserve trailing FAIL detail (multi-line
// raw test output) without reprinting the row that was already
// streamed. Same '\n' precondition as firstLine.
func restOfOutput(s string) string { return s[strings.IndexByte(s, '\n')+1:] }

func (r *runner) printResults(
	w io.Writer,
	results []checkResult,
	excluded []string,
	module string,
	totalTime time.Duration,
) error {
	hasFailures := false
	for _, res := range results {
		if r.emitProgress {
			_, _ = fmt.Fprint(w, restOfOutput(res.output))
		} else {
			_, _ = fmt.Fprint(w, res.output)
		}
		if !res.passed {
			hasFailures = true
		}
	}

	indent := ""
	if r.emitProgress {
		indent = strings.Repeat(" ", progressColWidth(len(results))) + "  "
	}
	for _, pkg := range excluded {
		row := skippedRow(gocover.RelPkg(pkg, module))
		_, _ = fmt.Fprintf(w, "%s%s", indent, row)
	}

	_, _ = fmt.Fprintf(w, "\nTotal time: %s\n", fmtDuration(totalTime))
	if hasFailures {
		_, _ = fmt.Fprintln(
			w, "ERROR: One or more packages failed "+
				"tests, are below their minimum coverage, "+
				"or have loose .covgate thresholds",
		)
		return fmt.Errorf("coverage gate failed")
	}

	_, _ = fmt.Fprintln(w, "All packages meet minimum coverage requirement")
	return nil
}

func printHeader(w io.Writer) {
	_, _ = fmt.Fprintf(
		w, "%-7s  %8s  %8s  %8s  %s\n",
		"STATUS", "COVERAGE", "REQUIRED", "TIME", "PACKAGE",
	)
	_, _ = fmt.Fprintf(
		w, "%-7s  %8s  %8s  %8s  %s\n",
		"-------", "--------", "--------", "--------", "-------",
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
		w, "%-*s  %-7s  %8s  %8s  %8s  %s\n",
		cw, "COUNT",
		"STATUS", "COVERAGE", "REQUIRED", "TIME", "PACKAGE",
	)
	_, _ = fmt.Fprintf(
		w, "%-*s  %-7s  %8s  %8s  %8s  %s\n",
		cw, dashes,
		"-------", "--------", "--------", "--------", "-------",
	)
}

// checkResult holds the output and pass/fail status for a single package check.
type checkResult struct {
	output   string // formatted line(s) to print
	passed   bool
	duration time.Duration
}

// checkPackageCtx holds the per-run constants passed to checkPackage.
type checkPackageCtx struct {
	module             string
	srcPrefix          string
	testDir            string
	threshold          float64
	tightnessEnabled   bool
	tightnessTolerance float64
}

func (r *runner) checkPackage(pkg string, ctx checkPackageCtx) checkResult {
	relPkg := gocover.RelPkg(pkg, ctx.module)
	pkgDir := "./" + relPkg
	explicit, hasExplicitCovgate := gocover.LookupThreshold(pkgDir)
	threshold := ctx.threshold
	if hasExplicitCovgate {
		threshold = explicit
	}

	testPaths := gocover.BuildTestPaths(pkg, relPkg, ctx.srcPrefix, ctx.testDir)

	start := time.Now()
	coverage, output, testErr := r.measure(pkg, testPaths)
	elapsed := time.Since(start)

	var b strings.Builder
	if testErr != nil {
		_, _ = fmt.Fprintf(
			&b, "%-7s  %8s  %8s  %8s  %s\n",
			"FAIL", "---", "---", fmtDuration(elapsed),
			relPkg+" (tests failed)",
		)
		_, _ = fmt.Fprintln(&b)
		_, _ = fmt.Fprint(&b, string(output))
		_, _ = fmt.Fprintln(&b)
		return checkResult{b.String(), false, elapsed}
	}

	if coverage < threshold {
		_, _ = fmt.Fprintf(
			&b, "%-7s  %7.1f%%  %7.1f%%  %8s  %s\n",
			"FAIL", coverage, threshold, fmtDuration(elapsed), relPkg,
		)
		return checkResult{b.String(), false, elapsed}
	}

	if ctx.tightnessEnabled && hasExplicitCovgate && coverage > 0 {
		gap := coverage - threshold
		if gap > ctx.tightnessTolerance {
			recommended := coverage - ctx.tightnessTolerance
			_, _ = fmt.Fprintf(
				&b, "%-7s  %7.1f%%  %7.1f%%  %8s  "+
					"%s (required lags actual by %.1fpp; "+
					"update .covgate to >= %.1f)\n",
				"LOOSE", coverage, threshold, fmtDuration(elapsed),
				relPkg, gap, recommended,
			)
			return checkResult{b.String(), false, elapsed}
		}
	}

	_, _ = fmt.Fprintf(
		&b, "%-7s  %7.1f%%  %7.1f%%  %8s  %s\n",
		"PASS", coverage, threshold, fmtDuration(elapsed), relPkg,
	)
	return checkResult{b.String(), true, elapsed}
}

// skippedRow formats a single SKIPPED row using the same column
// widths as PASS/FAIL/LOOSE. relPkg is the module-relative import
// path. Used for packages removed by --exclude so the user can see
// which ones were skipped.
func skippedRow(relPkg string) string {
	return fmt.Sprintf(
		"%-7s  %8s  %8s  %8s  %s\n",
		"SKIPPED", "---", "---", "---", relPkg,
	)
}

func fmtDuration(d time.Duration) string {
	d = d.Round(100 * time.Millisecond)
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := d - time.Duration(m)*time.Minute
	return fmt.Sprintf("%dm%02.0fs", m, s.Seconds())
}
