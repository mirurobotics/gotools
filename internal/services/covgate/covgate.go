package covgate

import (
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mirurobotics/gotools/internal/services/gocover"
)

// Opts holds the options for the covgate service.
type Opts struct {
	Packages           string
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
}

// Run checks per-package coverage against thresholds.
func Run(opts Opts) error {
	r := runner{
		goModule:       gocover.GoModule,
		goListPackages: gocover.GoListPackages,
		measure:        gocover.Measure,
	}
	return r.run(opts)
}

func (r *runner) run(opts Opts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	w := opts.Out

	parallelism := opts.Parallelism
	if parallelism <= 0 {
		parallelism = runtime.NumCPU()
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

	printHeader(w)

	ctx := checkPackageCtx{
		module:             module,
		srcPrefix:          opts.SrcPrefix,
		testDir:            opts.TestDir,
		threshold:          opts.DefaultThreshold,
		tightnessEnabled:   opts.TightnessEnabled,
		tightnessTolerance: opts.TightnessTolerance,
	}

	start := time.Now()
	results := r.runPackages(pkgs, ctx, parallelism)
	wallTime := time.Since(start)
	return r.printResults(w, results, wallTime)
}

func (r *runner) runPackages(
	pkgs []string, ctx checkPackageCtx, parallelism int,
) []checkResult {
	results := make([]checkResult, len(pkgs))
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup

	for i, pkg := range pkgs {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, p string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = r.checkPackage(p, ctx)
		}(i, pkg)
	}
	wg.Wait()
	return results
}

func (r *runner) printResults(
	w io.Writer, results []checkResult, totalTime time.Duration,
) error {
	hasFailures := false
	for _, res := range results {
		_, _ = fmt.Fprint(w, res.output)
		if !res.passed {
			hasFailures = true
		}
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
		w, "%-6s  %8s  %8s  %8s  %s\n",
		"STATUS", "COVERAGE", "REQUIRED", "TIME", "PACKAGE",
	)
	_, _ = fmt.Fprintf(
		w, "%-6s  %8s  %8s  %8s  %s\n",
		"------", "--------", "--------", "--------", "-------",
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
			&b, "%-6s  %8s  %8s  %8s  %s\n",
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
			&b, "%-6s  %7.1f%%  %7.1f%%  %8s  %s\n",
			"FAIL", coverage, threshold, fmtDuration(elapsed), relPkg,
		)
		return checkResult{b.String(), false, elapsed}
	}

	if ctx.tightnessEnabled && hasExplicitCovgate && coverage > 0 {
		gap := coverage - threshold
		if gap > ctx.tightnessTolerance {
			recommended := coverage - ctx.tightnessTolerance
			recommended = math.Ceil(recommended*10) / 10
			_, _ = fmt.Fprintf(
				&b, "%-6s  %7.1f%%  %7.1f%%  %8s  "+
					"%s (required lags actual by %.1fpp; "+
					"update .covgate to >= %.1f)\n",
				"LOOSE", coverage, threshold, fmtDuration(elapsed),
				relPkg, gap, recommended,
			)
			return checkResult{b.String(), false, elapsed}
		}
	}

	_, _ = fmt.Fprintf(
		&b, "%-6s  %7.1f%%  %7.1f%%  %8s  %s\n",
		"PASS", coverage, threshold, fmtDuration(elapsed), relPkg,
	)
	return checkResult{b.String(), true, elapsed}
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
