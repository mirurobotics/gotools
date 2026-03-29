package covgate

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// NewCommand returns a cobra command that checks per-package coverage against thresholds.
func NewCommand() *cobra.Command {
	var (
		packages         string
		srcPrefix        string
		testDir          string
		defaultThreshold float64
	)

	cmd := &cobra.Command{
		Use:   "covgate",
		Short: "Check per-package coverage against thresholds",
		Long:  "Discovers .covgate files, runs tests with coverage instrumentation, and checks per-package coverage against thresholds.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(packages, srcPrefix, testDir, defaultThreshold)
		},
	}

	cmd.Flags().StringVar(&packages, "packages", "./...", "Go package pattern for go list")
	cmd.Flags().StringVar(&srcPrefix, "src-prefix", "pkg", "source prefix for mapping external test directories")
	cmd.Flags().StringVar(&testDir, "test-dir", "", "external test directory relative to repo root (empty = none)")
	cmd.Flags().Float64Var(&defaultThreshold, "default-threshold", 80.0, "fallback threshold when no .covgate file exists")

	return cmd
}

func run(packages, srcPrefix, testDir string, defaultThreshold float64) error {
	fmt.Printf("Checking per-package coverage (default minimum: %.1f%%)...\n\n", defaultThreshold)

	// Get module path.
	modOut, err := execOutput("go", "list", "-m")
	if err != nil {
		return fmt.Errorf("go list -m: %w", err)
	}
	module := strings.TrimSpace(modOut)

	// List packages.
	listOut, err := execOutput("go", "list", packages)
	if err != nil {
		return fmt.Errorf("go list %s: %w", packages, err)
	}
	pkgs := nonEmptyLines(listOut)

	fmt.Printf("%-6s  %8s  %8s  %s\n", "STATUS", "COVERAGE", "REQUIRED", "PACKAGE")
	fmt.Printf("%-6s  %8s  %8s  %s\n", "------", "--------", "--------", "-------")

	hasFailures := false

	for _, pkg := range pkgs {
		relPkg := strings.TrimPrefix(pkg, module+"/")
		if relPkg == pkg {
			relPkg = "."
		}
		pkgDir := "./" + relPkg

		threshold := getThreshold(pkgDir, defaultThreshold)

		// Build test paths.
		testPaths := []string{pkg}
		if testDir != "" && srcPrefix != "" {
			testSub := strings.TrimPrefix(relPkg, srcPrefix+"/")
			if testSub != relPkg {
				extPath := filepath.Join(testDir, testSub)
				if info, err := os.Stat(extPath); err == nil && info.IsDir() {
					testPaths = append(testPaths, "./"+extPath)
				}
			}
		}

		// Run tests with coverage.
		tmpFile := "coverage.out"
		args := []string{"test", "-coverprofile=" + tmpFile, "-coverpkg=" + pkg}
		args = append(args, testPaths...)

		testCmd := exec.Command("go", args...)
		testCmd.Env = append(os.Environ())
		output, testErr := testCmd.CombinedOutput()

		if testErr != nil {
			fmt.Printf("%-6s  %8s  %8s  %s\n", "FAIL", "---", "---", relPkg+" (tests failed)")
			fmt.Println()
			fmt.Print(string(output))
			fmt.Println()
			hasFailures = true
			os.Remove(tmpFile)
			continue
		}

		// Extract coverage.
		coverage := extractCoverage(tmpFile)
		os.Remove(tmpFile)

		// Compare against threshold.
		if coverage < threshold {
			fmt.Printf("%-6s  %7.1f%%  %7.1f%%  %s\n", "FAIL", coverage, threshold, relPkg)
			hasFailures = true
		} else {
			fmt.Printf("%-6s  %7.1f%%  %7.1f%%  %s\n", "PASS", coverage, threshold, relPkg)
		}
	}

	fmt.Println()

	if hasFailures {
		fmt.Println("ERROR: One or more packages failed tests or are below their minimum coverage threshold")
		os.Exit(1)
	}

	fmt.Println("All packages meet minimum coverage requirement")
	return nil
}

func getThreshold(pkgDir string, defaultThreshold float64) float64 {
	covFile := filepath.Join(pkgDir, ".covgate")
	data, err := os.ReadFile(covFile)
	if err != nil {
		return defaultThreshold
	}
	line := strings.TrimSpace(strings.Split(string(data), "\n")[0])
	val, err := strconv.ParseFloat(line, 64)
	if err != nil {
		return defaultThreshold
	}
	return val
}

func extractCoverage(coverFile string) float64 {
	if _, err := os.Stat(coverFile); err != nil {
		return 0.0
	}
	out, err := execOutput("go", "tool", "cover", "-func="+coverFile)
	if err != nil {
		return 0.0
	}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				pct := strings.TrimSuffix(fields[len(fields)-1], "%")
				val, err := strconv.ParseFloat(pct, 64)
				if err == nil {
					return val
				}
			}
		}
	}
	return 0.0
}

func execOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func nonEmptyLines(s string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
