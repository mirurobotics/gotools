package lint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

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
	targets := goosTargets(opts.GOOS)
	results := make([]stepResult, 0, len(targets))
	for _, goos := range targets {
		step := fmt.Sprintf("golangci-lint (%s)", goos)
		results = append(results, timedRun(step, func() error {
			return RunGolangciGOOS(opts.Out, opts.Err, opts.NewFromRev, goos)
		}))
	}
	return collectResults(results)
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
	var results []stepResult
	if runGolangci {
		results = append(results, timedRun("golangci-lint", func() error {
			return RunGolangci(opts.Out, opts.Err, opts.NewFromRev)
		}))
	}
	if runDeadcode {
		results = append(results, timedRun("deadcode", func() error {
			return RunDeadcode(opts.Out, opts.Err, opts.DeadcodeExclude)
		}))
	}
	return collectResults(results)
}

// stepResult is the outcome of one timed pass/fail lint step.
type stepResult struct {
	timing stepTiming
	failed bool
}

// timedRun runs fn and records its duration under name.
func timedRun(name string, fn func() error) stepResult {
	start := time.Now()
	err := fn()
	return stepResult{timing: stepTiming{name, time.Since(start)}, failed: err != nil}
}

func runAnalyzersParallel(opts LintOpts) (failures []string, timings []stepTiming) {
	var (
		wg    sync.WaitGroup
		gcR   stepResult
		dcR   stepResult
		dcBuf bytes.Buffer
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		gcR = timedRun("golangci-lint", func() error {
			return RunGolangci(opts.Out, opts.Err, opts.NewFromRev)
		})
	}()
	go func() {
		defer wg.Done()
		dcR = timedRun("deadcode", func() error {
			return RunDeadcode(&dcBuf, opts.Err, opts.DeadcodeExclude)
		})
	}()
	wg.Wait()

	if dcBuf.Len() > 0 {
		_, _ = fmt.Fprint(opts.Out, dcBuf.String())
	}
	return collectResults([]stepResult{gcR, dcR})
}

// collectResults returns the timings of all steps, in order, and the names
// of the steps that failed.
func collectResults(results []stepResult) (failures []string, timings []stepTiming) {
	for _, r := range results {
		timings = append(timings, r.timing)
		if r.failed {
			failures = append(failures, r.timing.name)
		}
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
