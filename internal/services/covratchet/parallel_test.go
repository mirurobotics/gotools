package covratchet

import (
	"bytes"
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
	err := r.run(Opts{Out: &buf, Parallelism: 0})
	if err != nil {
		t.Fatalf("unexpected error with Parallelism=0 (NumCPU): %v", err)
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
	if err := r.run(Opts{Out: &buf, Parallelism: parallelism}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return buf.String()
}

func TestRun_Parallelism(t *testing.T) {
	testutil.MakePkgDirs(t, "pkg/a", "pkg/b", "pkg/c")
	out := runPkgs(t, false, 2, "pkg/a", "pkg/b", "pkg/c")
	testutil.AssertInOrder(t, out, "pkg/a", "pkg/b", "pkg/c")
	if !strings.Contains(out, "Updated: 3") {
		t.Errorf("expected Updated: 3 in output: %s", out)
	}
}

func TestRun_EmitsProgress_WhenAutoParallelism(t *testing.T) {
	testutil.MakePkgDirs(t, "pkg/a", "pkg/b")
	out := runPkgs(t, true, 0, "pkg/a", "pkg/b")
	for _, marker := range []string{"[1/2]", "[2/2]"} {
		if !testutil.HasLineWith(out, marker, "pkg/") {
			t.Errorf("expected %s progress line to include the package path:\n%s",
				marker, out)
		}
	}
	if got := strings.Count(out, "STATUS"); got != 1 {
		t.Errorf("expected one STATUS header (progress IS the table); got %d:\n%s",
			got, out)
	}
	testutil.AssertInOrder(t, out, "Running 2 packages with parallelism=", "COUNT")
	testutil.AssertInOrder(t, out, "[1/2]", "Done.")
	testutil.AssertInOrder(t, out, "[2/2]", "Done.")
	testutil.AssertRowsOnce(t, out, "pkg/a", "pkg/b")
}

func TestRun_SuppressesProgress_WhenExplicitParallelism(t *testing.T) {
	testutil.MakePkgDirs(t, "pkg/a", "pkg/b")
	out := runPkgs(t, false, 2, "pkg/a", "pkg/b")
	for _, unwanted := range []string{
		"[1/2]", "[2/2]", "Running 2 packages with parallelism=", "COUNT",
	} {
		if strings.Contains(out, unwanted) {
			t.Errorf("output unexpectedly contains %q:\n%s", unwanted, out)
		}
	}
	if got := strings.Count(out, "STATUS"); got != 1 {
		t.Errorf("expected one STATUS header (no progress section); got %d:\n%s",
			got, out)
	}
}
