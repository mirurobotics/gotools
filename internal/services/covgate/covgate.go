package covgate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mirurobotics/gotools/internal/services/gocover"
)

// Opts holds the options for the covgate service.
type Opts struct {
	Packages string
	// Exclude is a comma-separated list of Go list patterns whose
	// matched import paths are removed from the measurement set
	// before tests run. Empty means no exclusion.
	Exclude            string
	SrcPrefix          string
	TestDir            string
	DefaultThreshold   float64
	Parallelism        int
	TightnessEnabled   bool
	TightnessTolerance float64
	Out                io.Writer
}

type runner struct {
	goModule           func() (string, error)
	goListPackages     func(string) ([]string, error)
	goListTestPackages func(string) ([]string, error)
	measure            func(pkg string, testPaths []string) (float64, []byte, error)
	runTests           func(paths []string) ([]byte, error)
	parallelism        int
	emitProgress       bool
}

// workItem is a single scheduled measurement. testOnly marks a
// package under --test-dir that no source package claims via the
// src-prefix mapping: it is run without coverage instrumentation,
// since there is no source package to attribute coverage to.
type workItem struct {
	pkg      string
	testOnly bool
}

// effectiveParallelism is intentionally duplicated in covratchet;
// keep them in sync.
func effectiveParallelism(opts Opts) int {
	if opts.Parallelism > 0 {
		return opts.Parallelism
	}
	return runtime.NumCPU()
}

// Run checks per-package coverage against thresholds.
func Run(opts Opts) error {
	r := runner{
		goModule:           gocover.GoModule,
		goListPackages:     gocover.GoListPackages,
		goListTestPackages: gocover.GoListTestPackages,
		measure:            gocover.Measure,
		runTests:           gocover.RunTests,
		parallelism:        effectiveParallelism(opts),
		emitProgress:       opts.Parallelism == 0,
	}
	return r.run(opts)
}

func (r *runner) run(opts Opts) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	w := opts.Out

	parallelism := r.parallelism
	if parallelism <= 0 {
		parallelism = effectiveParallelism(opts)
	}

	_, _ = fmt.Fprintf(
		w, "Checking per-package coverage "+
			"(default minimum: %.1f%%)...\n\n",
		opts.DefaultThreshold,
	)

	module, items, excluded, err := r.discover(opts, w)
	if err != nil {
		return err
	}

	r.writeRunHeader(w, len(items), parallelism)

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

// discover resolves the module, lists source and orphan test packages,
// and returns the work items left after --exclude along with the
// excluded packages.
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

// findOrphanTests returns test-bearing packages under --test-dir
// that no source package claims through the src-prefix mapping.
// Without this, a test directory whose source counterpart is not
// a Go package (e.g. a parent directory holding only subpackages)
// is silently never run. The claimed set is computed from the
// full pre-exclude source list so that an excluded source
// package's external tests stay excluded rather than resurfacing
// as orphans.
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

// claimedTestDirs returns the set of test directories claimed by source
// packages through the src-prefix mapping. Packages outside srcPrefix
// claim nothing, and an empty srcPrefix claims nothing at all.
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

// buildWorkItems merges source packages and orphan test packages
// into one work list, applying --exclude across both so that
// exclude patterns (e.g. preflight skip lists) can drop orphan
// suites the same way they drop source packages.
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

// resolveExcludes expands each comma-separated exclude pattern with go
// list and returns the set of matched packages.
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

// applyExclude removes packages matched by the comma-separated
// exclude patterns from pkgs. It preserves the original order of
// pkgs in both returned slices and prints a one-line notice to w
// when any package is actually removed. An empty (or
// whitespace-only) exclude string returns pkgs unchanged, a nil
// excluded slice, and no output.
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

func (r *runner) runPackages(
	items []workItem, ctx checkPackageCtx, parallelism int, w io.Writer,
) []checkResult {
	total := len(items)
	results := make([]checkResult, total)
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
	var progressMu sync.Mutex
	countWidth := len(strconv.Itoa(total))
	colWidth := progressColWidth(total)

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, it workItem) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = r.checkItem(it, ctx)
			if r.emitProgress {
				label := fmt.Sprintf("[%*d/%d]", countWidth, idx+1, total)
				progressMu.Lock()
				_, _ = fmt.Fprintf(
					w, "%-*s  %s",
					colWidth, label, firstLine(results[idx].output),
				)
				progressMu.Unlock()
			}
		}(i, item)
	}
	wg.Wait()
	return results
}
