# Restore covgate/covratchet parallelism and add progress output when unbounded

## Scope

- `repos/gotools` (path: `/home/ben/miru/workbench3/repos/gotools`) — read-write. This plan file lives here.
- Base branch: `main`. Feature branch: `perf/restore-parallelism-and-progress` (already created).

No other repositories are read or written.

## Purpose / Big Picture

`covgate` and `covratchet` each measure per-package coverage by spawning one `go test` subprocess per package, bounded by a parallelism semaphore. Two recent regressions are visible on `main`:

1. **CI wall-clock time has gone up.** PR #28 (`c37f333`, "perf(covgate,covratchet): cap child GOMAXPROCS to avoid CI CPU oversubscription") lowered the effective worker count by switching the outer default from `runtime.NumCPU()` to `runtime.GOMAXPROCS(0)` *and* injecting `GOMAXPROCS=max(1, NumCPU/parallelism)` into every child `go test`. The intent was to avoid the NumCPU×NumCPU oversubscription on a constrained runner. In practice the cap is too aggressive for the actual CI runner (`ubuntu-latest-m` — a Miru-flavored larger runner where `GOMAXPROCS(0)` reports a much lower effective value than the physical CPU budget, and where build/link parallelism inside each `go test` mattered more than PR #28 assumed).
2. **No progress is printed while work is running.** Results are buffered into a `[]checkResult` (covgate) / `[]ratchetResult` (covratchet) and printed in original package order *after* the `WaitGroup` drains. For a long run with `--parallelism=0` (auto/unlimited) the user sees the header line, then silence, then a wall of output at the end. That makes it hard to tell whether covgate is making progress or stuck.

After this change:

- The outer default parallelism is restored toward the pre-PR-#28 behavior (`runtime.NumCPU()`), so on the `ubuntu-latest-m` runner CI gets back the throughput it had before. The cgroup-aware semantics PR #28 wanted are still available via the `GOMAXPROCS` env var (Go 1.25 honors it through `runtime.NumCPU()` as well — see Background — so callers who want a tighter cap can still set `GOMAXPROCS` in the environment).
- The child `GOMAXPROCS` injection is removed for `--parallelism=0` (auto/unlimited). It is retained, optionally, only when an explicit `--parallelism=N` is passed — see Design for the rationale and the alternative.
- When `--parallelism=0` (auto/unlimited), the runner emits per-package progress lines as work completes so the user can see throughput in real time. Lines go to `opts.Out` (same writer the final table uses), interleaved with the table only after all work finishes. Progress is suppressed when `--parallelism > 0` so users who pin a worker count still get the clean, deterministic table they have today.

## Progress

Add entries as work proceeds.

- [ ] Milestone 1: investigate / confirm CI runner CPU budget (record finding in Decision Log) and adjust the default-parallelism strategy accordingly.
- [ ] Milestone 2: restore default outer parallelism in covgate (`effectiveParallelism`), remove the unconditional child `GOMAXPROCS` cap for `parallelism=0`, update flag help text, and update tests.
- [ ] Milestone 3: apply the structurally identical change to covratchet, including the `.covgate` threshold (re-measure and adjust if needed).
- [ ] Milestone 4: add live progress reporting when `parallelism=0` (or, equivalently, when the auto-default path is taken). Add tests that pin the observable behavior (lines emitted as work completes, suppressed when `--parallelism > 0`).
- [ ] Milestone 5: full validation (build, test, covgate.sh, lint, preflight) and confirm preflight reports `clean` before publishing.

## Surprises & Discoveries

Add entries as work proceeds.

## Decision Log

Authoring decisions baked into this plan:

- 2026-05-17 — **Restore default outer parallelism to `runtime.NumCPU()`.** Rationale: PR #28's switch to `runtime.GOMAXPROCS(0)` plus an inner cap of `NumCPU/parallelism` was specifically aimed at the NumCPU×NumCPU oversubscription concern, but the repo has zero `t.Parallel` callsites — so the *inner* GOMAXPROCS only ever affected compile/link, not test execution. Capping it at `1` on `--parallelism=0` runs (which is what `NumCPU/NumCPU` produces) penalized the compile/link step on every single package without buying back any safety against test-time oversubscription. Reverting the outer default to `NumCPU()` brings CI wall-clock time back; keeping the env-respecting `GOMAXPROCS` escape hatch is still possible by setting `GOMAXPROCS` in the CI workflow if a tighter cap is ever needed.
- 2026-05-17 — **Drop the unconditional child `GOMAXPROCS=max(1, NumCPU/parallelism)` injection for the `--parallelism=0` path.** Rationale: with the outer count back to `NumCPU`, child GOMAXPROCS would compute to 1, which is the exact regression we are fixing. For explicit `--parallelism=N` (where `N < NumCPU`), the inner injection still has theoretical value for build/link parallelism budgeting, *but* the user-facing semantics of `--parallelism` would become surprising (specifying `-p 2` would silently change build parallelism inside each child). The simpler restoration is to drop the env injection entirely and document that `--parallelism` controls outer concurrency only. Callers who need both knobs can set `GOMAXPROCS` in the environment.
- 2026-05-17 — **Progress lines only when `parallelism=0`.** Rationale: the user asked for progress in the auto/unlimited case; explicit `--parallelism=N` runs are typically scripted (CI, benchmarks) and benefit from the existing deterministic-order final table. Adding progress unconditionally would change observable output for every CI consumer; scoping it to the auto path is a strictly additive change for users who already pin `-p`.
- 2026-05-17 — **Progress goes to `opts.Out`, not `os.Stderr`.** Rationale: the rest of the service writes to the injected `opts.Out` (defaults to `os.Stdout`). Tests construct a `bytes.Buffer` and pass it as `Out` to assert on output. Splitting progress to a separate writer would require a second `OutErr` field on `Opts`, complicating the API for a one-line-per-package feature.
- 2026-05-17 — **Progress format: one line per completed package, including index/total and elapsed time.** Example: `[3/42] PASS  internal/services/covgate  1.4s`. Rationale: matches the column shape the final table already uses (`STATUS … TIME … PACKAGE`), so the progress lines feel native, not bolted on. Index/total gives the user a fraction; per-package time gives a sense of throughput without requiring a TUI or carriage-return rewriting.
- 2026-05-17 — **No throttling/cadence dial.** Rationale: at most one line is emitted per package finish, and the package count is bounded by `go list ./...` (dozens, not thousands). Throttling would only add complexity.

## Outcomes & Retrospective

Add entries as work proceeds.

## Context and Orientation

The reader is assumed to have only the gotools working tree.

**Repository layout (relevant pieces):**

- `internal/services/covgate/covgate.go` — package `covgate`.
  - `Opts` (lines 16-29): includes `Parallelism int` (line 25) and `Out io.Writer` (line 28).
  - `effectiveParallelism(opts Opts) int` (lines 40-45): currently returns `runtime.GOMAXPROCS(0)` when `Parallelism <= 0`. **Will change** to return `runtime.NumCPU()`.
  - `childGOMAXPROCS(parallelism int) int` (lines 47-53): computes `max(1, runtime.GOMAXPROCS(0) / parallelism)`. **Will be removed** (along with the env injection in `Run`).
  - `Run(opts Opts) error` (lines 56-68): builds `extraEnv = []string{"GOMAXPROCS=N"}` and binds `runner.measure` to a closure calling `gocover.MeasureWithEnv` with that env. **Will change** to use `gocover.Measure` directly (no extra env).
  - `runner.run(opts Opts)` (lines 70-117): orchestrates list → exclude → header → `runPackages` → `printResults`. The buffered-then-printed pattern lives here.
  - `runner.runPackages(pkgs, ctx, parallelism)` (lines 166-184): the WaitGroup+semaphore loop. **This is where progress emission will be added.** It currently writes results into a pre-allocated `[]checkResult` indexed by input position and waits for the group; it does *not* emit anything during the run.
  - `printResults(w, results, excluded, module, wallTime)` (lines 186-217): prints the buffered results, the excluded rows, and the summary line.

- `internal/services/covratchet/covratchet.go` — package `covratchet`. Structurally identical:
  - `Opts` (lines 15-22): includes `Parallelism int` (line 20) and `Out io.Writer` (line 21).
  - `effectiveParallelism` (lines 33-38): same shape as covgate, returns `runtime.GOMAXPROCS(0)`. **Will change** to `runtime.NumCPU()`.
  - `childGOMAXPROCS` (lines 40-46): **will be removed** along with the env injection.
  - `Run(opts Opts)` (lines 49-61): builds `extraEnv` and binds `runner.measure` to `gocover.MeasureWithEnv`. **Will change** to call `gocover.Measure` directly.
  - `runner.run(opts Opts)` (lines 63-123): the WaitGroup+semaphore loop is inlined here at lines 89-102 (no `runPackages` helper). **This is where progress emission will be added.**

- `internal/commands/covgate.go` — Cobra wiring; flag help text at line 49: `"max concurrent package measurements (0 = GOMAXPROCS)"`. **Will change** to `"max concurrent package measurements (0 = NumCPU; emits progress)"`.
- `internal/commands/covratchet.go` — same shape at line 42. Same help-text change.

- `internal/services/gocover/gocover.go` — `Measure` (line 133) and `MeasureWithEnv` (line 143). After this change, covgate and covratchet should both call `Measure` directly; `MeasureWithEnv` is left in place for future callers but becomes unused inside the repo. The test `TestMeasureWithEnv_AppliesExtraEnv` in `gocover_test.go` still exercises it for coverage.

- `internal/testutil/testutil.go` — `MakePkgDir(t, rel)` and `WriteCovgateFile(t, dir, val)`. Tests in this plan reuse these.

**Existing tests that must be updated:**

- `internal/services/covgate/covgate_test.go`:
  - `TestRun_Parallelism_DefaultsToGOMAXPROCS` (line 440) — must be renamed to `TestRun_Parallelism_DefaultsToNumCPU` and the inline message at line 456 updated.
  - `TestEffectiveParallelism` (line 460) — `want := runtime.GOMAXPROCS(0)` at line 465 must change to `want := runtime.NumCPU()`.
  - `TestChildGOMAXPROCS` (line 472) — **delete entirely** (the helper is being removed).
  - `TestRun_PublicWrapper` and `TestRun_PublicWrapper_HappyPath` (lines 571, 585) — must be re-examined; the closure that called `MeasureWithEnv` is gone, so the body they exercise will change. They should keep exercising `Run(opts)` end-to-end against a real temp Go module so coverage stays at the current `100.0` threshold for the package.
- `internal/services/covratchet/covratchet_test.go`:
  - `TestRun_Parallelism_DefaultsToGOMAXPROCS` (line 390) — rename + message update.
  - `TestEffectiveParallelism` (line 410) — same `want` change.
  - `TestChildGOMAXPROCS` (line 422) — delete.
  - `TestRun_PublicWrapper_HappyPath` (line 432) — same re-examination as covgate.

**Coverage thresholds (enforced by `./scripts/covgate.sh`):**

- `internal/services/gocover/.covgate` = `92.1` — unchanged (we are not changing gocover behavior; `MeasureWithEnv` and its test stay).
- `internal/services/covgate/.covgate` = `100.0` — must remain at 100. The new progress code path needs explicit test coverage; the existing `TestRun_Parallelism` and the renamed default-parallelism test, plus the new `TestRun_EmitsProgress_WhenAutoParallelism` and `TestRun_SuppressesProgress_WhenExplicitParallelism`, are designed to keep coverage at 100.
- `internal/services/covratchet/.covgate` = `99.5` — likely needs to be re-measured (down if the removed `childGOMAXPROCS` branch reduces coverage; possibly back to ~97 range). Run `./scripts/covgate.sh` after the changes; if covgate reports tightness drift, update the file to the recommended value in a single commit at the end of Milestone 3.

**CI runner:** `.github/workflows/ci.yml` runs `./scripts/covgate.sh` and `LINT_FIX=0 ./scripts/lint.sh` on `ubuntu-latest-m` (a Miru-flavored larger runner). The premise of PR #28 — that `GOMAXPROCS(0)` is a tighter, cgroup-aware cap than `NumCPU()` — turned out to be empirically slower on this specific runner.

**Lint constraints:** the **collapse** linter (`internal/services/lint/linter/collapse/`) flags multi-line literals or signatures that would fit on one line. The `extraEnv` slice literal and `measure:` closure removed in this change were the original collapse-relevant pitfalls; the replacement code is simpler and does not introduce new ones.

## Plan of Work

The work is structured as five milestones, each ending in one git commit.

**Milestone 1 — Confirm the CI runner CPU budget and lock in the new default strategy.** Before changing code, capture the empirical observation that motivated the restoration: on `ubuntu-latest-m`, `runtime.NumCPU()` and `runtime.GOMAXPROCS(0)` differ in a way that hurt CI wall-clock time. Record the finding (and any links to recent CI run timings on `main`) in the Decision Log section above. This milestone is documentation-only; the deliverable is a one-line note in Decision Log.

If the investigation reveals that `NumCPU()` and `GOMAXPROCS(0)` *agree* on `ubuntu-latest-m` (i.e. the slowdown is entirely attributable to the child cap, not the outer source), narrow the change to "drop the child cap, keep `GOMAXPROCS(0)` as the outer source" and update the Decision Log accordingly. Milestones 2 and 3 still apply unchanged in shape; only the one-line `effectiveParallelism` return expression differs.

**Milestone 2 — Restore covgate defaults and remove child cap.**

In `internal/services/covgate/covgate.go`:

1. Change `effectiveParallelism` (lines 40-45) to return `runtime.NumCPU()` (or `runtime.GOMAXPROCS(0)`, per Milestone 1's outcome) when `Parallelism <= 0`.
2. Delete `childGOMAXPROCS` (lines 47-53).
3. In `Run(opts)` (lines 56-68), delete the `extraEnv := []string{...}` literal and the closure binding for `runner.measure`. Set `runner.measure` to `gocover.Measure` directly:

       r := runner{
           goModule:       gocover.GoModule,
           goListPackages: gocover.GoListPackages,
           measure:        gocover.Measure,
           parallelism:    parallelism,
       }

4. The `runtime` import is still needed for `effectiveParallelism`; the `fmt` import is still needed elsewhere. No import edits.

In `internal/commands/covgate.go`:

5. Update the `--parallelism` flag help text at line 49 from `"max concurrent package measurements (0 = GOMAXPROCS)"` to `"max concurrent package measurements (0 = NumCPU; emits progress)"`.

In `internal/services/covgate/covgate_test.go`:

6. Rename `TestRun_Parallelism_DefaultsToGOMAXPROCS` (line 440) to `TestRun_Parallelism_DefaultsToNumCPU` and update the inline message at line 456 from "GOMAXPROCS" to "NumCPU".
7. In `TestEffectiveParallelism` (line 460), change `want := runtime.GOMAXPROCS(0)` at line 465 to `want := runtime.NumCPU()`.
8. Delete `TestChildGOMAXPROCS` (lines 472-480).
9. Re-examine `TestRun_PublicWrapper` (line 571) and `TestRun_PublicWrapper_HappyPath` (line 585). Their original purpose was to exercise the closure body that called `MeasureWithEnv`; with the closure gone, they now exercise `gocover.Measure` directly. Keep them — they still cover `Run(opts)` end-to-end. If their internal assertions specifically referenced `GOMAXPROCS` env propagation, drop those assertions and keep the "returns nil, table printed" assertion.

Run the focused tests and the threshold script:

    go test ./internal/services/covgate/...
    ./scripts/covgate.sh

Expect `ok` and `All packages meet minimum coverage requirement`. Commit:

    git add internal/services/covgate/covgate.go internal/services/covgate/covgate_test.go internal/commands/covgate.go
    git commit -m "perf(covgate): restore NumCPU default and drop child GOMAXPROCS cap"

**Milestone 3 — Restore covratchet defaults and remove child cap.**

Apply structurally identical changes to `internal/services/covratchet/covratchet.go`:

1. Change `effectiveParallelism` (lines 33-38) to return `runtime.NumCPU()` (or whatever Milestone 1 settled on).
2. Delete `childGOMAXPROCS` (lines 40-46).
3. In `Run(opts)` (lines 49-61), drop the `extraEnv` literal and the closure; set `runner.measure: gocover.Measure` directly.

In `internal/commands/covratchet.go`:

4. Update flag help text at line 42 from `"max concurrent package measurements (0 = GOMAXPROCS)"` to `"max concurrent package measurements (0 = NumCPU; emits progress)"`.

In `internal/services/covratchet/covratchet_test.go`:

5. Rename `TestRun_Parallelism_DefaultsToGOMAXPROCS` (line 390) to `TestRun_Parallelism_DefaultsToNumCPU`; update the message at line 405.
6. In `TestEffectiveParallelism` (line 410), change `want := runtime.GOMAXPROCS(0)` to `want := runtime.NumCPU()`.
7. Delete `TestChildGOMAXPROCS` (lines 422-430).
8. Re-examine `TestRun_PublicWrapper_HappyPath` (line 432); same treatment as covgate's wrapper test.

Run focused tests and `./scripts/covgate.sh`:

    go test ./internal/services/covratchet/...
    ./scripts/covgate.sh

If covgate reports tightness drift on `internal/services/covratchet/.covgate`, update the file to the recommended value. Commit:

    git add internal/services/covratchet/covratchet.go internal/services/covratchet/covratchet_test.go internal/commands/covratchet.go internal/services/covratchet/.covgate
    git commit -m "perf(covratchet): restore NumCPU default and drop child GOMAXPROCS cap"

**Milestone 4 — Add live progress reporting when `parallelism=0`.**

The design: when the user did not pass `--parallelism` (i.e. `opts.Parallelism == 0`, the auto/unlimited branch), each goroutine in the worker pool prints one line *as the package finishes*. The order of progress lines reflects completion order (not input order); the final table at the end of the run still prints in input order, exactly as today. When `--parallelism > 0`, no progress is emitted — preserving today's behavior for scripted callers.

**Concrete design points:**

- A new boolean carried on `runner` (or threaded as an argument): `emitProgress bool`. Set to `true` in `Run(opts)` when `opts.Parallelism == 0`, false otherwise. Tracking the *original* opts value (not the resolved `effectiveParallelism` result) is essential — otherwise we cannot distinguish "user said 0" from "user said NumCPU explicitly".
- Progress destination: `opts.Out` (same writer the final table uses).
- Progress format (covgate): `[<idx+1>/<total>] <STATUS>  <relPkg>  <elapsed>\n` where `<STATUS>` is `PASS`/`FAIL`/`LOOSE` matching the table status and `<elapsed>` reuses `fmtDuration`. Example: `[3/42] PASS  internal/services/covgate  1.4s`.
- Progress format (covratchet): `[<idx+1>/<total>] <STATUS>  <relPkg>\n` where `<STATUS>` is `NEW`/`UP`/`OK`/`FAIL` matching `ratchetResult`. covratchet doesn't track per-package duration today; do not introduce one here — keep the line short.
- Concurrent writes: multiple workers may finish at once. Wrap the progress writes in a `sync.Mutex` (new field on `runner`, or scoped to `runPackages`). One `Fprintf` call per line; the mutex makes each line atomic.
- A leading line (before the table header) announcing that progress will follow: `Running <total> packages with parallelism=<N>; progress will appear as packages finish:` — printed only when `emitProgress` is true.
- Suppress the existing pre-table announcement and column header in the progress path? **No.** Keep the existing `Checking per-package coverage…` line and the `STATUS COVERAGE …` header. The new progress lines appear *between* the header and the final ordered table. Yes, the user briefly sees two passes (progress, then re-printed-in-order final results), but this is intentional — it gives the user real-time feedback *and* preserves the deterministic, sorted output that scripts may parse.

**File changes:**

1. `internal/services/covgate/covgate.go`:
   - Add `emitProgress bool` and `progressMu sync.Mutex` fields on `runner` (or pass `emitProgress` plus a `*sync.Mutex` through `runPackages` and the closure — local to `runPackages` is fine).
   - In `Run(opts)`, set `emitProgress := opts.Parallelism == 0` and pass it through to the runner.
   - In `runPackages`, when `emitProgress` is true: print the leading line once before the loop, then inside the goroutine — after `r.checkPackage` returns and the result is stored at `results[idx]` — acquire the mutex and emit the progress line using the result's `passed`/`output`/`duration` fields. (Note: the existing `checkResult` already carries `duration`, see line 234.)
   - The format helper for "extract the status keyword from a `checkResult`" is small: if `!res.passed && tests-failed` → `FAIL`; if `!res.passed && tightness-loose` → `LOOSE`; else → `PASS`. Easier: derive status from `res.output` (it's the first whitespace-delimited token of the first line) rather than re-computing.
2. `internal/services/covratchet/covratchet.go`:
   - Same shape, but `runPackages` is inlined into `runner.run` today. Either extract a `runPackages` helper to match covgate's structure, or thread `emitProgress` directly into the existing `for i, pkg := range pkgs` loop. The first is cleaner — extract it for symmetry with covgate, then add progress emission inside the goroutine.
   - covratchet's status keyword comes from `ratchetResult.output` (first whitespace token).
3. No flag changes; the progress behavior is implied by `--parallelism=0`. (Help text already updated to mention "emits progress" in Milestones 2 and 3.)

**Tests:**

In `internal/services/covgate/covgate_test.go`, add two new tests:

4. `TestRun_EmitsProgress_WhenAutoParallelism` — set `Parallelism: 0`, capture stdout via `bytes.Buffer`, set up two packages, assert that the output contains `[1/2]` and `[2/2]` lines *before* the final table rows (use byte offsets to enforce ordering: the leading-line announcement appears before the `STATUS` header, but the `[k/total]` progress lines appear between the header and the final table — so each `[k/total]` index must appear at an offset earlier than the corresponding final-table `PASS` row for the same package).
5. `TestRun_SuppressesProgress_WhenExplicitParallelism` — set `Parallelism: 2`, two packages, assert the output contains **no** `[k/total]` substring.

In `internal/services/covratchet/covratchet_test.go`, mirror with:

6. `TestRun_EmitsProgress_WhenAutoParallelism`.
7. `TestRun_SuppressesProgress_WhenExplicitParallelism`.

Run focused tests and `./scripts/covgate.sh`:

    go test ./internal/services/covgate/... ./internal/services/covratchet/...
    ./scripts/covgate.sh

Expect `ok` and `All packages meet minimum coverage requirement`. If covgate tightness drifts, update the affected `.covgate` file. Commit:

    git add internal/services/covgate internal/services/covratchet
    git commit -m "feat(covgate,covratchet): emit live progress when --parallelism=0"

**Milestone 5 — Final validation.**

1. From `/home/ben/miru/workbench3/repos/gotools`:

       go build ./...
       go test ./...
       ./scripts/covgate.sh
       LINT_FIX=0 ./scripts/lint.sh
       ./scripts/preflight.sh

   Each command must exit 0. `./scripts/covgate.sh` must end with `All packages meet minimum coverage requirement`. `./scripts/lint.sh` must end with `Lint complete`. `./scripts/preflight.sh` must end with `=== All checks passed ===`.

2. Sanity-check the new progress output by hand: from the repo root, run `go run ./cmd/miru covgate` (with default `--parallelism=0`) and confirm progress lines appear interleaved with work. Then run `go run ./cmd/miru covgate -p 2` and confirm no progress lines appear (only the table).

3. Confirm `./scripts/preflight.sh` reports `=== All checks passed ===` before publishing the branch. (No further commit unless validation surfaces a fix.)

## Test steps

The following tests must exist after this change. Each is named by the file it belongs in and the `Test…` function name; run them with the listed `go test` invocation.

**New tests (added in Milestone 4):**

- `internal/services/covgate/covgate_test.go`:
  - `TestRun_EmitsProgress_WhenAutoParallelism` — `Parallelism=0`, two fake packages, assert `[1/2]` and `[2/2]` appear in output before the final ordered table rows.
  - `TestRun_SuppressesProgress_WhenExplicitParallelism` — `Parallelism=2`, two packages, assert no `[` progress prefix appears.
- `internal/services/covratchet/covratchet_test.go`:
  - `TestRun_EmitsProgress_WhenAutoParallelism` — same shape as covgate's.
  - `TestRun_SuppressesProgress_WhenExplicitParallelism` — same shape as covgate's.

**Renamed / updated tests (Milestones 2 and 3):**

- `internal/services/covgate/covgate_test.go`:
  - `TestRun_Parallelism_DefaultsToGOMAXPROCS` → `TestRun_Parallelism_DefaultsToNumCPU` (rename + message update).
  - `TestEffectiveParallelism` — assert default resolves to `runtime.NumCPU()`.
  - `TestChildGOMAXPROCS` — **deleted**.
  - `TestRun_PublicWrapper` and `TestRun_PublicWrapper_HappyPath` — keep, drop any `GOMAXPROCS`-env-specific assertions.
- `internal/services/covratchet/covratchet_test.go`:
  - `TestRun_Parallelism_DefaultsToGOMAXPROCS` → `TestRun_Parallelism_DefaultsToNumCPU`.
  - `TestEffectiveParallelism` — same.
  - `TestChildGOMAXPROCS` — **deleted**.
  - `TestRun_PublicWrapper_HappyPath` — keep, drop GOMAXPROCS-specific assertions.

**Unchanged tests that must keep passing:**

- `internal/services/covgate/covgate_test.go::TestRun_Parallelism` — verifies the final table preserves input order with `Parallelism=2`. This is the load-bearing test that prevents the new progress emission from breaking output ordering.
- `internal/services/covratchet/covratchet_test.go::TestRun_Parallelism` — same role for covratchet.
- `internal/services/gocover/gocover_test.go::TestMeasureWithEnv_AppliesExtraEnv` — keep; `MeasureWithEnv` remains in the public API even though covgate/covratchet no longer call it.

**How to run them:**

    go test ./internal/services/covgate/...
    go test ./internal/services/covratchet/...
    go test ./internal/services/gocover/...
    go test ./...

The last is the full-repo run that CI executes via `./scripts/covgate.sh`.

## Validation

All commands run from `/home/ben/miru/workbench3/repos/gotools`.

- `go build ./...` exits 0.
- `go test ./...` exits 0 with all packages PASS.
- `./scripts/covgate.sh` exits 0 and ends with `All packages meet minimum coverage requirement`.
- `LINT_FIX=0 ./scripts/lint.sh` exits 0 and ends with `Lint complete`.
- `./scripts/preflight.sh` exits 0 and ends with `=== All checks passed ===`.
- Behavioral acceptance (verified by tests, not commands): `TestRun_Parallelism_DefaultsToNumCPU` confirms `--parallelism=0` resolves to `runtime.NumCPU()`. `TestRun_EmitsProgress_WhenAutoParallelism` confirms per-package progress lines appear during a `--parallelism=0` run. `TestRun_SuppressesProgress_WhenExplicitParallelism` confirms progress is silent when the user pinned a worker count. `TestRun_Parallelism` (unchanged) confirms the final ordered table is preserved.
- **Preflight must report `clean` before changes are published.** Do not push the feature branch or open a PR until `./scripts/preflight.sh` ends with `=== All checks passed ===`.

## Idempotence and Recovery

- Every edit is a deterministic source-file change; re-running the same edit produces the same file. Re-running `go test`, `./scripts/covgate.sh`, `LINT_FIX=0 ./scripts/lint.sh`, and `./scripts/preflight.sh` is safe and side-effect-free.
- If a milestone commit fails (lint or test regression), fix forward with a new commit on the same branch — do not amend earlier milestones, so the per-milestone history stays bisectable.
- If the covratchet `.covgate` threshold drifts after Milestone 3, `./scripts/covgate.sh` will fail with a tightness recommendation pointing at the new value; apply that update in a small dedicated commit.
- If progress emission accidentally breaks the existing `TestRun_Parallelism` ordering assertion, the fix is to ensure the *final* `printResults` loop still writes in input order (the change adds progress between the header and the table, not in place of the table). The `[]checkResult` slice is indexed by input position, so as long as `printResults` iterates `results` directly, ordering is preserved.
- The branch can be discarded entirely (`git checkout main`) at any point with no external state to undo — there are no migrations, generated artifacts, or out-of-tree changes.
- If empirical CI timing after this change still shows regression vs. the pre-PR-#28 baseline, the next investigation step is to instrument `./scripts/covgate.sh` to print per-package wall-clock time (Milestone 2 in PR #21 already added wall-clock; verify it's printing) and inspect whether the bottleneck is link, compile, or test-execution. That investigation is out of scope for this plan; capture findings in Surprises & Discoveries and open a follow-up plan.
