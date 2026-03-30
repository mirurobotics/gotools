package covratchet

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/gocover"
)

// Opts holds the options for the covratchet service.
type Opts struct {
	Packages  string
	SrcPrefix string
	TestDir   string
}

// Run ratchets up .covgate thresholds.
func Run(opts Opts) error {
	fmt.Println("Updating .covgate files (ratchet up only)...")
	fmt.Println()

	module, err := gocover.GoModule()
	if err != nil {
		return err
	}

	pkgs, err := gocover.GoListPackages(opts.Packages)
	if err != nil {
		return err
	}

	printHeader()

	updated := 0
	unchanged := 0

	for _, pkg := range pkgs {
		u, unch := ratchetPackage(pkg, module, opts.SrcPrefix, opts.TestDir)
		updated += u
		unchanged += unch
	}

	fmt.Println()
	fmt.Printf("Done. Updated: %d, Unchanged: %d\n", updated, unchanged)
	return nil
}

func printHeader() {
	fmt.Printf("%-6s  %8s  %8s  %s\n", "STATUS", "PREVIOUS", "CURRENT", "PACKAGE")
	fmt.Printf("%-6s  %8s  %8s  %s\n", "------", "--------", "-------", "-------")
}

func ratchetPackage(pkg, module, srcPrefix, testDir string) (updated, unchanged int) {
	relPkg := gocover.RelPkg(pkg, module)
	pkgDir := "./" + relPkg
	covgateFile := pkgDir + "/.covgate"

	current := readCovgate(covgateFile)
	testPaths := gocover.BuildTestPaths(pkg, relPkg, srcPrefix, testDir)
	actual := measureCoverage(pkg, testPaths)

	if current == "" {
		writeCovgate(covgateFile, actual)
		fmt.Printf("%-6s  %8s  %7.1f%%  %s\n", "NEW", "\u2014", actual, relPkg)
		return 1, 0
	}

	if current == "0" {
		fmt.Printf("%-6s  %8s  %8s  %s\n", "SKIP", "0%", "\u2014", relPkg)
		return 0, 1
	}

	currentVal, _ := strconv.ParseFloat(current, 64)
	if actual > currentVal {
		writeCovgate(covgateFile, actual)
		fmt.Printf("%-6s  %7s%%  %7.1f%%  %s\n", "UP", current, actual, relPkg)
		return 1, 0
	}

	fmt.Printf("%-6s  %7s%%  %7.1f%%  %s\n", "OK", current, actual, relPkg)
	return 0, 1
}

func measureCoverage(pkg string, testPaths []string) float64 {
	tmpFile := "coverage.out"
	args := []string{"test", "-coverprofile=" + tmpFile, "-coverpkg=" + pkg}
	args = append(args, testPaths...)

	testCmd := exec.Command("go", args...)
	testCmd.Env = append(os.Environ(), "GOWORK=off")
	testCmd.Stdout = nil
	testCmd.Stderr = nil
	_ = testCmd.Run()

	actual := gocover.ExtractCoverage(tmpFile)
	_ = os.Remove(tmpFile)
	return actual
}

func readCovgate(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(data), "\n")[0])
}

func writeCovgate(path string, coverage float64) {
	content := fmt.Sprintf("%.1f\n", coverage)
	_ = os.WriteFile(path, []byte(content), 0o644)
}
