package coverage

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Opts holds the options for the coverage service.
type Opts struct {
	SubPkg     string
	SrcPrefix  string
	TestPrefix string
	NoHTML     bool
}

// Run generates a coverage report.
func Run(opts Opts) error {
	coverFile := "coverage.out"
	_ = os.Remove(coverFile)

	testArgs := buildTestArgs(opts.SubPkg, opts.SrcPrefix, opts.TestPrefix, coverFile)

	testCmd := exec.Command("go", testArgs...)
	testCmd.Env = append(os.Environ(), "GOWORK=off")
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	if err := testCmd.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}

	if err := printCoverageSummary(coverFile); err != nil {
		return err
	}

	if !opts.NoHTML {
		return openHTMLReport(coverFile)
	}
	return nil
}

func buildTestArgs(subPkg, srcPrefix, testPrefix, coverFile string) []string {
	if subPkg == "" {
		return []string{
			"test",
			"-coverprofile=" + coverFile,
			"-coverpkg=" + srcPrefix + "/...",
			srcPrefix + "/...",
			testPrefix + "/...",
		}
	}

	extDir := testPrefix + "/" + subPkg
	args := []string{
		"test",
		"-coverprofile=" + coverFile,
		"-coverpkg=" + srcPrefix + "/" + subPkg,
		srcPrefix + "/" + subPkg,
	}

	if info, err := os.Stat(extDir); err == nil && info.IsDir() {
		args = append(args, testPrefix+"/"+subPkg)
	}
	return args
}

func printCoverageSummary(coverFile string) error {
	funcCmd := exec.Command("go", "tool", "cover", "-func="+coverFile)
	funcCmd.Stderr = os.Stderr
	out, err := funcCmd.Output()
	if err != nil {
		return fmt.Errorf("go tool cover -func: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "total:") {
			fmt.Println(line)
		}
	}
	return nil
}

func openHTMLReport(coverFile string) error {
	htmlCmd := exec.Command("go", "tool", "cover", "-html="+coverFile)
	htmlCmd.Stdout = os.Stdout
	htmlCmd.Stderr = os.Stderr
	if err := htmlCmd.Run(); err != nil {
		return fmt.Errorf("go tool cover -html: %w", err)
	}
	return nil
}
