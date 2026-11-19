package lint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/cmdutil"
)

// RunGolangci runs golangci-lint.
func RunGolangci(out io.Writer, errW io.Writer, newFromRev string) error {
	_, _ = fmt.Fprintln(out, "Running golangci-lint...")
	args := append([]string{"tool", "golangci-lint"}, golangciArgs(newFromRev)...)
	if err := RunExternal(out, errW, "go", args...); err != nil {
		_, _ = fmt.Fprintf(errW, "golangci-lint failed: %v\n", err)
		return err
	}
	return nil
}

// RunGolangciGOOS runs a host-built golangci-lint with GOOS set to goos.
func RunGolangciGOOS(out io.Writer, errW io.Writer, newFromRev, goos string) error {
	_, _ = fmt.Fprintf(out, "Running golangci-lint for %s...\n", goos)
	bin, err := hostToolPath("golangci-lint")
	if err != nil {
		_, _ = fmt.Fprintf(errW, "golangci-lint for %s failed: %v\n", goos, err)
		return err
	}
	return runGolangciBin(out, errW, bin, newFromRev, goos)
}

func runGolangciBin(out, errW io.Writer, bin, newFromRev, goos string) error {
	//nolint:gosec,noctx // G204: trusted subprocess
	cmd := exec.Command(bin, golangciArgs(newFromRev)...)
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOOS="+goos)
	cmd.Stdout = out
	cmd.Stderr = errW
	if err := cmd.Run(); err != nil {
		_, _ = fmt.Fprintf(errW, "golangci-lint for %s failed: %v\n", goos, err)
		return err
	}
	return nil
}

func golangciArgs(newFromRev string) []string {
	args := []string{"run"}
	if newFromRev != "" {
		args = append(args, "--new-from-rev="+newFromRev)
	}
	return args
}

//nolint:privdoc // GOCACHEPROG= avoids a temporary tool path
func hostToolPath(name string) (string, error) {
	cmd := cmdutil.GoCommand("tool", "-n", name)
	cmd.Env = append(cmd.Env,
		"GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "GOCACHEPROG=")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("resolve go tool %s: %w\n%s", name, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunGofumpt runs gofumpt in fix or check mode.
func RunGofumpt(out io.Writer, errW io.Writer, fix bool) error {
	if fix {
		_, _ = fmt.Fprintln(out, "Running gofumpt...")
		return RunExternal(out, errW, "go", "tool", "gofumpt", "-w", ".")
	}

	_, _ = fmt.Fprintln(out, "Checking gofumpt...")
	cmd := cmdutil.GoCommand("tool", "gofumpt", "-l", ".")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gofumpt failed: %w\n%s", err, stderr.String())
	}
	trimmed := strings.TrimSpace(stdout.String())
	if trimmed != "" {
		_, _ = fmt.Fprintln(out, "Files need formatting:")
		_, _ = fmt.Fprintln(out, trimmed)
		return fmt.Errorf("gofumpt found unformatted files")
	}
	return nil
}

// RunDeadcode runs the deadcode checker, optionally filtering output.
func RunDeadcode(out io.Writer, errW io.Writer, excludePattern string) error {
	_, _ = fmt.Fprintln(out, "Running deadcode...")
	cmd := cmdutil.GoCommand("tool", "deadcode", "-test", "./...")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	filtered := FilterDeadcodeOutput(stdout.String(), excludePattern)
	if len(filtered) > 0 {
		for _, line := range filtered {
			_, _ = fmt.Fprintln(out, line)
		}
		return fmt.Errorf("deadcode found issues")
	}
	if err != nil {
		_, _ = fmt.Fprintf(errW, "deadcode failed: %v\n%s", err, stderr.String())
		return fmt.Errorf("deadcode: %w", err)
	}
	return nil
}

// FilterDeadcodeOutput strips module paths and lines matching excludePattern.
func FilterDeadcodeOutput(raw, excludePattern string) []string {
	var filtered []string
	for _, line := range strings.Split(raw, "\n") {
		if strings.Contains(line, "/go/pkg/mod/") {
			continue
		}
		if line == "" {
			continue
		}
		filtered = append(filtered, line)
	}

	if excludePattern != "" {
		var kept []string
		for _, line := range filtered {
			if !strings.Contains(line, excludePattern) {
				kept = append(kept, line)
			}
		}
		filtered = kept
	}
	return filtered
}

// RunExternal runs a command, writing its stdout and stderr to out and errW.
func RunExternal(out io.Writer, errW io.Writer, name string, args ...string) error {
	var cmd *exec.Cmd
	if name == "go" {
		cmd = cmdutil.GoCommand(args...)
	} else {
		//nolint:gosec,noctx // G204: trusted subprocess
		cmd = exec.Command(name, args...)
	}
	cmd.Stdout = out
	cmd.Stderr = errW
	return cmd.Run()
}
