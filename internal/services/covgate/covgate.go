package covgate

import (
	"fmt"
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
}

// Run checks per-package coverage against thresholds.
func Run(opts Opts) error {
	fmt.Printf(
		"Checking per-package coverage "+
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

	printHeader()

	hasFailures := false
	for _, pkg := range pkgs {
		ok := checkPackage(
			pkg, module, opts.SrcPrefix,
			opts.TestDir, opts.DefaultThreshold,
		)
		if !ok {
			hasFailures = true
		}
	}

	fmt.Println()
	if hasFailures {
		fmt.Println(
			"ERROR: One or more packages failed " +
				"tests or are below their minimum coverage",
		)
		return fmt.Errorf("coverage gate failed")
	}

	fmt.Println("All packages meet minimum coverage requirement")
	return nil
}

func printHeader() {
	fmt.Printf("%-6s  %8s  %8s  %s\n", "STATUS", "COVERAGE", "REQUIRED", "PACKAGE")
	fmt.Printf("%-6s  %8s  %8s  %s\n", "------", "--------", "--------", "-------")
}

func checkPackage(
	pkg, module, srcPrefix, testDir string,
	defaultThreshold float64,
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
		fmt.Printf(
			"%-6s  %8s  %8s  %s\n",
			"FAIL", "---", "---",
			relPkg+" (tests failed)",
		)
		fmt.Println()
		fmt.Print(string(output))
		fmt.Println()
		_ = os.Remove(tmpFile)
		return false
	}

	coverage := gocover.ExtractCoverage(tmpFile)
	_ = os.Remove(tmpFile)

	status := "PASS"
	if coverage < threshold {
		status = "FAIL"
	}
	fmt.Printf("%-6s  %7.1f%%  %7.1f%%  %s\n", status, coverage, threshold, relPkg)
	return coverage >= threshold
}
