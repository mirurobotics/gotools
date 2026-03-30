package covratchet

import (
	"fmt"
	"io"
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
	Out       io.Writer
}

// Run ratchets up .covgate thresholds.
func Run(opts Opts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	w := opts.Out

	_, _ = fmt.Fprintln(w, "Updating .covgate files (ratchet up only)...")
	_, _ = fmt.Fprintln(w)

	module, err := gocover.GoModule()
	if err != nil {
		return err
	}

	pkgs, err := gocover.GoListPackages(opts.Packages)
	if err != nil {
		return err
	}

	printHeader(w)

	updated := 0
	unchanged := 0
	failed := 0

	for _, pkg := range pkgs {
		u, unch, f := ratchetPackage(pkg, module, opts.SrcPrefix, opts.TestDir, w)
		updated += u
		unchanged += unch
		failed += f
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

func ratchetPackage(
	pkg, module, srcPrefix, testDir string, w io.Writer,
) (updated, unchanged, failed int) {
	relPkg := gocover.RelPkg(pkg, module)
	pkgDir := "./" + relPkg
	covgateFile := pkgDir + "/.covgate"

	current := readCovgate(covgateFile)
	testPaths := gocover.BuildTestPaths(pkg, relPkg, srcPrefix, testDir)
	actual, err := measureCoverage(pkg, testPaths)
	if err != nil {
		_, _ = fmt.Fprintf(
			w, "%-6s  %8s  %8s  %s\n",
			"FAIL", "\u2014", "\u2014", relPkg,
		)
		return 0, 0, 1
	}

	if current == "" {
		if err := writeCovgate(covgateFile, actual); err != nil {
			_, _ = fmt.Fprintf(
				w, "%-6s  %8s  %7.1f%%  %s (%v)\n",
				"FAIL", "\u2014", actual, relPkg, err,
			)
			return 0, 0, 1
		}
		_, _ = fmt.Fprintf(
			w, "%-6s  %8s  %7.1f%%  %s\n",
			"NEW", "\u2014", actual, relPkg,
		)
		return 1, 0, 0
	}

	if current == "0" {
		_, _ = fmt.Fprintf(w, "%-6s  %8s  %8s  %s\n", "SKIP", "0%", "\u2014", relPkg)
		return 0, 1, 0
	}

	currentVal, _ := strconv.ParseFloat(current, 64)
	if actual > currentVal {
		if err := writeCovgate(covgateFile, actual); err != nil {
			_, _ = fmt.Fprintf(
				w, "%-6s  %7s%%  %7.1f%%  %s (%v)\n",
				"FAIL", current, actual, relPkg, err,
			)
			return 0, 0, 1
		}
		_, _ = fmt.Fprintf(
			w, "%-6s  %7s%%  %7.1f%%  %s\n",
			"UP", current, actual, relPkg,
		)
		return 1, 0, 0
	}

	_, _ = fmt.Fprintf(w, "%-6s  %7s%%  %7.1f%%  %s\n", "OK", current, actual, relPkg)
	return 0, 1, 0
}

func measureCoverage(pkg string, testPaths []string) (float64, error) {
	tmpFile := "coverage.out"
	args := []string{"test", "-coverprofile=" + tmpFile, "-coverpkg=" + pkg}
	args = append(args, testPaths...)

	testCmd := exec.Command("go", args...)
	testCmd.Env = append(os.Environ(), "GOWORK=off")
	testCmd.Stdout = nil
	testCmd.Stderr = nil
	if err := testCmd.Run(); err != nil {
		_ = os.Remove(tmpFile)
		return 0, fmt.Errorf("tests failed for %s: %w", pkg, err)
	}

	actual := gocover.ExtractCoverage(tmpFile)
	_ = os.Remove(tmpFile)
	return actual, nil
}

func readCovgate(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(data), "\n")[0])
}

func writeCovgate(path string, coverage float64) error {
	content := fmt.Sprintf("%.1f\n", coverage)
	return os.WriteFile(path, []byte(content), 0o644)
}
