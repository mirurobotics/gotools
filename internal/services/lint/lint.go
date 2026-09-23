package lint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mirurobotics/gotools/internal/services/cmdutil"
	"github.com/mirurobotics/gotools/internal/services/lint/linter"
)

type stepTiming struct {
	name     string
	duration time.Duration
}

// LintOpts holds the options for the lint orchestrator.
type LintOpts struct {
	Paths string
	DoFix bool
	LinterFlags
	Deadcode        bool
	DeadcodeExclude string
	NoGofumpt       bool
	NoGolangci      bool
	NewFromRev      string
	// GOOS lists extra target platforms, comma-separated;
	// golangci-lint runs once more per target.
	GOOS string
	Out  io.Writer
	Err  io.Writer
}

// RunLint runs the full lint suite: custom linter,
// gofumpt, golangci-lint, and deadcode, plus
// golangci-lint for each extra GOOS target.
func RunLint(opts LintOpts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}

	totalStart := time.Now()
	failures, timings, fatal := runLintSteps(opts)
	printTimings(opts.Out, timings, time.Since(totalStart))

	if fatal != nil {
		return fatal
	}
	if len(failures) > 0 {
		return fmt.Errorf("lint failed: %s", strings.Join(failures, ", "))
	}

	_, _ = fmt.Fprintln(opts.Out, "Lint complete")
	return nil
}

func runLintSteps(
	opts LintOpts,
) (failures []string, timings []stepTiming, fatal error) {
	if opts.Paths != "" {
		start := time.Now()
		hadIssues, err := runCustomLinter(opts)
		timings = append(timings, stepTiming{"custom linter", time.Since(start)})
		if err != nil {
			return failures, timings, err
		}
		if hadIssues {
			failures = append(failures, "custom linter")
		}
	}

	if !opts.NoGofumpt {
		start := time.Now()
		err := RunGofumpt(opts.Out, opts.Err, opts.DoFix)
		timings = append(timings, stepTiming{"gofumpt", time.Since(start)})
		if err != nil {
			return failures, timings, fmt.Errorf("gofumpt: %w", err)
		}
	}

	f, t := runAnalyzers(opts)
	failures = append(failures, f...)
	timings = append(timings, t...)

	if !opts.NoGolangci {
		f, t = runGolangciTargets(opts)
		failures = append(failures, f...)
		timings = append(timings, t...)
	}
	return failures, timings, nil
}

func runGolangciTargets(opts LintOpts) (failures []string, timings []stepTiming) {
	for _, goos := range goosTargets(opts.GOOS) {
		step := fmt.Sprintf("golangci-lint (%s)", goos)
		start := time.Now()
		err := RunGolangciGOOS(opts.Out, opts.Err, opts.NewFromRev, goos)
		if err != nil {
			failures = append(failures, step)
		}
		timings = append(timings, stepTiming{step, time.Since(start)})
	}
	return failures, timings
}

// goosTargets splits the comma-separated raw into target
// platforms in first-seen order, dropping blanks,
// duplicates, and the host GOOS.
func goosTargets(raw string) []string {
	var targets []string
	seen := map[string]bool{runtime.GOOS: true}
	for _, goos := range strings.Split(raw, ",") {
		goos = strings.TrimSpace(goos)
		if goos == "" || seen[goos] {
			continue
		}
		seen[goos] = true
		targets = append(targets, goos)
	}
	return targets
}

// runAnalyzers runs golangci-lint and deadcode. When both
// are enabled they run concurrently; deadcode output is
// buffered to avoid interleaving with golangci-lint.
func runAnalyzers(opts LintOpts) (failures []string, timings []stepTiming) {
	runGolangci := !opts.NoGolangci
	runDeadcode := opts.Deadcode

	if runGolangci && runDeadcode {
		return runAnalyzersParallel(opts)
	}
	if runGolangci {
		start := time.Now()
		if err := RunGolangci(opts.Out, opts.Err, opts.NewFromRev); err != nil {
			failures = append(failures, "golangci-lint")
		}
		timings = append(timings, stepTiming{"golangci-lint", time.Since(start)})
	}
	if runDeadcode {
		start := time.Now()
		if err := RunDeadcode(opts.Out, opts.Err, opts.DeadcodeExclude); err != nil {
			failures = append(failures, "deadcode")
		}
		timings = append(timings, stepTiming{"deadcode", time.Since(start)})
	}
	return failures, timings
}

func runAnalyzersParallel(opts LintOpts) (failures []string, timings []stepTiming) {
	type result struct {
		timing stepTiming
		failed bool
		outBuf string
	}

	var (
		wg  sync.WaitGroup
		gcR result
		dcR result
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		start := time.Now()
		err := RunGolangci(opts.Out, opts.Err, opts.NewFromRev)
		gcR = result{
			timing: stepTiming{"golangci-lint", time.Since(start)},
			failed: err != nil,
			outBuf: "",
		}
	}()
	go func() {
		defer wg.Done()
		var buf bytes.Buffer
		start := time.Now()
		err := RunDeadcode(&buf, opts.Err, opts.DeadcodeExclude)
		dcR = result{
			timing: stepTiming{"deadcode", time.Since(start)},
			failed: err != nil,
			outBuf: buf.String(),
		}
	}()
	wg.Wait()

	if dcR.outBuf != "" {
		_, _ = fmt.Fprint(opts.Out, dcR.outBuf)
	}
	timings = append(timings, gcR.timing, dcR.timing)
	if gcR.failed {
		failures = append(failures, "golangci-lint")
	}
	if dcR.failed {
		failures = append(failures, "deadcode")
	}
	return failures, timings
}

func printTimings(w io.Writer, timings []stepTiming, total time.Duration) {
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Timings")
	_, _ = fmt.Fprintln(w, "-------")
	width := 16
	for _, t := range timings {
		width = max(width, len(t.name))
	}
	for _, t := range timings {
		_, _ = fmt.Fprintf(w, "%-*s %7s\n", width, t.name, fmtDuration(t.duration))
	}
	_, _ = fmt.Fprintf(w, "%-*s %7s\n", width, "total", fmtDuration(total))
}

func fmtDuration(d time.Duration) string {
	d = d.Round(100 * time.Millisecond)
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := d - time.Duration(m)*time.Minute
	return fmt.Sprintf("%dm%02.0fs", m, s.Seconds())
}

func runCustomLinter(opts LintOpts) (bool, error) {
	cfg, err := BuildLinterConfig(opts.LinterFlags)
	if err != nil {
		return false, err
	}
	cfg.Out = opts.Out
	cfg.Err = opts.Err

	totalDiags := 0
	for _, p := range strings.Split(opts.Paths, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, _ = fmt.Fprintf(opts.Out, "Running custom linter on %s...\n", p)
		diags, fixed, runErr := linter.Run(p, opts.DoFix, cfg)
		if runErr != nil {
			return false, fmt.Errorf("custom linter on %s: %w", p, runErr)
		}
		if opts.DoFix && fixed > 0 {
			_, _ = fmt.Fprintf(opts.Out, "%d file(s) fixed in %s.\n", fixed, p)
		}
		if diags > 0 {
			_, _ = fmt.Fprintf(opts.Out, "%d violation(s) found in %s.\n", diags, p)
		}
		totalDiags += diags
	}
	return totalDiags > 0, nil
}

// RunGolangci runs golangci-lint. If newFromRev is
// non-empty, only new issues since that revision are
// reported.
func RunGolangci(out io.Writer, errW io.Writer, newFromRev string) error {
	_, _ = fmt.Fprintln(out, "Running golangci-lint...")
	args := append([]string{"tool", "golangci-lint"}, golangciArgs(newFromRev)...)
	if err := RunExternal(out, errW, "go", args...); err != nil {
		_, _ = fmt.Fprintf(errW, "golangci-lint failed: %v\n", err)
		return err
	}
	return nil
}

// RunGolangciGOOS runs golangci-lint with GOOS set to goos,
// linting the files that build for that target. GOARCH is
// left to the environment.
func RunGolangciGOOS(out io.Writer, errW io.Writer, newFromRev, goos string) error {
	_, _ = fmt.Fprintf(out, "Running golangci-lint for %s...\n", goos)
	// GOOS in the environment of `go tool` cross-compiles the tool
	// itself, so build the host binary and set GOOS on that.
	dir, err := os.MkdirTemp("", "miru-golangci-")
	if err != nil {
		_, _ = fmt.Fprintf(errW, "golangci-lint for %s failed: %v\n", goos, err)
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	bin, err := hostToolPath("golangci-lint", dir)
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

// hostToolPath builds the named `go tool` dependency for
// the host into dir, ignoring any inherited GOOS and GOARCH,
// and returns the path of its binary. It builds with
// `go build -o` rather than `go tool -n` because the latter
// returns a deleted temporary path when GOCACHEPROG is set.
func hostToolPath(name, dir string) (string, error) {
	pkg, err := toolPackage(name)
	if err != nil {
		return "", fmt.Errorf("resolve go tool %s: %w", name, err)
	}
	bin := filepath.Join(dir, name+exeSuffix())
	if _, err := runHostGo("build", "-o", bin, pkg); err != nil {
		return "", fmt.Errorf("build go tool %s: %w", name, err)
	}
	return bin, nil
}

// toolPackage returns the package of the go.mod tool
// directive that `go tool name` would run.
func toolPackage(name string) (string, error) {
	out, err := runHostGo("list", "-f", "{{.ImportPath}}", "tool")
	if err != nil {
		return "", err
	}
	for _, pkg := range strings.Fields(out) {
		if toolName(pkg) == name {
			return pkg, nil
		}
	}
	return "", fmt.Errorf("no tool named %s in go.mod", name)
}

// toolName mirrors cmd/go: the last path element, skipping a
// trailing major-version suffix such as /v2.
func toolName(pkg string) string {
	elem := path.Base(pkg)
	if isMajorVersion(elem) {
		return path.Base(path.Dir(pkg))
	}
	return elem
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func isMajorVersion(elem string) bool {
	n, ok := strings.CutPrefix(elem, "v")
	return ok && n != "" && strings.Trim(n, "0123456789") == ""
}

func runHostGo(args ...string) (string, error) {
	cmd := cmdutil.GoCommand(args...)
	cmd.Env = append(cmd.Env, "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w\n%s", err, stderr.String())
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

// RunDeadcode runs the deadcode checker, optionally
// filtering output.
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

// FilterDeadcodeOutput filters deadcode output, removing
// module paths and optional exclude patterns.
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

// RunExternal runs an external Go toolchain command,
// inheriting stdout/stderr from the provided writers.
// The first arg is expected to be the go subcommand.
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
