# Cap child GOMAXPROCS to avoid CI CPU oversubscription

## Scope

- `repos/gotools` (path: `/home/ben/miru/workbench5/repos/gotools`) — read-write. This plan file lives here.
- Base branch: `main`. Feature branch: `perf/covgate-child-gomaxprocs`.

No other repositories are read or written.

## Purpose / Big Picture

The covgate and covratchet services in gotools each spawn one `go test` subprocess per package and run those subprocesses concurrently behind a semaphore. Today the outer parallelism defaults to `runtime.NumCPU()` and each child `go test` inherits the parent's GOMAXPROCS (≈ NumCPU), so on a 4-core CI runner the effective worker thread count is roughly NumCPU × NumCPU ≈ 16 threads competing for 4 CPUs. Because the repo has zero `t.Parallel` callsites, each child test process is essentially single-threaded for the test phase, so the inner GOMAXPROCS only buys parallelism during compile/build.

After this change:

- Outer parallelism defaults to `runtime.GOMAXPROCS(0)` (Go 1.25's cgroup-aware, env-respecting primitive) instead of `runtime.NumCPU()`.
- Each child `go test` subprocess receives `GOMAXPROCS=N` where `N = max(1, runtime.GOMAXPROCS(0) / parallelism)`, so outer × inner ≈ available CPUs.
- The user observes the same external behavior from `./scripts/covgate.sh` and the `covgate` / `covratchet` CLI commands, but CI runs no longer oversubscribe the runner. With `--parallelism=0` on N CPUs, outer=N, inner=1; with explicit `--parallelism=2` on 8 cores, outer=2, inner=4.

## Progress

Add entries as work proceeds.

- [ ] Milestone 1: refactor `gocover.Measure` into `MeasureWithEnv` + thin wrapper, with new test.
- [ ] Milestone 2: update covgate (helpers, default switch, child env injection, test additions/renames, flag help text).
- [ ] Milestone 3: update covratchet (same shape as covgate) and bump `internal/services/covratchet/.covgate` to `99.5`.
- [ ] Milestone 4: full validation (build, test, covgate.sh, lint) and confirm clean preflight.

## Surprises & Discoveries

Add entries as work proceeds.

## Decision Log

Add entries as work proceeds. Authoring decisions already baked into this plan:

- 2026-05-08 — default outer parallelism source switched from `runtime.NumCPU()` to `runtime.GOMAXPROCS(0)`. Rationale: Go 1.25 (the repo's toolchain) makes `GOMAXPROCS(0)` cgroup-aware and env-respecting, which is the right primitive for CI containers.
- 2026-05-08 — child cap is `max(1, runtime.GOMAXPROCS(0) / parallelism)`, injected via subprocess env. Rationale: keeps outer × inner ≈ available CPUs without adding new flags.
- 2026-05-08 — implemented as a new `MeasureWithEnv` plus `Measure` wrapper rather than mutating `Measure`'s signature. Rationale: preserves existing call sites in covratchet's runner binding and gocover tests.

## Outcomes & Retrospective

Add entries as work proceeds.

## Context and Orientation

The reader is assumed to have only the gotools working tree.

**Repository layout (relevant pieces):**

- `internal/services/gocover/gocover.go` — package `gocover`. Exposes `Measure(pkg string, testPaths []string) (float64, []byte, error)` at lines 133-157. It builds args `go test -coverprofile=<tmp> -coverpkg=<pkg> <testPaths...>`, calls `cmdutil.GoCommand(args...)` (which returns an `*exec.Cmd` with `Env = append(os.Environ(), "GOWORK=off")`), then runs `testCmd.CombinedOutput()`. No callers of `Measure` exist outside of covgate, covratchet, and their tests.
- `internal/services/covgate/covgate.go` — package `covgate`. `Run(opts) error` at line 34 constructs a `runner` and calls `r.run(opts)` at line 43. The runner struct is at lines 27-31 and contains `measure func(pkg string, testPaths []string) (float64, []byte, error)`. The default-parallelism block is at line 51 (`if parallelism <= 0 { parallelism = runtime.NumCPU() }`). `r.runPackages` is at line 87 and the semaphore is created at line 91.
- `internal/services/covratchet/covratchet.go` — package `covratchet`. Same shape: `Run(opts)` at line 31, `r.run(opts)` at line 40, parallelism default at line 48, semaphore inlined at lines 67-79. Runner struct at lines 24-28 with the same `measure` signature as covgate.
- `internal/commands/covgate.go` and `internal/commands/covratchet.go` — Cobra command wiring for the two services. Each has a `--parallelism` flag with help text `(0 = NumCPU)` that needs updating.
- `internal/testutil/` — provides `MakePkgDir(t, rel)` and `WriteCovgateFile(t, dir, val)` helpers used by tests.
- `internal/services/gocover/gocover_test.go` — has `makeGoProject(t *testing.T)` at lines 193-213 that creates a tiny temp module (`go.mod` declaring `module testmod` with `go 1.23`, plus `mypkg/lib.go` and `mypkg/lib_test.go`) and chdirs into the temp dir. `TestMeasure` at lines 279-289 and `TestMeasure_TestFailure` are the model for the new `TestMeasureWithEnv_AppliesExtraEnv`.
- `internal/services/covgate/covgate_test.go` — has `fakeMeasure(cov float64) func(string, []string) (float64, []byte, error)` at lines 38-39. `TestRun_Parallelism_DefaultsToNumCPU` is at line 218 with the inline error message at line 234 (must be renamed to `TestRun_Parallelism_DefaultsToGOMAXPROCS` and have its message updated).
- `internal/services/covratchet/covratchet_test.go` — `fakeMeasure` at lines 93-94. `TestRun_Parallelism_DefaultsToNumCPU` at line 389 with the message at line 405 (same rename treatment).

**Coverage thresholds (the project enforces these via covgate):**

- `internal/services/gocover/.covgate` = `92.1`
- `internal/services/covgate/.covgate` = `100.0`
- `internal/services/covratchet/.covgate` = `97.4` — will change to `99.5` (see milestone 3).

**CI:** `.github/workflows/ci.yml` runs `./scripts/covgate.sh` and `LINT_FIX=0 ./scripts/lint.sh` on `ubuntu-latest-m`.

**Lint:** `LINT_FIX=0 ./scripts/lint.sh` invokes `go run ./cmd/miru lint --paths=internal --exclude=nofmt,bgctx --fix=false`. The custom **collapse** linter (in `internal/services/lint/linter/collapse/`) flags multi-line literals or signatures that would fit on one line. Two collapse-relevant pitfalls in this change:

- The extraEnv slice literal must be one line: `extraEnv := []string{fmt.Sprintf("GOMAXPROCS=%d", childGOMAXPROCS(parallelism))}`.
- The closure signature must stay on one line: `measure: func(pkg string, testPaths []string) (float64, []byte, error) { ... }`.

**Why this change is needed (problem statement):** outer parallelism `runtime.NumCPU()` × inherited inner GOMAXPROCS ≈ NumCPU² on a constrained CI runner. With zero `t.Parallel` callsites in the repo, the inner GOMAXPROCS does no test-execution work; it only matters for compile/build. Capping it lets us collapse the product back to ≈ available CPUs without giving up build parallelism on machines with explicit per-process capacity.

## Plan of Work

The work is structured as four milestones, each ending in one git commit.

**Milestone 1 — Refactor `gocover.Measure`.** In `internal/services/gocover/gocover.go`, introduce a new exported function:

    func MeasureWithEnv(pkg string, testPaths []string, extraEnv []string) (float64, []byte, error)

`MeasureWithEnv` does what today's `Measure` does, except after constructing the `*exec.Cmd` from `cmdutil.GoCommand(args...)` it appends `extraEnv` to `testCmd.Env`. Reduce the existing `Measure` to a thin wrapper:

    func Measure(pkg string, testPaths []string) (float64, []byte, error) {
        return MeasureWithEnv(pkg, testPaths, nil)
    }

This preserves every existing caller (covratchet's runner.measure assignment, gocover_test.go's `TestMeasure` and `TestMeasure_TestFailure`) without signature changes.

In `internal/services/gocover/gocover_test.go`, add `TestMeasureWithEnv_AppliesExtraEnv`. Use `makeGoProject(t)` to set up a tiny temp module, then call `MeasureWithEnv("./mypkg", []string{"./mypkg"}, []string{"GOMAXPROCS=1"})` and assert the call succeeds and returns a coverage value. The test exists primarily to cover the new branch (the `.covgate` threshold is `92.1`).

**Milestone 2 — Update covgate.** In `internal/services/covgate/covgate.go`, add two package-private helpers near the top of the file:

    func effectiveParallelism(opts Opts) int {
        if opts.Parallelism > 0 {
            return opts.Parallelism
        }
        return runtime.GOMAXPROCS(0)
    }

    func childGOMAXPROCS(parallelism int) int {
        n := runtime.GOMAXPROCS(0) / parallelism
        if n < 1 {
            return 1
        }
        return n
    }

In `Run(opts)` (line 34), compute the parallelism and the extraEnv once, then bind `runner.measure` to a closure that calls `gocover.MeasureWithEnv` with that extraEnv. The closure signature must stay on one line (collapse linter):

    parallelism := effectiveParallelism(opts)
    extraEnv := []string{fmt.Sprintf("GOMAXPROCS=%d", childGOMAXPROCS(parallelism))}
    r := &runner{
        measure: func(pkg string, testPaths []string) (float64, []byte, error) { return gocover.MeasureWithEnv(pkg, testPaths, extraEnv) },
        // ...other existing fields...
    }

In `r.run(opts)`, replace the inline `if parallelism <= 0 { parallelism = runtime.NumCPU() }` (line 51) with `parallelism := effectiveParallelism(opts)`. The semaphore at line 91 then uses this value.

In `internal/commands/covgate.go`, update the `--parallelism` flag help text from `(0 = NumCPU)` to `(0 = GOMAXPROCS)`.

In `internal/services/covgate/covgate_test.go`:

- Add `TestEffectiveParallelism` (assert that an explicit value passes through unchanged and that 0 yields a positive default).
- Add `TestChildGOMAXPROCS` (assert oversubscribed parallelism returns 1; parallelism=1 returns ≥ 1).
- Add `TestRun_PublicWrapper_PassesThrough` — exercise the closure body in `Run` end-to-end against a real temp Go module (use the `makeGoProject` pattern from gocover_test.go, or similar). This is required because covgate's `.covgate` threshold is `100.0`, so the new closure body must be covered.
- Rename `TestRun_Parallelism_DefaultsToNumCPU` (line 218) to `TestRun_Parallelism_DefaultsToGOMAXPROCS`. Update the inline error message at line 234 from "NumCPU" to "GOMAXPROCS".

**Milestone 3 — Update covratchet.** Apply structurally identical changes to `internal/services/covratchet/covratchet.go`: add the same two helpers (`effectiveParallelism`, `childGOMAXPROCS`) — they may be duplicated; the change is intentionally scoped per-service. Compute `parallelism` and `extraEnv` once in `Run(opts)` and bind `runner.measure` to a one-line closure that calls `gocover.MeasureWithEnv`. Replace the inline default in `r.run` (line 48) with `effectiveParallelism(opts)`. The semaphore at lines 67-79 then uses the result.

In `internal/commands/covratchet.go`, update the `--parallelism` flag help text from `(0 = NumCPU)` to `(0 = GOMAXPROCS)`.

In `internal/services/covratchet/covratchet_test.go`, add `TestEffectiveParallelism`, `TestChildGOMAXPROCS`, and `TestRun_PublicWrapper_PassesThrough` mirroring the covgate additions. Rename `TestRun_Parallelism_DefaultsToNumCPU` (line 389) to `TestRun_Parallelism_DefaultsToGOMAXPROCS` and update the message at line 405.

After the test additions, covratchet's measured coverage rises to ~100%. The current `.covgate` value of `97.4` then trips covgate's tightness check (which requires the recorded threshold to be within 0.5pp of measured). Update `internal/services/covratchet/.covgate` from `97.4` to `99.5` (the value covgate's own tightness recommendation suggests).

**Milestone 4 — Final validation.** Run the full validation suite (build, test, covgate.sh, lint) from inside `repos/gotools` and confirm preflight reports `clean` before publishing.

## Concrete Steps

All commands run from `/home/ben/miru/workbench5/repos/gotools` unless stated otherwise.

**Milestone 1 — Refactor `gocover.Measure`.**

1. Edit `internal/services/gocover/gocover.go`: introduce `MeasureWithEnv(pkg, testPaths, extraEnv)`; rewrite `Measure` as `return MeasureWithEnv(pkg, testPaths, nil)`.
2. Edit `internal/services/gocover/gocover_test.go`: add `TestMeasureWithEnv_AppliesExtraEnv` using `makeGoProject(t)`.
3. Run focused tests:

       go test ./internal/services/gocover/...

   Expect `ok` for the package.

4. Commit (from `/home/ben/miru/workbench5/repos/gotools`):

       git add internal/services/gocover/gocover.go internal/services/gocover/gocover_test.go
       git commit -m "refactor(gocover): add MeasureWithEnv for subprocess env injection"

**Milestone 2 — Update covgate.**

1. Edit `internal/services/covgate/covgate.go`: add `effectiveParallelism` and `childGOMAXPROCS`. In `Run(opts)`, build `extraEnv` (one-line literal) and bind `runner.measure` to a one-line closure calling `gocover.MeasureWithEnv`. In `r.run(opts)`, replace the line-51 default block with `parallelism := effectiveParallelism(opts)`.
2. Edit `internal/commands/covgate.go`: update flag help text from `(0 = NumCPU)` to `(0 = GOMAXPROCS)`.
3. Edit `internal/services/covgate/covgate_test.go`:
   - Add `TestEffectiveParallelism`.
   - Add `TestChildGOMAXPROCS`.
   - Add `TestRun_PublicWrapper_PassesThrough` (real temp Go module, exercises the closure body).
   - Rename `TestRun_Parallelism_DefaultsToNumCPU` (line 218) to `TestRun_Parallelism_DefaultsToGOMAXPROCS` and update the message at line 234.
4. Run focused tests:

       go test ./internal/services/covgate/...

   Expect `ok`.

5. Sanity-check the package's covgate threshold:

       ./scripts/covgate.sh

   Expect `All packages meet minimum coverage requirement`.

6. Commit:

       git add internal/services/covgate/covgate.go internal/services/covgate/covgate_test.go internal/commands/covgate.go
       git commit -m "perf(covgate): default to GOMAXPROCS and cap child GOMAXPROCS"

**Milestone 3 — Update covratchet.**

1. Edit `internal/services/covratchet/covratchet.go`: add `effectiveParallelism` and `childGOMAXPROCS` helpers; build `extraEnv` once in `Run(opts)`; bind `runner.measure` to a one-line closure calling `gocover.MeasureWithEnv`; replace the default block in `r.run` (line 48) with `parallelism := effectiveParallelism(opts)`.
2. Edit `internal/commands/covratchet.go`: update flag help text from `(0 = NumCPU)` to `(0 = GOMAXPROCS)`.
3. Edit `internal/services/covratchet/covratchet_test.go`:
   - Add `TestEffectiveParallelism`.
   - Add `TestChildGOMAXPROCS`.
   - Add `TestRun_PublicWrapper_PassesThrough`.
   - Rename `TestRun_Parallelism_DefaultsToNumCPU` (line 389) to `TestRun_Parallelism_DefaultsToGOMAXPROCS` and update the message at line 405.
4. Update the threshold file `internal/services/covratchet/.covgate` from `97.4` to `99.5`.
5. Run focused tests and the threshold script:

       go test ./internal/services/covratchet/...
       ./scripts/covgate.sh

   Expect `ok` and `All packages meet minimum coverage requirement`.

6. Commit:

       git add internal/services/covratchet/covratchet.go internal/services/covratchet/covratchet_test.go internal/services/covratchet/.covgate internal/commands/covratchet.go
       git commit -m "perf(covratchet): default to GOMAXPROCS and cap child GOMAXPROCS"

**Milestone 4 — Final validation.**

1. From `/home/ben/miru/workbench5/repos/gotools`:

       go build ./...
       go test ./...
       ./scripts/covgate.sh
       LINT_FIX=0 ./scripts/lint.sh

   Each command must exit 0. The covgate script must end with `All packages meet minimum coverage requirement`. The lint script must end with `Lint complete`.

2. Confirm preflight reports `clean` before publishing the branch. (No further commit unless the validation surfaces a fix; if it does, that fix lands as its own commit on the branch.)

## Validation and Acceptance

All commands run from `/home/ben/miru/workbench5/repos/gotools`.

- `go build ./...` exits 0.
- `go test ./...` exits 0 with all packages PASS.
- `./scripts/covgate.sh` exits 0 and ends with `All packages meet minimum coverage requirement`.
- `LINT_FIX=0 ./scripts/lint.sh` exits 0 and ends with `Lint complete`.
- Behavioral acceptance: with `--parallelism=0` on an N-CPU runner, the outer semaphore is sized to `runtime.GOMAXPROCS(0)` and each spawned `go test` subprocess receives `GOMAXPROCS=1` in its environment. With `--parallelism=2` on an 8-CPU runner, child subprocesses receive `GOMAXPROCS=4`. (The `MeasureWithEnv` test plus the public-wrapper end-to-end tests cover this contract.)
- Preflight must report `clean` before changes are published. Do not push the feature branch or open a PR until preflight is clean.

## Idempotence and Recovery

- Every edit is a deterministic source-file change; re-running the same edit produces the same file. Re-running `go test`, `./scripts/covgate.sh`, and `LINT_FIX=0 ./scripts/lint.sh` is safe and side-effect-free.
- If a milestone commit fails (lint or test regression), fix forward with a new commit on the same branch — do not amend earlier milestones, so the per-milestone history stays bisectable.
- If the covratchet `.covgate` threshold update is forgotten, `./scripts/covgate.sh` will fail with a tightness recommendation pointing at `99.5`; apply that update and re-run.
- If `MeasureWithEnv` is introduced but `Measure` is not preserved as a wrapper, existing tests in covratchet and gocover will fail to compile; restoring the wrapper fixes it without touching call sites.
- The branch can be discarded entirely (`git checkout main`) at any point with no external state to undo — there are no migrations, generated artifacts, or out-of-tree changes.
