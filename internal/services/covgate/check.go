package covgate

import (
	"fmt"
	"strings"
	"time"

	"github.com/mirurobotics/gotools/internal/services/gocover"
)

// checkItem checks one work item, choosing the test-only check for
// orphan test packages.
func (r *runner) checkItem(it workItem, ctx checkPackageCtx) checkResult {
	if it.testOnly {
		return r.checkTestOnly(it.pkg, ctx)
	}
	return r.checkPackage(it.pkg, ctx)
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
	threshold, hasExplicitCovgate := packageThreshold(relPkg, ctx.threshold)
	testPaths := gocover.BuildTestPaths(pkg, relPkg, ctx.srcPrefix, ctx.testDir)

	start := time.Now()
	coverage, output, testErr := r.measure(pkg, testPaths)
	elapsed := time.Since(start)

	if testErr != nil {
		return testsFailedResult(relPkg, output, elapsed)
	}

	row := func(status, suffix string) string {
		return fmt.Sprintf(
			"%-7s  %7.1f%%  %7.1f%%  %8s  %s%s\n",
			status, coverage, threshold, fmtDuration(elapsed), relPkg, suffix,
		)
	}
	if coverage < threshold {
		return checkResult{row("FAIL", ""), false, elapsed}
	}
	if ctx.tightnessEnabled && hasExplicitCovgate && coverage > 0 {
		if gap := coverage - threshold; gap > ctx.tightnessTolerance {
			suffix := fmt.Sprintf(
				" (required lags actual by %.1fpp; update .covgate to >= %.1f)",
				gap, coverage-ctx.tightnessTolerance,
			)
			return checkResult{row("LOOSE", suffix), false, elapsed}
		}
	}
	return checkResult{row("PASS", ""), true, elapsed}
}

// packageThreshold returns the package's .covgate threshold if it has one,
// otherwise the default. explicit reports whether a .covgate was found.
func packageThreshold(relPkg string, def float64) (threshold float64, explicit bool) {
	if v, ok := gocover.LookupThreshold("./" + relPkg); ok {
		return v, true
	}
	return def, false
}

// testsFailedResult formats a FAIL row for a package whose tests failed,
// followed by the test output.
func testsFailedResult(
	relPkg string, output []byte, elapsed time.Duration,
) checkResult {
	var b strings.Builder
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

// checkTestOnly runs a test-only package without coverage
// instrumentation. The coverage columns render as "---": the
// package exercises source spread across many packages, so there
// is no single threshold to hold it to; the gate is that its
// tests pass.
func (r *runner) checkTestOnly(pkg string, ctx checkPackageCtx) checkResult {
	relPkg := gocover.RelPkg(pkg, ctx.module)

	start := time.Now()
	output, testErr := r.runTests([]string{"./" + relPkg})
	elapsed := time.Since(start)

	if testErr != nil {
		return testsFailedResult(relPkg, output, elapsed)
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(
		&b, "%-7s  %8s  %8s  %8s  %s\n",
		"PASS", "---", "---", fmtDuration(elapsed), relPkg,
	)
	return checkResult{b.String(), true, elapsed}
}
