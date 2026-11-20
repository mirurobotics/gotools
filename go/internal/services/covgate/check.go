package covgate

import (
	"fmt"
	"strings"
	"time"

	"github.com/mirurobotics/tools/go/internal/services/gocover"
)

type checkResult struct {
	output   string
	passed   bool
	duration time.Duration
}

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

func packageThreshold(relPkg string, def float64) (threshold float64, explicit bool) {
	if v, ok := gocover.LookupThreshold("./" + relPkg); ok {
		return v, true
	}
	return def, false
}

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
