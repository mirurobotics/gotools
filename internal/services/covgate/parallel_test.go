package covgate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/gocover"
	"github.com/mirurobotics/gotools/internal/testutil"
)

func TestRun_Parallelism_DefaultsToNumCPU(t *testing.T) {
	testutil.MakePkgDir(t, "pkg/a")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{modName + "/pkg/a"}, nil
		},
		measure: fakeMeasure(90.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0, Parallelism: 0})
	if err != nil {
		t.Fatalf("unexpected error with Parallelism=0 (NumCPU): %v", err)
	}
}

func hasProgressLineWith(out, marker, contains string) bool {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, marker) && strings.Contains(line, contains) {
			return true
		}
	}
	return false
}

func chdirWithPkgs(t *testing.T, rels ...string) {
	t.Helper()
	tmp := t.TempDir()
	t.Chdir(tmp)
	for _, rel := range rels {
		//nolint:gosec // G301: test directory
		if err := os.MkdirAll(filepath.Join(tmp, rel), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func runPkgs(t *testing.T, progress bool, parallelism int, rels ...string) string {
	t.Helper()
	pkgs := make([]string, 0, len(rels))
	for _, rel := range rels {
		pkgs = append(pkgs, modName+"/"+rel)
	}
	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	r := runner{
		goModule:       func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) { return pkgs, nil },
		measure:        fakeMeasure(90.0),
		progress:       gocover.Progress{Enabled: progress, Table: table()},
	}
	//nolint:exhaustruct // test uses partial initialization
	err := r.run(Opts{Out: &buf, DefaultThreshold: 80.0, Parallelism: parallelism})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return buf.String()
}

func assertInOrder(t *testing.T, out string, ordered ...string) {
	t.Helper()
	prev, prevIdx := "", -1
	for _, s := range ordered {
		idx := strings.Index(out, s)
		if idx < 0 {
			t.Errorf("output missing %q:\n%s", s, out)
			return
		}
		if idx <= prevIdx {
			t.Errorf("expected %q (%d) after %q (%d):\n%s", s, idx, prev, prevIdx, out)
		}
		prev, prevIdx = s, idx
	}
}

func TestRun_Parallelism(t *testing.T) {
	chdirWithPkgs(t, "pkg/a", "pkg/b", "pkg/c")
	out := runPkgs(t, false, 2, "pkg/a", "pkg/b", "pkg/c")
	assertInOrder(t, out, "pkg/a", "pkg/b", "pkg/c")
}

func TestRun_EmitsProgress_WhenAutoParallelism(t *testing.T) {
	chdirWithPkgs(t, "pkg/a", "pkg/b")
	out := runPkgs(t, true, 0, "pkg/a", "pkg/b")
	for _, marker := range []string{"[1/2]", "[2/2]"} {
		if !hasProgressLineWith(out, marker, "90.0%") {
			t.Errorf("expected %s progress line to include coverage data (90.0%%):\n%s",
				marker, out)
		}
	}
	if got := strings.Count(out, "STATUS"); got != 1 {
		t.Errorf("expected one STATUS header (progress IS the table); got %d:\n%s",
			got, out)
	}
	assertInOrder(t, out, "Running 2 packages with parallelism=", "COUNT")
	assertInOrder(t, out, "[1/2]", "All packages meet")
	assertInOrder(t, out, "[2/2]", "All packages meet")
	assertRowsOnce(t, out, "pkg/a", "pkg/b")
}

func TestRun_SuppressesProgress_WhenExplicitParallelism(t *testing.T) {
	chdirWithPkgs(t, "pkg/a", "pkg/b")
	out := runPkgs(t, false, 2, "pkg/a", "pkg/b")
	assertOutput(t, out, nil, []string{
		"[1/2]", "[2/2]", "Running 2 packages with parallelism=", "COUNT",
	})
	if got := strings.Count(out, "STATUS"); got != 1 {
		t.Errorf("expected one STATUS header (no progress section); got %d:\n%s",
			got, out)
	}
}

func assertRowsOnce(t *testing.T, out string, pkgs ...string) {
	t.Helper()
	for _, pkg := range pkgs {
		if got := strings.Count(out, pkg+"\n"); got != 1 {
			t.Errorf("expected %s row once, got %d:\n%s", pkg, got, out)
		}
	}
}
