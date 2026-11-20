package covgate

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/testutil"
)

func TestCheckPackage_Loose_Fires(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "10.0\n")

	//nolint:exhaustruct // test uses partial initialization
	run := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := run.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if res.passed {
		t.Error("expected fail for loose threshold")
	}
	if !strings.Contains(res.output, "LOOSE") {
		t.Errorf("output missing LOOSE: %s", res.output)
	}
	if !strings.Contains(res.output, "70.0pp") {
		t.Errorf("output missing 70.0pp gap: %s", res.output)
	}
	if !strings.Contains(res.output, ">= 79.5") {
		t.Errorf("output missing recommended 79.5: %s", res.output)
	}
}

func TestCheckPackage_Loose_WithinTolerance(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "79.6\n")

	//nolint:exhaustruct // test uses partial initialization
	run := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := run.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass within tolerance: %s", res.output)
	}
	if !strings.Contains(res.output, "PASS") {
		t.Errorf("output missing PASS: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE in output: %s", res.output)
	}
}

func TestCheckPackage_Loose_AtExactTolerance(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "79.5\n")

	//nolint:exhaustruct // test uses partial initialization
	run := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := run.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass at exact tolerance boundary: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE at boundary: %s", res.output)
	}
}

func TestCheckPackage_Loose_JustOverTolerance(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "79.4\n")

	//nolint:exhaustruct // test uses partial initialization
	run := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := run.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if res.passed {
		t.Errorf("expected fail just over tolerance: %s", res.output)
	}
	if !strings.Contains(res.output, "LOOSE") {
		t.Errorf("output missing LOOSE: %s", res.output)
	}
}

func TestCheckPackage_Loose_ZeroCoverageAllowed(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "0\n")

	//nolint:exhaustruct // test uses partial initialization
	run := runner{measure: fakeMeasure(0.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := run.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass for 0/0: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE for zero coverage: %s", res.output)
	}
}

func TestCheckPackage_Loose_NoCovgateFile_UsesDefault(t *testing.T) {
	testutil.MakePkgDir(t, pkgRel)

	//nolint:exhaustruct // test uses partial initialization
	run := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := run.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          10.0,
		tightnessEnabled:   true,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass when using default threshold: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE for default-fallback package: %s", res.output)
	}
}

func TestCheckPackage_Loose_Disabled(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "10.0\n")

	//nolint:exhaustruct // test uses partial initialization
	run := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := run.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   false,
		tightnessTolerance: 0.5,
	})
	if !res.passed {
		t.Errorf("expected pass when tightness disabled: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE when tightness disabled: %s", res.output)
	}
}

func TestCheckPackage_CustomTolerance(t *testing.T) {
	dir := testutil.MakePkgDir(t, pkgRel)
	testutil.WriteCovgateFile(t, dir, "70.0\n")

	//nolint:exhaustruct // test uses partial initialization
	run := runner{measure: fakeMeasure(80.0)}

	//nolint:exhaustruct // test uses partial initialization
	res := run.checkPackage(pkgName, checkPackageCtx{
		module:             modName,
		threshold:          50.0,
		tightnessEnabled:   true,
		tightnessTolerance: 15.0,
	})
	if !res.passed {
		t.Errorf("expected pass within custom tolerance: %s", res.output)
	}
	if strings.Contains(res.output, "LOOSE") {
		t.Errorf("unexpected LOOSE within custom tolerance: %s", res.output)
	}
}

func TestRun_LooseFailsOverall(t *testing.T) {
	dir := testutil.MakePkgDir(t, "pkg/a")
	testutil.WriteCovgateFile(t, dir, "10.0\n")

	var buf bytes.Buffer
	//nolint:exhaustruct // test uses partial initialization
	job := runner{
		goModule: func() (string, error) { return modName, nil },
		goListPackages: func(string) ([]string, error) {
			return []string{"example.com/mod/pkg/a"}, nil
		},
		measure: fakeMeasure(80.0),
	}

	//nolint:exhaustruct // test uses partial initialization
	err := job.run(Opts{
		Out:                &buf,
		DefaultThreshold:   50.0,
		TightnessEnabled:   true,
		TightnessTolerance: 0.5,
	})
	if err == nil {
		t.Fatal("expected error from loose threshold")
	}
	out := buf.String()
	if !strings.Contains(out, "LOOSE") {
		t.Errorf("output missing LOOSE: %s", out)
	}
	if !strings.Contains(out, "have loose .covgate thresholds") {
		t.Errorf("output missing updated error message: %s", out)
	}
}
