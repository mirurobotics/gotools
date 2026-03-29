package covratchet

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

// NewCommand returns a cobra command that ratchets up .covgate thresholds.
func NewCommand() *cobra.Command {
	var (
		packages  string
		srcPrefix string
		testDir   string
	)

	cmd := &cobra.Command{
		Use:   "covratchet",
		Short: "Ratchet up per-package coverage thresholds",
		Long:  "Discovers .covgate files, runs tests with coverage instrumentation, and ratchets thresholds up when actual coverage exceeds the current threshold.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(packages, srcPrefix, testDir)
		},
	}

	cmd.Flags().StringVar(&packages, "packages", "./...", "Go package pattern for go list")
	cmd.Flags().StringVar(&srcPrefix, "src-prefix", "pkg", "source prefix for mapping external test directories")
	cmd.Flags().StringVar(&testDir, "test-dir", "", "external test directory relative to repo root (empty = none)")

	return cmd
}

func run(packages, srcPrefix, testDir string) error {
	fmt.Println("Updating .covgate files (ratchet up only)...")
	fmt.Println()

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

	fmt.Printf("%-6s  %8s  %8s  %s\n", "STATUS", "PREVIOUS", "CURRENT", "PACKAGE")
	fmt.Printf("%-6s  %8s  %8s  %s\n", "------", "--------", "-------", "-------")

	updated := 0
	unchanged := 0

	for _, pkg := range pkgs {
		relPkg := strings.TrimPrefix(pkg, module+"/")
		if relPkg == pkg {
			relPkg = "."
		}
		pkgDir := "./" + relPkg
		covgateFile := filepath.Join(pkgDir, ".covgate")

		// Get current threshold.
		current := readCovgate(covgateFile)

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

		// Run tests with coverage (suppress output).
		tmpFile := "coverage.out"
		args := []string{"test", "-coverprofile=" + tmpFile, "-coverpkg=" + pkg}
		args = append(args, testPaths...)

		testCmd := exec.Command("go", args...)
		testCmd.Env = append(os.Environ())
		testCmd.Stdout = nil
		testCmd.Stderr = nil
		_ = testCmd.Run() // continue even on test failure

		// Extract actual coverage.
		actual := extractCoverage(tmpFile)
		os.Remove(tmpFile)

		// If no .covgate file exists, create one.
		if current == "" {
			writeCovgate(covgateFile, actual)
			fmt.Printf("%-6s  %8s  %7.1f%%  %s\n", "NEW", "\u2014", actual, relPkg)
			updated++
			continue
		}

		// Skip packages with threshold 0 (opted out).
		if current == "0" {
			fmt.Printf("%-6s  %8s  %8s  %s\n", "SKIP", "0%", "\u2014", relPkg)
			unchanged++
			continue
		}

		currentVal, _ := strconv.ParseFloat(current, 64)

		// Ratchet up only.
		if actual > currentVal {
			writeCovgate(covgateFile, actual)
			fmt.Printf("%-6s  %7s%%  %7.1f%%  %s\n", "UP", current, actual, relPkg)
			updated++
		} else {
			fmt.Printf("%-6s  %7s%%  %7.1f%%  %s\n", "OK", current, actual, relPkg)
			unchanged++
		}
	}

	fmt.Println()
	fmt.Printf("Done. Updated: %d, Unchanged: %d\n", updated, unchanged)
	return nil
}

func readCovgate(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(strings.Split(string(data), "\n")[0])
	return line
}

func writeCovgate(path string, coverage float64) {
	content := fmt.Sprintf("%.1f\n", coverage)
	_ = os.WriteFile(path, []byte(content), 0o644)
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
