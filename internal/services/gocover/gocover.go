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

// GetThreshold reads the coverage threshold from a
// .covgate file, falling back to the default.
func GetThreshold(pkgDir string, defaultThreshold float64) float64 {
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

// ExtractCoverage parses the total coverage percentage
// from a Go coverage profile.
func ExtractCoverage(coverFile string) float64 {
	if _, err := os.Stat(coverFile); err != nil {
		return 0.0
	}
	out, err := ExecOutput(nil, "go", "tool", "cover", "-func="+coverFile)
	if err != nil {
		return 0.0
	}
	return ParseCoverageOutput(out)
}

// ParseCoverageOutput extracts the total coverage
// percentage from "go tool cover -func" output.
func ParseCoverageOutput(text string) float64 {
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
			return val
		}
	}
	return 0.0
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

// ExecOutput runs a command with GOWORK=off and returns
// its stdout.
func ExecOutput(errW io.Writer, name string, args ...string) (string, error) {
	if errW == nil {
		errW = os.Stderr
	}
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
