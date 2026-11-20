package gocover

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/mirurobotics/tools/go/internal/services/cmdutil"
)

// Parallelism returns n when positive, otherwise the number of CPUs.
func Parallelism(n int) int {
	if n > 0 {
		return n
	}
	return runtime.NumCPU()
}

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

// GoListTestPackages lists packages matching pattern that contain tests.
func GoListTestPackages(pattern string) ([]string, error) {
	out, err := ExecOutput(
		nil, "go", "list", "-f",
		"{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}",
		pattern,
	)
	if err != nil {
		return nil, fmt.Errorf("go list %s: %w", pattern, err)
	}
	return NonEmptyLines(out), nil
}

// GetThreshold reads the .covgate threshold, falling back to the default.
func GetThreshold(pkgDir string, defaultThreshold float64) float64 {
	if v, ok := LookupThreshold(pkgDir); ok {
		return v
	}
	return defaultThreshold
}

// LookupThreshold reads the coverage threshold from a .covgate file.
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

// BuildTestPaths returns the test package paths for a source package.
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

// Measure runs tests for pkg with coverage and returns the percentage and output.
func Measure(pkg string, testPaths []string) (float64, []byte, error) {
	return MeasureWithEnv(pkg, testPaths, nil)
}

// RunTests runs go test on the given paths without coverage instrumentation.
func RunTests(paths []string) ([]byte, error) {
	args := make([]string, 0, 1+len(paths))
	args = append(args, "test")
	args = append(args, paths...)
	return cmdutil.GoCommand(args...).CombinedOutput()
}

// MeasureWithEnv is Measure with extraEnv appended to the test environment.
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

// ExtractCoverage parses the total coverage percentage from a Go coverage profile.
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

// ParseCoverageOutput extracts total coverage from "go tool cover -func" output.
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

// ExecOutput runs a command with GOWORK=off and returns its stdout.
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

// NonEmptyLines splits a string into non-empty, trimmed lines.
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
