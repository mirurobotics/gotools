package covgate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mirurobotics/gotools/internal/services/gocover"
)

// Opts holds the options for the covgate service.
type Opts struct {
	Packages string
	// Exclude lists comma-separated go list patterns to drop before tests run.
	Exclude            string
	SrcPrefix          string
	TestDir            string
	DefaultThreshold   float64
	Parallelism        int
	TightnessEnabled   bool
	TightnessTolerance float64
	Out                io.Writer
}

// Run checks per-package coverage against thresholds.
func Run(opts Opts) error {
	progress := gocover.Progress{Enabled: opts.Parallelism == 0, Table: table()}
	r := runner{
		goModule:           gocover.GoModule,
		goListPackages:     gocover.GoListPackages,
		goListTestPackages: gocover.GoListTestPackages,
		measure:            gocover.Measure,
		runTests:           gocover.RunTests,
		progress:           progress,
	}
	return r.run(opts)
}

type workItem struct {
	pkg      string
	testOnly bool
}

type runner struct {
	goModule           func() (string, error)
	goListPackages     func(string) ([]string, error)
	goListTestPackages func(string) ([]string, error)
	measure            func(pkg string, testPaths []string) (float64, []byte, error)
	runTests           func(paths []string) ([]byte, error)
	progress           gocover.Progress
}

func (r *runner) run(opts Opts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	w := opts.Out

	parallelism := gocover.Parallelism(opts.Parallelism)

	_, _ = fmt.Fprintf(
		w, "Checking per-package coverage "+
			"(default minimum: %.1f%%)...\n\n",
		opts.DefaultThreshold,
	)

	module, items, excluded, err := r.discover(opts, w)
	if err != nil {
		return err
	}

	r.progress.WriteHeader(w, len(items), parallelism)

	ctx := checkPackageCtx{
		module:             module,
		srcPrefix:          opts.SrcPrefix,
		testDir:            opts.TestDir,
		threshold:          opts.DefaultThreshold,
		tightnessEnabled:   opts.TightnessEnabled,
		tightnessTolerance: opts.TightnessTolerance,
	}

	start := time.Now()
	results := r.runPackages(items, ctx, parallelism, w)
	wallTime := time.Since(start)
	return r.printResults(w, results, excluded, module, wallTime)
}

func (r *runner) discover(
	opts Opts, w io.Writer,
) (module string, items []workItem, excluded []string, err error) {
	module, err = r.goModule()
	if err != nil {
		return "", nil, nil, err
	}

	pkgs, err := r.goListPackages(opts.Packages)
	if err != nil {
		return "", nil, nil, err
	}

	orphans, err := r.findOrphanTests(pkgs, module, opts)
	if err != nil {
		return "", nil, nil, err
	}

	items, excluded, err = r.buildWorkItems(pkgs, orphans, opts.Exclude, w)
	if err != nil {
		return "", nil, nil, err
	}
	return module, items, excluded, nil
}

func (r *runner) findOrphanTests(
	pkgs []string, module string, opts Opts,
) ([]string, error) {
	if opts.TestDir == "" {
		return nil, nil
	}
	testDir := filepath.ToSlash(filepath.Clean(opts.TestDir))
	claimed := claimedTestDirs(pkgs, module, testDir, opts.SrcPrefix)
	testPkgs, err := r.goListTestPackages("./" + testDir + "/...")
	if err != nil {
		return nil, err
	}
	orphans := make([]string, 0, len(testPkgs))
	for _, testPkg := range testPkgs {
		if _, ok := claimed[gocover.RelPkg(testPkg, module)]; ok {
			continue
		}
		orphans = append(orphans, testPkg)
	}
	return orphans, nil
}

func (r *runner) buildWorkItems(
	pkgs, orphans []string, exclude string, w io.Writer,
) ([]workItem, []string, error) {
	all := make([]string, 0, len(pkgs)+len(orphans))
	all = append(all, pkgs...)
	all = append(all, orphans...)
	kept, excluded, err := r.applyExclude(all, exclude, w)
	if err != nil {
		return nil, nil, err
	}
	orphanSet := make(map[string]struct{}, len(orphans))
	for _, orphan := range orphans {
		orphanSet[orphan] = struct{}{}
	}
	items := make([]workItem, 0, len(kept))
	for _, pkg := range kept {
		_, testOnly := orphanSet[pkg]
		items = append(items, workItem{pkg: pkg, testOnly: testOnly})
	}
	return items, excluded, nil
}

func (r *runner) applyExclude(
	pkgs []string, exclude string, w io.Writer,
) (kept []string, excluded []string, err error) {
	if strings.TrimSpace(exclude) == "" {
		return pkgs, nil, nil
	}

	excludedSet, err := r.resolveExcludes(exclude)
	if err != nil {
		return nil, nil, err
	}

	kept = make([]string, 0, len(pkgs))
	excluded = make([]string, 0, len(excludedSet))
	for _, p := range pkgs {
		if _, drop := excludedSet[p]; drop {
			excluded = append(excluded, p)
			continue
		}
		kept = append(kept, p)
	}

	if len(excluded) > 0 {
		_, _ = fmt.Fprintf(
			w, "Excluded %d package(s) from coverage measurement\n",
			len(excluded),
		)
	}
	return kept, excluded, nil
}

func (r *runner) resolveExcludes(exclude string) (map[string]struct{}, error) {
	excludedSet := make(map[string]struct{})
	for _, raw := range strings.Split(exclude, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		matched, err := r.goListPackages(entry)
		if err != nil {
			return nil, fmt.Errorf("resolve exclude %q: %w", entry, err)
		}
		for _, p := range matched {
			excludedSet[p] = struct{}{}
		}
	}
	return excludedSet, nil
}

func (r *runner) runPackages(
	items []workItem, ctx checkPackageCtx, parallelism int, w io.Writer,
) []checkResult {
	run := func(i int) checkResult {
		var res checkResult
		if items[i].testOnly {
			res = r.checkTestOnly(items[i].pkg, ctx)
		} else {
			res = r.checkPackage(items[i].pkg, ctx)
		}
		return res
	}
	row := func(res checkResult) string { return res.output }
	return gocover.RunEach(r.progress, w, len(items), parallelism, run, row)
}

func claimedTestDirs(
	pkgs []string, module, testDir, srcPrefix string,
) map[string]struct{} {
	claimed := make(map[string]struct{}, len(pkgs))
	if srcPrefix == "" {
		return claimed
	}
	for _, pkg := range pkgs {
		relPkg := gocover.RelPkg(pkg, module)
		testSub := strings.TrimPrefix(relPkg, srcPrefix+"/")
		if testSub == relPkg {
			continue
		}
		claimed[testDir+"/"+testSub] = struct{}{}
	}
	return claimed
}
