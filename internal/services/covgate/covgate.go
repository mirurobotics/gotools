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

// Run checks per-package coverage against thresholds.
func Run(opts Opts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	w := opts.Out

	_, _ = fmt.Fprintf(
		w, "Checking per-package coverage "+
			"(default minimum: %.1f%%)...\n\n",
		opts.DefaultThreshold,
	)

	module, err := gocover.GoModule()
	if err != nil {
		return err
	}

	pkgs, err := gocover.GoListPackages(opts.Packages)
	if err != nil {
		return err
	}

	printHeader(w)

	hasFailures := false
	for _, pkg := range pkgs {
		ok := checkPackage(
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

func checkPackage(
	pkg, module, srcPrefix, testDir string,
	defaultThreshold float64, w io.Writer,
) bool {
	relPkg := gocover.RelPkg(pkg, module)
	pkgDir := "./" + relPkg
	threshold := gocover.GetThreshold(pkgDir, defaultThreshold)

	testPaths := gocover.BuildTestPaths(pkg, relPkg, srcPrefix, testDir)

	tmpFile := "coverage.out"
	args := []string{"test", "-coverprofile=" + tmpFile, "-coverpkg=" + pkg}
	args = append(args, testPaths...)

	testCmd := exec.Command("go", args...)
	testCmd.Env = append(os.Environ(), "GOWORK=off")
	output, testErr := testCmd.CombinedOutput()

	if testErr != nil {
		_, _ = fmt.Fprintf(
			w, "%-6s  %8s  %8s  %s\n",
			"FAIL", "---", "---",
			relPkg+" (tests failed)",
		)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprint(w, string(output))
		_, _ = fmt.Fprintln(w)
		_ = os.Remove(tmpFile)
		return false
	}

	coverage := gocover.ExtractCoverage(tmpFile)
	_ = os.Remove(tmpFile)

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
