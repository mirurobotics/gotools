package covgate

import (
	"fmt"
	"io"
	"os"
	"os/exec"

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
		measure:        defaultMeasure,
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

	hasFailures := false
	for _, pkg := range pkgs {
		ok := r.checkPackage(
			pkg, module, opts.SrcPrefix,
			opts.TestDir, opts.DefaultThreshold, w,
		)
		if !ok {
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

func (r *runner) checkPackage(
	pkg, module, srcPrefix, testDir string,
	defaultThreshold float64, w io.Writer,
) bool {
	relPkg := gocover.RelPkg(pkg, module)
	pkgDir := "./" + relPkg
	threshold := gocover.GetThreshold(pkgDir, defaultThreshold)

	testPaths := gocover.BuildTestPaths(pkg, relPkg, srcPrefix, testDir)

	coverage, output, testErr := r.measure(pkg, testPaths)
	if testErr != nil {
		_, _ = fmt.Fprintf(
			w, "%-6s  %8s  %8s  %s\n",
			"FAIL", "---", "---",
			relPkg+" (tests failed)",
		)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprint(w, string(output))
		_, _ = fmt.Fprintln(w)
		return false
	}

	status := "PASS"
	if coverage < threshold {
		status = "FAIL"
	}
	_, _ = fmt.Fprintf(
		w, "%-6s  %7.1f%%  %7.1f%%  %s\n",
		status, coverage, threshold, relPkg,
	)
	return coverage >= threshold
}

func defaultMeasure(pkg string, testPaths []string) (float64, []byte, error) {
	tmpFile := "coverage.out"
	args := make([]string, 0, 3+len(testPaths))
	args = append(args, "test", "-coverprofile="+tmpFile, "-coverpkg="+pkg)
	args = append(args, testPaths...)

	//nolint:gosec,noctx // G204: trusted subprocess
	testCmd := exec.Command("go", args...)
	testCmd.Env = append(os.Environ(), "GOWORK=off")
	output, testErr := testCmd.CombinedOutput()
	if testErr != nil {
		_ = os.Remove(tmpFile)
		return 0, output, testErr
	}

	coverage := gocover.ExtractCoverage(tmpFile)
	_ = os.Remove(tmpFile)
	return coverage, output, nil
}
