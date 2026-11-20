package covgate

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/mirurobotics/gotools/internal/services/gocover"
	"github.com/mirurobotics/gotools/internal/testutil"
)

func TestTable_Header(t *testing.T) {
	var buf bytes.Buffer
	gocover.Progress{Enabled: false, Table: table()}.WriteHeader(&buf, 0, 0)
	out := buf.String()

	cols := []string{"STATUS", "COVERAGE", "REQUIRED", "TIME", "PACKAGE"}
	for _, col := range cols {
		if !strings.Contains(out, col) {
			t.Errorf("output missing column %q", col)
		}
	}
	if !strings.Contains(out, "------") {
		t.Error("output missing separator line")
	}
}

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0.0s"},
		{500 * time.Millisecond, "0.5s"},
		{3200 * time.Millisecond, "3.2s"},
		{60 * time.Second, "1m00s"},
		{7*time.Minute + 45*time.Second, "7m45s"},
	}
	for _, tc := range tests {
		if got := fmtDuration(tc.d); got != tc.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestRun_OutputContainsTiming(t *testing.T) {
	testutil.MakePkgDir(t, "pkg/a")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	job := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{"example.com/mod/pkg/a"}, nil
		},
		measure:  fakeMeasure(90.0),
		progress: gocover.Progress{Enabled: false, Table: table()},
	}

	//nolint:exhaustruct // test uses partial initialization
	err := job.run(Opts{Out: &buf, DefaultThreshold: 80.0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "TIME") {
		t.Errorf("output missing TIME column header: %s", output)
	}
	if !strings.Contains(output, "Total time:") {
		t.Errorf("output missing total time: %s", output)
	}
}

func TestPrintResults_UsesWallTime(t *testing.T) {
	results := []checkResult{
		{output: "line1\n", passed: true, duration: 5 * time.Second},
		{output: "line2\n", passed: true, duration: 5 * time.Second},
		{output: "line3\n", passed: true, duration: 5 * time.Second},
	}

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	run := runner{}
	err := run.printResults(&buf, results, nil, modName, 3*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Total time: 3.0s") {
		t.Errorf("expected wall-clock total 3.0s, got: %s", out)
	}
	if strings.Contains(out, "15.0s") {
		t.Errorf("total should not be sum of durations (15.0s): %s", out)
	}
}

func TestCheckPackage_OutputContainsTime(t *testing.T) {
	testutil.MakePkgDir(t, pkgRel)

	//nolint:exhaustruct // test uses partial initialization
	run := runner{measure: fakeMeasure(85.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := run.checkPackage(pkgName, checkPackageCtx{module: modName, threshold: 80.0})
	if !strings.Contains(res.output, "0.0s") {
		t.Errorf("output missing duration: %s", res.output)
	}
	if res.duration < 0 {
		t.Errorf("expected non-negative duration, got %v", res.duration)
	}
}
