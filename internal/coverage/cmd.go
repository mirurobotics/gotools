package coverage

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// NewCommand returns a cobra command that generates coverage reports.
func NewCommand() *cobra.Command {
	var (
		srcPrefix  string
		testPrefix string
		noHTML     bool
	)

	cmd := &cobra.Command{
		Use:   "coverage [sub-package]",
		Short: "Generate Go coverage report",
		Long:  "Runs tests with coverage instrumentation and generates a coverage report.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			var subPkg string
			if len(args) > 0 {
				subPkg = args[0]
			}
			return run(subPkg, srcPrefix, testPrefix, noHTML)
		},
	}

	cmd.Flags().StringVar(&srcPrefix, "src-prefix", "./pkg", "source package prefix")
	cmd.Flags().StringVar(&testPrefix, "test-prefix", "./tests", "external test prefix")
	cmd.Flags().BoolVar(&noHTML, "no-html", false, "skip opening HTML report")

	return cmd
}

func run(subPkg, srcPrefix, testPrefix string, noHTML bool) error {
	coverFile := "coverage.out"
	os.Remove(coverFile)

	var testArgs []string

	if subPkg == "" {
		// Cover all packages.
		testArgs = []string{
			"test",
			"-coverprofile=" + coverFile,
			"-coverpkg=" + srcPrefix + "/...",
			srcPrefix + "/...",
			testPrefix + "/...",
		}
	} else {
		// Cover a specific sub-package.
		extDir := testPrefix + "/" + subPkg
		if info, err := os.Stat(extDir); err == nil && info.IsDir() {
			testArgs = []string{
				"test",
				"-coverprofile=" + coverFile,
				"-coverpkg=" + srcPrefix + "/" + subPkg,
				srcPrefix + "/" + subPkg,
				testPrefix + "/" + subPkg,
			}
		} else {
			testArgs = []string{
				"test",
				"-coverprofile=" + coverFile,
				"-coverpkg=" + srcPrefix + "/" + subPkg,
				srcPrefix + "/" + subPkg,
			}
		}
	}

	testCmd := exec.Command("go", testArgs...)
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	if err := testCmd.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}

	// Print coverage summary.
	funcCmd := exec.Command("go", "tool", "cover", "-func="+coverFile)
	funcCmd.Stderr = os.Stderr
	out, err := funcCmd.Output()
	if err != nil {
		return fmt.Errorf("go tool cover -func: %w", err)
	}
	// Print only the total line, matching the shell script's `grep total`.
	for _, line := range splitLines(string(out)) {
		if len(line) >= 5 && line[:5] == "total" {
			fmt.Println(line)
		}
	}

	if !noHTML {
		htmlCmd := exec.Command("go", "tool", "cover", "-html="+coverFile)
		htmlCmd.Stdout = os.Stdout
		htmlCmd.Stderr = os.Stderr
		if err := htmlCmd.Run(); err != nil {
			return fmt.Errorf("go tool cover -html: %w", err)
		}
	}

	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
