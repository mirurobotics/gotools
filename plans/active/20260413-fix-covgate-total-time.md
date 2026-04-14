# Fix covgate "Total time" to report wall-clock time instead of summed durations

This ExecPlan is a living document. The sections Progress, Surprises & Discoveries, Decision Log, and Outcomes & Retrospective must be kept up to date as work proceeds.

## Scope

| Repository | Access | Description |
|-----------|--------|-------------|
| `gotools/` | read-write | Bug fix in covgate service source and tests |

## Purpose / Big Picture

The covgate service runs per-package test coverage checks in parallel using goroutines. After all packages finish, it prints a "Total time" line. Currently this total is computed by summing every individual package duration. When packages run concurrently, this sum overstates the actual elapsed time. For example, four packages each taking 3 seconds in parallel report "Total time: 12.0s" instead of the correct wall-clock time of roughly 3 seconds.

After this fix, the "Total time" line will report the true wall-clock time from when package execution started to when all packages finished. Individual per-package TIME values remain unchanged.

## Progress

- [ ] Milestone 1 -- Measure wall-clock time in `run()` and pass it to `printResults()`
- [ ] Milestone 2 -- Update `printResults()` to accept and use the wall-clock duration
- [ ] Milestone 3 -- Update tests to verify the new behavior
- [ ] Milestone 4 -- Full test suite green, ready for commit

## Surprises & Discoveries

_Nothing discovered yet._

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-13 | Measure wall time around `runPackages()` in `run()`, not inside `runPackages()` itself | Keeps `runPackages()` focused on orchestration; `run()` is the natural caller that owns the full lifecycle |
| 2026-04-13 | Pass wall-clock duration as parameter to `printResults()` rather than storing it on the runner struct | Avoids mutable state on the runner; the duration is a single value used in one place |
| 2026-04-13 | Keep `duration` field on `checkResult` struct unchanged | Per-package durations are still printed in each row and remain useful |

## Outcomes & Retrospective

_Fill in after the work is merged._

## Context and Orientation

### Key types and functions

- `runner` (struct): holds function fields `goModule`, `goListPackages`, `measure`. Methods: `run()`, `runPackages()`, `checkPackage()`, `printResults()`.
- `checkResult` (struct): holds `output` (string), `passed` (bool), `duration` (time.Duration) for each package.
- `run()`: entry point. Calls `runPackages()` then `printResults()`.
- `runPackages()`: launches goroutines with a semaphore for concurrency, returns `[]checkResult`.
- `printResults()`: iterates results, prints output, sums durations (the bug), prints total.

### The bug

In `printResults()`, the total time is computed as `total += res.duration` for each result. Since `runPackages()` runs packages concurrently, `total` is the sum of all individual durations, not the wall-clock time.

## Plan of Work

### Step 1 -- Baseline: confirm tests pass

    go test ./internal/services/covgate/...

### Step 2 -- Add wall-clock timing in run()

Replace:

    results := r.runPackages(pkgs, ctx, parallelism)
    return r.printResults(w, results)

With:

    start := time.Now()
    results := r.runPackages(pkgs, ctx, parallelism)
    wallTime := time.Since(start)
    return r.printResults(w, results, wallTime)

### Step 3 -- Update printResults() signature and body

Change signature to accept `totalTime time.Duration`. Remove `var total time.Duration` and `total += res.duration`. Use `totalTime` in the format call.

### Step 4 -- Add new test TestPrintResults_UsesWallTime

Constructs three results each with 5s duration, passes 3s as wall time, confirms output says "Total time: 3.0s" (not "15.0s").

### Step 5 -- Run full test suite

    go test ./...

### Step 6 -- Commit

## Validation and Acceptance

1. `go test ./internal/services/covgate/...` reports `ok` with zero failures.
2. The new `TestPrintResults_UsesWallTime` test passes.
3. `go test ./...` reports `ok` for every package.
4. Preflight must report `clean` before changes are published.
