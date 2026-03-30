package coverage

import (
	"fmt"
	"io"
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
	Out        io.Writer
	Err        io.Writer
}

// Run generates a coverage report.
func Run(opts Opts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}

	coverFile := "coverage.out"
	_ = os.Remove(coverFile)

	testArgs := buildTestArgs(opts.SubPkg, opts.SrcPrefix, opts.TestPrefix, coverFile)

	testCmd := exec.Command("go", testArgs...)
	testCmd.Env = append(os.Environ(), "GOWORK=off")
	testCmd.Stdout = opts.Out
	testCmd.Stderr = opts.Err
	if err := testCmd.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}

	if err := printCoverageSummary(coverFile, opts.Out, opts.Err); err != nil {
		return err
	}

	if !opts.NoHTML {
		return openHTMLReport(coverFile, opts.Out, opts.Err)
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

func printCoverageSummary(coverFile string, out io.Writer, errW io.Writer) error {
	funcCmd := exec.Command("go", "tool", "cover", "-func="+coverFile)
	funcCmd.Stderr = errW
	cmdOut, err := funcCmd.Output()
	if err != nil {
		return fmt.Errorf("go tool cover -func: %w", err)
	}
	for _, line := range strings.Split(string(cmdOut), "\n") {
		if strings.HasPrefix(line, "total:") {
			_, _ = fmt.Fprintln(out, line)
		}
	}
	return nil
}

func openHTMLReport(coverFile string, out io.Writer, errW io.Writer) error {
	htmlCmd := exec.Command("go", "tool", "cover", "-html="+coverFile)
	htmlCmd.Stdout = out
	htmlCmd.Stderr = errW
	if err := htmlCmd.Run(); err != nil {
		return fmt.Errorf("go tool cover -html: %w", err)
	}
	return nil
}
