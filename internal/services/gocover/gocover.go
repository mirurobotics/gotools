package gocover

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/cmdutil"
)

// RelPkg computes a module-relative package path.
func RelPkg(pkg, module string) string {
	rel := strings.TrimPrefix(pkg, module+"/")
	if rel == pkg {
		return "."
	}
	return rel
}

// GoModule returns the current module name.
func GoModule() (string, error) {
	out, err := ExecOutput(nil, "go", "list", "-m")
	if err != nil {
		return "", fmt.Errorf("go list -m: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// GoListPackages lists packages matching a pattern.
func GoListPackages(pattern string) ([]string, error) {
	out, err := ExecOutput(nil, "go", "list", pattern)
	if err != nil {
		return nil, fmt.Errorf("go list %s: %w", pattern, err)
	}
	return NonEmptyLines(out), nil
}

// LookupThreshold reads the coverage threshold from a
// .covgate file. The second return value is true when an
// explicit .covgate file exists and parses successfully.
func LookupThreshold(pkgDir string) (float64, bool) {
	covFile := filepath.Join(pkgDir, ".covgate")
	//nolint:gosec // G304: trusted file path
	data, err := os.ReadFile(covFile)
	if err != nil {
		return 0, false
	}
	line := strings.TrimSpace(strings.Split(string(data), "\n")[0])
	val, err := strconv.ParseFloat(line, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

// GetThreshold reads the coverage threshold from a
// .covgate file, falling back to the default.
func GetThreshold(pkgDir string, defaultThreshold float64) float64 {
	if v, ok := LookupThreshold(pkgDir); ok {
		return v
	}
	return defaultThreshold
}

// ExtractCoverage parses the total coverage percentage
// from a Go coverage profile.
func ExtractCoverage(coverFile string) (float64, error) {
	if _, err := os.Stat(coverFile); err != nil {
		return 0, fmt.Errorf("coverage file: %w", err)
	}
	out, err := ExecOutput(nil, "go", "tool", "cover", "-func="+coverFile)
	if err != nil {
		return 0, fmt.Errorf("go tool cover: %w", err)
	}
	val, err := ParseCoverageOutput(out)
	if err != nil {
		return 0, fmt.Errorf("parse coverage output: %w", err)
	}
	return val, nil
}

// ParseCoverageOutput extracts the total coverage
// percentage from "go tool cover -func" output.
func ParseCoverageOutput(text string) (float64, error) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "total:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pct := strings.TrimSuffix(fields[len(fields)-1], "%")
		val, err := strconv.ParseFloat(pct, 64)
		if err == nil {
			return val, nil
		}
	}
	return 0, fmt.Errorf("no total coverage line found in output")
}

// BuildTestPaths returns the test package paths for a
// given source package, including any external test dir.
func BuildTestPaths(pkg, relPkg, srcPrefix, testDir string) []string {
	paths := []string{pkg}
	if testDir == "" || srcPrefix == "" {
		return paths
	}
	testSub := strings.TrimPrefix(relPkg, srcPrefix+"/")
	if testSub == relPkg {
		return paths
	}
	extPath := filepath.Join(testDir, testSub)
	info, err := os.Stat(extPath)
	if err == nil && info.IsDir() {
		paths = append(paths, "./"+extPath)
	}
	return paths
}

// Measure runs tests for pkg with coverage profiling and
// returns the coverage percentage and combined test output.
// Uses a temp file for the coverage profile, cleaned up
// automatically.
func Measure(pkg string, testPaths []string) (float64, []byte, error) {
	return MeasureWithEnv(pkg, testPaths, nil)
}

// MeasureWithEnv is like Measure but appends extraEnv to
// the test subprocess environment when non-empty.
// Entries in extraEnv override earlier definitions of the
// same key (os/exec uses last-wins).
// This lets callers inject variables such as GOMAXPROCS
// into the child go test invocation.
func MeasureWithEnv(
	pkg string, testPaths []string, extraEnv []string,
) (float64, []byte, error) {
	tmpFile, err := os.CreateTemp("", "miru-coverage-*.out")
	if err != nil {
		return 0, nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	args := make([]string, 0, 3+len(testPaths))
	args = append(args, "test", "-coverprofile="+tmpPath, "-coverpkg="+pkg)
	args = append(args, testPaths...)

	testCmd := cmdutil.GoCommand(args...)
	if len(extraEnv) > 0 {
		testCmd.Env = append(testCmd.Env, extraEnv...)
	}
	output, testErr := testCmd.CombinedOutput()
	if testErr != nil {
		return 0, output, testErr
	}

	coverage, coverErr := ExtractCoverage(tmpPath)
	if coverErr != nil {
		return 0, output, fmt.Errorf("extract coverage: %w", coverErr)
	}
	return coverage, output, nil
}

// PrewarmBuild compiles the test binaries for the given paths
// without running any tests, populating the shared Go build
// cache so that subsequent parallel `go test` runs do not
// stampede on the same shared dependency compiles. It runs two
// passes (skipping either when empty):
//
//   - coverPaths are compiled INSTRUMENTED via
//     `go test -run='^$' -cover <coverPaths...>`, which defaults
//     to instrumenting each listed package for itself — matching
//     covgate's per-package `-coverpkg=<pkg>` instrumentation, so
//     the warmed objects are reused by the coverage runs.
//   - plainPaths are compiled NON-instrumented via
//     `go test -run='^$' <plainPaths...>`; these are the
//     integration-test dirs covgate builds as plain deps of the
//     coverage binary, not coverpkg targets.
//
// Running each group in one pass lets Go's build planner
// deduplicate the shared compiles. A genuine build error is
// returned (with combined output) so callers can surface it.
func PrewarmBuild(coverPaths, plainPaths []string) error {
	if len(coverPaths) == 0 && len(plainPaths) == 0 {
		return nil
	}
	if len(coverPaths) > 0 {
		args := make([]string, 0, 3+len(coverPaths))
		args = append(args, "test", "-run=^$", "-cover")
		args = append(args, coverPaths...)
		cmd := cmdutil.GoCommand(args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("prewarm build: %w\n%s", err, out)
		}
	}
	if len(plainPaths) > 0 {
		args := make([]string, 0, 2+len(plainPaths))
		args = append(args, "test", "-run=^$")
		args = append(args, plainPaths...)
		cmd := cmdutil.GoCommand(args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("prewarm build: %w\n%s", err, out)
		}
	}
	return nil
}

// ExecOutput runs a command with GOWORK=off and returns
// its stdout.
func ExecOutput(errW io.Writer, name string, args ...string) (string, error) {
	if errW == nil {
		errW = os.Stderr
	}
	//nolint:gosec,noctx // G204: trusted subprocess
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = errW
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// NonEmptyLines splits a string into non-empty,
// trimmed lines.
func NonEmptyLines(s string) []string {
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
