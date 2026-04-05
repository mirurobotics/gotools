package covgate

import (
	"fmt"
	"io"
	"os"

	"github.com/mirurobotics/gotools/internal/services/gocover"
)

// Opts holds the options for the covgate service.
type Opts struct {
	Packages         string
	SrcPrefix        string
	TestDir          string
	DefaultThreshold float64
	Out              io.Writer
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
		module:    module,
		srcPrefix: opts.SrcPrefix,
		testDir:   opts.TestDir,
		threshold: opts.DefaultThreshold,
		w:         w,
	}
	hasFailures := false
	for _, pkg := range pkgs {
		if !r.checkPackage(pkg, ctx) {
			hasFailures = true
		}
	}

	_, _ = fmt.Fprintln(w)
	if hasFailures {
		_, _ = fmt.Fprintln(
			w, "ERROR: One or more packages failed "+
				"tests or are below their minimum coverage",
		)
		return fmt.Errorf("coverage gate failed")
	}

	_, _ = fmt.Fprintln(w, "All packages meet minimum coverage requirement")
	return nil
}

func printHeader(w io.Writer) {
	_, _ = fmt.Fprintf(
		w, "%-6s  %8s  %8s  %s\n",
		"STATUS", "COVERAGE", "REQUIRED", "PACKAGE",
	)
	_, _ = fmt.Fprintf(
		w, "%-6s  %8s  %8s  %s\n",
		"------", "--------", "--------", "-------",
	)
}

// checkPackageCtx holds the per-run constants passed to checkPackage.
type checkPackageCtx struct {
	module    string
	srcPrefix string
	testDir   string
	threshold float64
	w         io.Writer
}

func (r *runner) checkPackage(pkg string, ctx checkPackageCtx) bool {
	relPkg := gocover.RelPkg(pkg, ctx.module)
	pkgDir := "./" + relPkg
	threshold := gocover.GetThreshold(pkgDir, ctx.threshold)

	testPaths := gocover.BuildTestPaths(pkg, relPkg, ctx.srcPrefix, ctx.testDir)

	coverage, output, testErr := r.measure(pkg, testPaths)
	if testErr != nil {
		_, _ = fmt.Fprintf(
			ctx.w, "%-6s  %8s  %8s  %s\n",
			"FAIL", "---", "---",
			relPkg+" (tests failed)",
		)
		_, _ = fmt.Fprintln(ctx.w)
		_, _ = fmt.Fprint(ctx.w, string(output))
		_, _ = fmt.Fprintln(ctx.w)
		return false
	}

	status := "PASS"
	if coverage < threshold {
		status = "FAIL"
	}
	_, _ = fmt.Fprintf(
		ctx.w, "%-6s  %7.1f%%  %7.1f%%  %s\n",
		status, coverage, threshold, relPkg,
	)
	return coverage >= threshold
}
