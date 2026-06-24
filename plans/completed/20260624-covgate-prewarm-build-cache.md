# Pre-warm the Go build cache before covgate's parallel per-package coverage runs

This ExecPlan is a living document. The sections Progress, Surprises & Discoveries, Decision Log, and Outcomes & Retrospective must be kept up to date as work proceeds.

## Scope

| Repository | Access | Description |
|-----------|--------|-------------|
| `gotools/` | read-write | Add a build-cache pre-warm pass to the covgate service and tests |

This plan lives in `gotools/plans/backlog/` because all code changes happen inside the `gotools` repo, in `internal/services/covgate/` (and a small read-only reuse of `internal/services/gocover/`). No CLI, DB, or external-service changes are required.

## Purpose / Big Picture

`miru covgate` enforces per-package Go coverage thresholds. It resolves the package set, then fans out `N` parallel per-package coverage runs (`N` = effective parallelism, default `runtime.NumCPU()`). Each run executes, via `gocover.Measure`:

    go test -coverprofile=<tmp> -coverpkg=<pkg> <testPaths...>

The `-coverpkg=<pkg>` flag instruments only that package's own (small, unique) target. But every one of those `go test` invocations must first **compile the shared, non-instrumented dependency packages** that the test binary imports. Go does not lock build actions across separate `go test` processes, so when `N` runs start simultaneously they each independently begin compiling the same heavy shared dependencies before any process has populated `~/.cache/go-build`. This is a compile stampede that roughly doubles per-package wall time on heavy packages.

The fix proven downstream: run **one** coherent compile pass over the same package + test-path set first — `go test -run='^$' <paths...>` with no `-coverpkg` instrumentation — so the shared dependencies land in the build cache exactly once. After that single pass, the `N` parallel coverage runs find their shared deps already cached and only pay for their own small instrumented target. In CI this cut covgate's wall time from ~188s to ~44s.

Key insight that shapes the design: **the stampede is on the SHARED NON-INSTRUMENTED dependency compiles, not on the per-package instrumented target.** Each coverage run instruments only its own unique package via `-coverpkg`. Therefore a single **plain** (non-instrumented) compile pass over the union of package + test paths is sufficient. We must **not** replicate per-package `-coverpkg` instrumentation in the warm pass — doing so would defeat Go's single build planner and re-introduce per-package work.

After this change:

- When covgate runs with parallelism > 1 **and** more than one package, it performs one warm-up compile pass before fanning out, then runs the parallel coverage checks exactly as today.
- When parallelism <= 1 **or** there is only a single package (no concurrency, hence no stampede), the warm pass is skipped and behavior is byte-for-byte identical to today.
- Coverage results, thresholds, `--exclude` handling, printed rows, and progress output are all unchanged. The warm pass is purely cache-warming.
- If the warm compile fails with a genuine build error, covgate propagates the error instead of silently proceeding.

## Progress

- [ ] Milestone 1 — Add an injectable `prewarm` seam to the `runner` struct and wire the real implementation in `Run`
- [ ] Milestone 2 — Implement the warm-pass helper in `gocover` (one `go test -run='^$'` over the union of test paths)
- [ ] Milestone 3 — Call the warm pass in `(*runner).run` before `runPackages`, gated on `parallelism > 1 && len(pkgs) > 1`, propagating errors
- [ ] Milestone 4 — Add/extend tests in `covgate_test.go` for the three seam assertions (invoked-once / skipped / error-propagated)
- [ ] Milestone 5 — Run preflight (`./scripts/preflight.sh`) and ensure it reports `clean`; fix anything flagged

Use timestamps when you complete steps. Split partially completed work into "done" and "remaining" as needed.

## Surprises & Discoveries

(Add entries as work proceeds.)

## Decision Log

- Decision: The warm pass is a **single** `go test -run='^$' <paths...>` invocation over the union of all package + test paths, NOT a per-package loop.
  Rationale: Go's single build planner deduplicates and parallelizes the shared dependency compiles within one invocation. A per-package loop would re-stampede or serialize redundantly. One invocation is the whole point.
  Date/Author: 2026-06-24, planning subagent.

- Decision: The warm pass uses **plain** compilation — no `-coverpkg`, no `-coverprofile`.
  Rationale: The stampede is on shared non-instrumented dependency compiles. Each real run instruments only its own unique `-coverpkg` target, which is small and not shared, so warming it buys nothing. Adding `-coverpkg` to the warm pass would force a coverage-instrumented build of the union and defeat the single-planner benefit. `-run='^$'` matches no tests, so the pass compiles the test binaries but executes nothing.
  Date/Author: 2026-06-24, planning subagent.

- Decision: The warm pass covers exactly the **same package + test-path set** the real per-package runs will build, computed by reusing the existing `gocover.BuildTestPaths(pkg, relPkg, srcPrefix, testDir)` logic that `checkPackage` already uses — collected into a de-duplicated union. Do not duplicate path-construction logic.
  Rationale: Warming a different set than the real runs build would leave some shared deps cold (under-warm) or compile unused packages (waste). Reuse guarantees the warm set tracks the real set across future path-logic changes.
  Date/Author: 2026-06-24, planning subagent.

- Decision: Gate the warm pass automatically on `parallelism > 1 && len(pkgs) > 1`. No new `Opts` flag in the first cut.
  Rationale: With parallelism <= 1 there is no concurrency and thus no stampede; with a single package there is nothing to stampede against. In both cases the warm pass is pure overhead (it would double-compile that one package). The automatic gate matches the prompt's accepted default and keeps the `Opts` surface unchanged. A `--no-prewarm` opt-out can be added later if a need arises; it is intentionally out of scope here to avoid touching the CLI surface and `lint-surface` checks.
  Date/Author: 2026-06-24, planning subagent.

- Decision: The reported **"Total time" / wall-clock stays measured exactly as today** — around `runPackages` only (the existing `start := time.Now()` / `wallTime := time.Since(start)` bracket in `(*runner).run`). The warm pass runs **before** that bracket and is treated as setup, excluded from the reported total.
  Rationale: "Total time" today communicates the parallel measurement phase. Keeping its definition stable preserves byte-comparable output for existing callers and tests (`TestPrintResults_UsesWallTime`, `TestRun_OutputContainsTiming`), and avoids conflating one-time cache warming with the steady-state measurement cost. The prompt explicitly prefers keeping the reported time measured the same way; this is that choice, stated.
  Date/Author: 2026-06-24, planning subagent.

- Decision: Warm-pass build failure propagates as an error from `(*runner).run` (returned before `runPackages` is reached).
  Rationale: A genuine build error means the per-package runs would fail too; surfacing it early is correct and avoids a confusing later failure. The warm pass must never silently swallow a compile error.
  Date/Author: 2026-06-24, planning subagent.

## Outcomes & Retrospective

(Summarize at completion or major milestones.)

## Context and Orientation

Covgate is a Go test-coverage gate exposed as the `covgate` subcommand of the `miru` CLI. All paths below are relative to the repo root `gotools/`.

### Service layer — `internal/services/covgate/covgate.go`

- `Opts` struct: `Packages`, `Exclude`, `SrcPrefix`, `TestDir`, `DefaultThreshold`, `Parallelism`, `TightnessEnabled`, `TightnessTolerance`, `Out`. (No new field is added by this plan.)
- `runner` struct currently injects three seams plus two scalar fields:

      type runner struct {
          goModule       func() (string, error)
          goListPackages func(string) ([]string, error)
          measure        func(pkg string, testPaths []string) (float64, []byte, error)
          parallelism    int
          emitProgress   bool
      }

  This plan adds a fourth seam, `prewarm`, alongside them (see Milestone 1).
- `Run(opts Opts) error` constructs the `runner`, wiring the real `gocover.GoModule`, `gocover.GoListPackages`, `gocover.Measure`, `effectiveParallelism(opts)`, and `emitProgress`. It then calls `r.run(opts)`. This is where the real `prewarm` implementation is wired.
- `(*runner).run(opts Opts)` is the orchestration entry point. Current flow:
  1. resolve `module` via `r.goModule()`
  2. resolve `pkgs` via `r.goListPackages(opts.Packages)`
  3. `r.applyExclude(pkgs, opts.Exclude, w)` → final `pkgs`, `excluded`
  4. `r.writeRunHeader(w, len(pkgs), parallelism)`
  5. build `checkPackageCtx`
  6. `start := time.Now()` → `results := r.runPackages(pkgs, ctx, parallelism, w)` → `wallTime := time.Since(start)`
  7. `r.printResults(w, results, excluded, module, wallTime)`

  The warm pass is inserted **between step 5 and step 6** — after the final `pkgs` is known and `ctx` is built, but **before** the `start := time.Now()` timing bracket so it is excluded from "Total time".
- `checkPackageCtx` holds `module`, `srcPrefix`, `testDir`, `threshold`, `tightnessEnabled`, `tightnessTolerance`. The warm pass needs `module`, `srcPrefix`, `testDir` to build the same test paths.
- `(*runner).checkPackage(pkg, ctx)` is the per-package worker. It computes:

      relPkg := gocover.RelPkg(pkg, ctx.module)
      ...
      testPaths := gocover.BuildTestPaths(pkg, relPkg, ctx.srcPrefix, ctx.testDir)

  and then calls `r.measure(pkg, testPaths)`. **The warm pass must reuse this exact pair (`RelPkg` then `BuildTestPaths`) per package, unioned across all packages.**

### Dependency layer — `internal/services/gocover/gocover.go`

- `RelPkg(pkg, module string) string` — module-relative path.
- `BuildTestPaths(pkg, relPkg, srcPrefix, testDir string) []string` — returns `[]string{pkg}` plus, when an external test dir applies, `"./" + extPath`. This is the authoritative test-path construction; both `covgate.checkPackage` and `covratchet` already reuse it (`internal/services/covratchet/covratchet.go:213`).
- `Measure(pkg, testPaths)` / `MeasureWithEnv(pkg, testPaths, extraEnv)` — the real per-package coverage runner; uses `cmdutil.GoCommand("test", "-coverprofile=...", "-coverpkg="+pkg, testPaths...)`. The warm pass is the non-instrumented sibling of this: same `cmdutil.GoCommand` mechanism, but `go test -run='^$' <union paths...>` with no coverage flags.
- `cmdutil.GoCommand(args ...string) *exec.Cmd` (`internal/services/cmdutil/cmd.go`) — builds an `exec.Cmd` for `go ...` with `GOWORK=off`. The warm pass MUST use this so it inherits the same toolchain/env as the real runs (otherwise it could warm a different build graph than the measured runs).

### Tests — `internal/services/covgate/covgate_test.go`

- Established seam-injection pattern: construct a `runner{...}` literal with stubbed `goModule`, `goListPackages`, `measure`, marked `//nolint:exhaustruct // test uses partial initialization`. `fakeMeasure(cov)` returns a passing measure stub.
- `TestRun_AllPass`, `TestRun_WithFailure`, `TestRun_Parallelism`, `TestRun_Exclude`, `TestRun_GoListError` show the run-level wiring. New tests inject the new `prewarm` seam the same way.
- `TestPrintResults_UsesWallTime` and `TestRun_OutputContainsTiming` lock the "Total time" semantics that this change must not disturb.

### Validation scripts (`scripts/`)

- `scripts/preflight.sh` — runs `lint.sh`, `covgate.sh`, and `lint-surface.sh`; prints `=== All checks passed ===` and exits 0 when clean. **This is the canonical preflight gate; a clean preflight is mandatory before publishing.**

### Glossary

- **Warm pass / pre-warm**: a single `go test -run='^$' <paths...>` invocation that compiles the test binaries (populating the shared build cache) but runs no tests.
- **Stampede**: multiple concurrent `go test` processes independently compiling the same uncached shared dependency at once, because Go does not lock build actions across processes.
- **Union test-path set**: the de-duplicated concatenation of `BuildTestPaths(...)` across every package covgate will measure.
- **Seam**: an injectable function field on `runner`, substituted in tests.
- **Preflight clean**: `./scripts/preflight.sh` exits 0 and prints `=== All checks passed ===`.

## Plan of Work

### Milestone 1 — Add the `prewarm` seam to `runner` and wire it in `Run`

Edit `internal/services/covgate/covgate.go`.

1. Add a fourth function field to the `runner` struct, mirroring the existing seam style and signature shape of `measure`:

       type runner struct {
           goModule       func() (string, error)
           goListPackages func(string) ([]string, error)
           measure        func(pkg string, testPaths []string) (float64, []byte, error)
           prewarm        func(testPaths []string) error
           parallelism    int
           emitProgress   bool
       }

   The seam takes the already-computed union of test paths and returns only an error (it produces no coverage). Keeping path computation in covgate (not in the seam) lets unit tests assert on the exact path set passed in.

2. In `Run`, wire the real implementation (added in Milestone 2) alongside the existing seams:

       r := runner{
           goModule:       gocover.GoModule,
           goListPackages: gocover.GoListPackages,
           measure:        gocover.Measure,
           prewarm:        gocover.PrewarmBuild,
           parallelism:    effectiveParallelism(opts),
           emitProgress:   opts.Parallelism == 0,
       }

### Milestone 2 — Implement the warm-pass helper in `gocover`

Edit `internal/services/gocover/gocover.go`. Add an exported `PrewarmBuild`:

       // PrewarmBuild compiles the test binaries for the given test
       // paths without running any tests, populating the shared Go
       // build cache so that subsequent parallel `go test` runs do
       // not stampede on the same shared dependency compiles. It runs
       // a single `go test -run='^$' <paths...>` (no coverage
       // instrumentation) so Go's build planner deduplicates the
       // shared compiles in one coherent pass. A genuine build error
       // is returned (with combined output) so callers can surface it.
       func PrewarmBuild(testPaths []string) error {
           if len(testPaths) == 0 {
               return nil
           }
           args := make([]string, 0, 2+len(testPaths))
           args = append(args, "test", "-run=^$")
           args = append(args, testPaths...)
           cmd := cmdutil.GoCommand(args...)
           if out, err := cmd.CombinedOutput(); err != nil {
               return fmt.Errorf("prewarm build: %w\n%s", err, out)
           }
           return nil
       }

   Notes:
   - Reuse `cmdutil.GoCommand` so the warm pass inherits `GOWORK=off` and the same toolchain as `Measure`/`MeasureWithEnv`.
   - Do **not** add `-coverpkg` or `-coverprofile` — plain compilation only (see Decision Log).
   - `-run=^$` matches no test names, so binaries compile but nothing executes.

### Milestone 3 — Call the warm pass in `(*runner).run`, gated and error-propagating

Edit `internal/services/covgate/covgate.go` inside `(*runner).run`, **after** the `checkPackageCtx` is built and **before** `start := time.Now()`:

1. Compute the gate and, when it holds, the union of test paths, then call the seam:

       if parallelism > 1 && len(pkgs) > 1 {
           warmPaths := collectWarmPaths(pkgs, ctx)
           if err := r.prewarm(warmPaths); err != nil {
               return err
           }
       }

2. Add a small helper that reuses the exact path logic `checkPackage` uses, unioned and de-duplicated while preserving first-seen order:

       // collectWarmPaths returns the de-duplicated union of the test
       // paths covgate will build for every package, using the same
       // RelPkg + BuildTestPaths logic as checkPackage so the warm
       // pass compiles exactly the set the real runs need.
       func collectWarmPaths(pkgs []string, ctx checkPackageCtx) []string {
           seen := make(map[string]struct{})
           var paths []string
           for _, pkg := range pkgs {
               relPkg := gocover.RelPkg(pkg, ctx.module)
               for _, p := range gocover.BuildTestPaths(
                   pkg, relPkg, ctx.srcPrefix, ctx.testDir,
               ) {
                   if _, ok := seen[p]; ok {
                       continue
                   }
                   seen[p] = struct{}{}
                   paths = append(paths, p)
               }
           }
           return paths
       }

3. Leave the existing timing bracket, `runPackages`, and `printResults` untouched. "Total time" continues to measure only the parallel phase.

The relevant region of `(*runner).run` becomes:

       ctx := checkPackageCtx{ ... }   // unchanged

       if parallelism > 1 && len(pkgs) > 1 {
           if err := r.prewarm(collectWarmPaths(pkgs, ctx)); err != nil {
               return err
           }
       }

       start := time.Now()
       results := r.runPackages(pkgs, ctx, parallelism, w)
       wallTime := time.Since(start)
       return r.printResults(w, results, excluded, module, wallTime)

### Milestone 4 — Tests

See **Test Steps** below for the enumerated cases. All new tests inject the `prewarm` seam using the existing `//nolint:exhaustruct` runner-literal style.

### Milestone 5 — Validation pass

Run `./scripts/preflight.sh` from the repo root and fix any findings until it reports clean (see Validation and Acceptance).

## Concrete Steps

All commands run from `/home/ben/miru/workbench6/repos/gotools/`. The current branch is `perf/covgate-prewarm-build-cache`; do not switch branches.

### Step 0 — Sanity baseline

    git status
    go build ./...
    go test ./internal/services/covgate/... ./internal/services/gocover/...

Expected: clean tree (or only this plan file added), build exit 0, both packages `ok`.

### Step 1 — Implement Milestones 1–3 (source)

Edit `internal/services/gocover/gocover.go` (add `PrewarmBuild`) and `internal/services/covgate/covgate.go` (add `prewarm` seam, wire `Run`, add `collectWarmPaths`, insert the gated call). Then:

    go build ./...

Expected: exit 0.

### Step 2 — Implement Milestone 4 (tests)

Edit `internal/services/covgate/covgate_test.go` per Test Steps below. Then:

    go test ./internal/services/covgate/... ./internal/services/gocover/...

Expected: `ok` for both.

### Step 3 — Milestone 5 preflight

    ./scripts/preflight.sh

Expected final line `=== All checks passed ===`, exit 0. If not clean, read the offending output (lint / covgate / surface lint — note `PrewarmBuild` is a new exported symbol that `lint-surface` will see; that is expected and acceptable as a deliberate API addition), fix in place, and rerun until clean.

## Test Steps

Add the following to `internal/services/covgate/covgate_test.go`, using the established seam-injection pattern (a `runner{...}` literal with stubbed `goModule`, `goListPackages`, `measure`, plus the new `prewarm`, each marked `//nolint:exhaustruct`). Use `fakeMeasure(90.0)` for the measure stub. These cover the three required assertions.

The shared stub shape for the `prewarm` seam records its invocations:

    var (
        warmCalls int
        warmPaths []string
    )
    prewarm := func(paths []string) error {
        warmCalls++
        warmPaths = paths
        return nil
    }

### Test 1 — `TestRun_Prewarm_InvokedOnce_WhenParallelAndMultiPkg`

Assertion (1): warm pass invoked exactly once with the full package/test-path set when parallelism > 1 and there are multiple packages.

- Use a single temp dir with subdirs `pkg/a`, `pkg/b`, `pkg/c` (mirror `TestRun_Parallelism`'s `t.TempDir()` + `t.Chdir` setup so `BuildTestPaths`/`RelPkg` resolve).
- `goModule` returns `modName`; `goListPackages` returns the three `modName + "/pkg/{a,b,c}"` paths.
- Inject `prewarm` recording stub; run `Opts{Out: &buf, DefaultThreshold: 80.0, Parallelism: 3}`.
- Assert: `warmCalls == 1`.
- Assert the union of test paths was passed: each of the three packages' import paths appears in `warmPaths` (with `SrcPrefix`/`TestDir` empty, `BuildTestPaths` returns just `[]string{pkg}`, so `warmPaths` should equal the three package paths). Confirm length 3 and membership.
- Sanity: `err == nil`, output still contains all three rows and `Total time:`.

### Test 2 — `TestRun_Prewarm_Skipped_WhenSinglePackageOrSerial`

Assertion (2): warm pass skipped when parallelism <= 1 OR a single package. Table-driven with subtests:

- **SingleParallelism**: three packages, `Parallelism: 1`. Assert `warmCalls == 0`.
- **SinglePackage**: one package `pkg/a`, `Parallelism: 4`. Assert `warmCalls == 0`.
- (Optional) **SinglePackageSerial**: one package, `Parallelism: 1`. Assert `warmCalls == 0`.

In every subtest assert `err == nil` and that the normal per-package rows still print (behavior unchanged when warm pass is skipped).

### Test 3 — `TestRun_Prewarm_ErrorPropagates`

Assertion (3): a warm-pass build error is propagated and `runPackages` is not reached.

- Three packages, `Parallelism: 3` (gate satisfied).
- Inject `prewarm` returning `fmt.Errorf("prewarm build: boom")`.
- Inject a `measure` stub that flips a bool / increments a counter if called, so the test can assert the real runs did **not** start.
- Run and assert: `err != nil`; `err.Error()` contains `"prewarm build"` (or `"boom"`); the measure stub was never invoked; the output does **not** contain `Total time:` (run returned before printing results).

### Notes on existing tests

- `TestPrintResults_UsesWallTime` and `TestRun_OutputContainsTiming` must remain green unchanged — they validate the "Total time" semantics this change deliberately preserves.
- Existing `run`-level tests that build a `runner` literal without a `prewarm` field still satisfy their cases because they use `Parallelism: 0/1/2` with package counts where the gate is false (single package) or are unaffected; where an existing test has parallelism > 1 with multiple packages (e.g. `TestRun_Parallelism` uses `Parallelism: 2` with three packages), it will now invoke `r.prewarm`, which is a **nil** function in that literal and would panic. **Therefore: audit every existing `runner{...}` literal whose `run`/`Run` path can satisfy `parallelism > 1 && len(pkgs) > 1` and give it a no-op `prewarm: func([]string) error { return nil }`.** Concretely this includes `TestRun_Parallelism` and `TestRun_SuppressesProgress_WhenExplicitParallelism` (both `Parallelism: 2`, three / two packages). Add the no-op seam to those literals as part of Milestone 4. Tests that go through `Run` (the public wrapper) get the real `prewarm` automatically and need no change.

## Validation and Acceptance

Acceptance is observable behavior plus a clean preflight, in this order. **Changes MUST NOT be published (no push, no PR) until preflight reports `clean`.**

1. **gocover tests pass.**

       go test ./internal/services/gocover/...

   Expected: `ok`. `PrewarmBuild` compiles and behaves (covered indirectly; the covgate seam tests assert the wiring).

2. **covgate tests pass**, including the three new tests `TestRun_Prewarm_InvokedOnce_WhenParallelAndMultiPkg`, `TestRun_Prewarm_Skipped_WhenSinglePackageOrSerial`, `TestRun_Prewarm_ErrorPropagates`, and the unchanged timing tests:

       go test ./internal/services/covgate/...

   Expected: `ok`.

3. **Build is clean.**

       go build ./...

   Expected: exit 0, no output.

4. **Output is unchanged for callers.** No new rows, no change to `Total time` definition, no change to `--exclude`/threshold/tightness output. Confirmed by the surviving `TestRun_*` output assertions and `TestPrintResults_UsesWallTime`.

5. **Preflight is clean — mandatory gate.**

       ./scripts/preflight.sh

   Expected final line: `=== All checks passed ===` with exit 0. **No changes are published, no PR is opened, and no branch is pushed until this command reports `clean`.** If it fails on lint, covgate, or surface lint, fix the underlying issue and rerun. Do not bypass preflight.

## Idempotence and Recovery

- Source edits (Milestones 1–3) are additive: one new exported function (`gocover.PrewarmBuild`), one new struct field (`runner.prewarm`), one new unexported helper (`collectWarmPaths`), and one gated call site. Re-applying is safe; if partially applied, re-read the file and reconcile against Plan of Work.
- Test edits (Milestone 4): if Go reports duplicate function names after a retry, remove the half-applied block and re-paste. Remember the no-op `prewarm` seam must be added to the pre-existing parallel `runner` literals (`TestRun_Parallelism`, `TestRun_SuppressesProgress_WhenExplicitParallelism`) or those tests will nil-panic.
- Preflight (Milestone 5) is idempotent — rerun freely.
- Rollback: the change is purely cache-warming with no semantic effect on results; revert the milestone commits in reverse order with `git revert <sha>` if it must be removed after merge. Use `git restore <path>` (not `git reset --hard`) to discard uncommitted work on specific files during a retry.
