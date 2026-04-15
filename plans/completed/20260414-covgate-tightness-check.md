# Add covgate "tightness" check so required thresholds stay close to actual coverage

This ExecPlan is a living document. The sections Progress, Surprises & Discoveries, Decision Log, and Outcomes & Retrospective must be kept up to date as work proceeds.

## Scope

| Repository | Access | Description |
|-----------|--------|-------------|
| `gotools/` | read-write | New tightness check in covgate service, CLI flag wiring, tests |

This plan lives in `gotools/plans/backlog/` because all code changes are in the covgate service and its cobra command, both inside `gotools`.

## Purpose / Big Picture

Covgate's job is to enforce a minimum coverage threshold per package, configured in a `.covgate` file containing a single float (e.g., `61.5`). Today, a package with a `.covgate` of `10.0` and actual coverage of `80.0%` passes silently. That gap hides coverage regressions: if coverage later drops to `15%`, covgate still passes because `15 >= 10`. The required value has drifted too far below reality to act as a guard.

After this change, `miru covgate` fails a package when its required threshold lags the actual measured coverage by more than `0.5` percentage points. A user observes:

    $ miru covgate
    STATUS  COVERAGE  REQUIRED      TIME  PACKAGE
    ------  --------  --------  --------  -------
    LOOSE      80.0%     10.0%      1.2s  internal/foo (required lags actual by 70.0pp; update .covgate to >= 79.5)
    PASS       85.0%     84.5%      0.8s  internal/bar
    PASS       90.0%     89.6%      0.9s  internal/baz

    ERROR: One or more packages failed tests, are below their minimum coverage, or have loose .covgate thresholds
    exit code: 1

The check is always on, with `--tightness-tolerance` (default `0.5`) to tune the allowed gap and `--tightness=false` to disable entirely (escape hatch for debugging).

## Progress

- [ ] Milestone 1 — Add tightness check logic to `checkPackage()` and threaded config through `Opts` and `checkPackageCtx`
- [ ] Milestone 2 — Wire new CLI flags in `internal/commands/covgate.go`
- [ ] Milestone 3 — Update existing tests and add new tests covering tightness behavior
- [ ] Milestone 4 — Update repo's own `.covgate` files if any are now loose; run preflight clean

## Surprises & Discoveries

(Add entries as work proceeds.)

## Decision Log

- Decision: Build the check into covgate itself, not as a separate repo-level test.
  Rationale: The user wants `miru covgate` to flag this everywhere it runs, not only in gotools' own test suite. Keeps one tool responsible for the whole coverage gate contract.
  Date/Author: 2026-04-14, planning subagent.

- Decision: Fire only when `actual - required > tolerance`, using strict greater-than.
  Rationale: Equal-to-tolerance is explicitly allowed ("within 0.5%"). Strict `>` preserves the boundary as passing.
  Date/Author: 2026-04-14, planning subagent.

- Decision: Do not fire when actual is `0`.
  Rationale: A package with no executable tests can legitimately have `required = 0`, `actual = 0`. The gap is `0`, so strict `>` would not fire anyway, but we document the case explicitly to protect against future signed-arithmetic regressions.
  Date/Author: 2026-04-14, planning subagent.

- Decision: Do not fire for packages without an explicit `.covgate` file (i.e., those using `--default-threshold` fallback).
  Rationale: `--default-threshold` is a global floor applied project-wide, not a per-package declaration. Tightening it per-package would silently flag every package not explicitly configured, which is noise rather than signal. Tightness only applies to values an operator deliberately wrote down.
  Date/Author: 2026-04-14, planning subagent.

- Decision: New status label `LOOSE`, not a reused `FAIL`.
  Rationale: Distinguishes "required is too loose" from "coverage is below required". Both cause non-zero exit, but operators need to see at a glance which is which.
  Date/Author: 2026-04-14, planning subagent.

- Decision: Tightness is always on, with `--tightness=false` to disable and `--tightness-tolerance` to tune.
  Rationale: The user's motivation is "we don't want" drift — opt-out is correct. The tolerance flag gives a tuning knob for projects that want tighter or looser bands.
  Date/Author: 2026-04-14, planning subagent.

## Outcomes & Retrospective

(Fill in at completion.)

## Context and Orientation

Covgate is the per-package coverage gate tool invoked as `miru covgate` (cobra command). Key files:

- `internal/services/covgate/covgate.go` — service logic.
  - `Opts` struct: `Packages`, `SrcPrefix`, `TestDir`, `DefaultThreshold`, `Parallelism`, `Out`.
  - `runner` struct with function fields `goModule`, `goListPackages`, `measure` (injected for tests).
  - `run()` → `runPackages()` → `checkPackage()` → `printResults()`.
  - `checkPackageCtx` struct: `module`, `srcPrefix`, `testDir`, `threshold` (the default fallback).
  - `checkResult` struct: `output string`, `passed bool`, `duration time.Duration`.
  - `checkPackage()` computes `threshold := gocover.GetThreshold(pkgDir, ctx.threshold)`, runs measure, and compares `coverage < threshold` to set `FAIL`.
- `internal/services/covgate/covgate_test.go` — tests using `testutil.MakePkgDir` and `testutil.WriteCovgateFile`, with `fakeMeasure(cov float64)` to inject coverage values.
- `internal/services/gocover/gocover.go` — `GetThreshold(pkgDir, defaultThreshold)` reads the `.covgate` file.
- `internal/commands/covgate.go` — cobra wiring. Defines flags on `covgate.Opts`.
- `internal/testutil/testutil.go` — `MakePkgDir`, `WriteCovgateFile` helpers.
- `.covgate` files — per-package config; a single float on line 1, e.g., `61.5`. Many exist under `internal/...`.

Non-obvious terms:

- "tightness" — how close `required` is to `actual`. A tight config tracks reality; a loose one has drifted.
- "pp" (percentage points) — the unit of the gap, e.g., `80.0 - 10.0 = 70.0pp`.

Preflight (`scripts/preflight.sh`) runs `lint.sh`, `covgate.sh`, and `lint-surface.sh` in parallel. "Clean" means all three exit 0 and the script prints `=== All checks passed ===`.

## Plan of Work

### Milestone 1 — Service logic

In `internal/services/gocover/gocover.go`, add `LookupThreshold(pkgDir) (float64, bool)` and refactor `GetThreshold` to delegate:

    func LookupThreshold(pkgDir string) (float64, bool) {
        covFile := filepath.Join(pkgDir, ".covgate")
        data, err := os.ReadFile(covFile)
        if err != nil {
            return 0, false
        }
        line := strings.TrimSpace(strings.Split(string(data), "\n")[0])
        val, err := strconv.ParseFloat(line, 64)
        if err != nil {
            return 0, false
        }
        return val, true
    }

    func GetThreshold(pkgDir string, defaultThreshold float64) float64 {
        if v, ok := LookupThreshold(pkgDir); ok {
            return v
        }
        return defaultThreshold
    }

In `internal/services/covgate/covgate.go`:

- Extend `Opts` with `TightnessEnabled bool` and `TightnessTolerance float64`. Because Go zero-values `bool` to `false`, the cobra layer must explicitly set the default to `true`.
- Extend `checkPackageCtx` with matching fields and thread them from `opts` into the context inside `run()`.
- In `checkPackage()`, call `LookupThreshold` to distinguish explicit `.covgate` from default fallback. After the existing `coverage < threshold` FAIL branch, add:

      if ctx.tightnessEnabled && hasExplicitCovgate && coverage > 0 {
          gap := coverage - threshold
          if gap > ctx.tightnessTolerance {
              // LOOSE row, passed = false
          }
      }

  Output row format (reusing the existing column layout; status column reads `LOOSE`):

      %-6s  %7.1f%%  %7.1f%%  %8s  %s (required lags actual by %.1fpp; update .covgate to >= %.1f)\n

  The recommended floor is `actual - tolerance`, one decimal.

- Update `printResults()` error message from

      ERROR: One or more packages failed tests or are below their minimum coverage

  to

      ERROR: One or more packages failed tests, are below their minimum coverage, or have loose .covgate thresholds

### Milestone 2 — CLI wiring

Edit `internal/commands/covgate.go`:

1. Register the two new flags after the existing ones:

       fl.BoolVar(
           &opts.TightnessEnabled, "tightness", true,
           "fail when required threshold lags actual coverage by more than --tightness-tolerance",
       )
       fl.Float64Var(
           &opts.TightnessTolerance, "tightness-tolerance", 0.5,
           "max allowed gap in percentage points between actual coverage and required threshold",
       )

2. No other changes in this file — `opts` already flows into `covgate.Run(opts)`.

### Milestone 3 — Tests

Edit `internal/services/covgate/covgate_test.go`:

1. Update existing tests that construct `checkPackageCtx` directly to set the new fields. The existing tests (`TestCheckPackage_Pass`, `TestCheckPackage_Fail_BelowThreshold`, etc.) use `checkPackageCtx{module: modName, threshold: 80.0}`. Add `tightnessEnabled: false` to preserve their current semantics, OR update the test fixtures so actual vs required remain within 0.5pp. Prefer `tightnessEnabled: false` where the original intent was to test the FAIL/PASS logic in isolation, and add a separate test that exercises tightness explicitly.

2. Add these new tests:

   - `TestCheckPackage_Loose_Fires`: `.covgate` = `10.0`, fakeMeasure = `80.0`, default = `50`, tightness on, tolerance `0.5`. Expect `!res.passed`, output contains `LOOSE`, `70.0pp`, and `>= 79.5`.

   - `TestCheckPackage_Loose_WithinTolerance`: `.covgate` = `79.6`, fakeMeasure = `80.0`, tolerance `0.5`. Gap is `0.4`. Expect `res.passed` true, output contains `PASS`, no `LOOSE`.

   - `TestCheckPackage_Loose_AtExactTolerance`: `.covgate` = `79.5`, fakeMeasure = `80.0`, tolerance `0.5`. Gap is `0.5`. Expect `res.passed` true (strict `>` boundary).

   - `TestCheckPackage_Loose_JustOverTolerance`: `.covgate` = `79.4`, fakeMeasure = `80.0`, tolerance `0.5`. Gap is `0.6`. Expect `!res.passed` and `LOOSE`.

   - `TestCheckPackage_Loose_ZeroCoverageAllowed`: no `.covgate` OR `.covgate` = `0`, fakeMeasure = `0.0`. Expect `res.passed` true, no `LOOSE` (guard on `coverage > 0`).

   - `TestCheckPackage_Loose_NoCovgateFile_UsesDefault`: no `.covgate` file, fakeMeasure = `80.0`, default threshold = `10.0`, tightness on. Expect `res.passed` true and NO `LOOSE` — the default fallback is a global floor, not a per-package declaration. (This verifies the `LookupThreshold` ok-check correctly skips packages without explicit config.)

   - `TestCheckPackage_Loose_Disabled`: `.covgate` = `10.0`, fakeMeasure = `80.0`, `tightnessEnabled: false`. Expect `res.passed` true, no `LOOSE` in output.

   - `TestCheckPackage_CustomTolerance`: `.covgate` = `70.0`, fakeMeasure = `80.0`, tolerance `15.0`. Gap is `10`, within custom tolerance. Expect `res.passed` true.

   - `TestRun_LooseFailsOverall`: end-to-end through `r.run()`. One package with `.covgate = 10.0` and fakeMeasure = `80.0`. Expect `r.run()` returns non-nil error and `buf.String()` contains both `LOOSE` and the updated `ERROR: ... or have loose .covgate thresholds` message.

3. Add a unit test for `gocover.LookupThreshold` in `internal/services/gocover/gocover_test.go`:

   - `TestLookupThreshold_Present`: write `.covgate` with `61.5`, expect `(61.5, true)`.
   - `TestLookupThreshold_Missing`: empty dir, expect `(0, false)`.
   - `TestLookupThreshold_Malformed`: write `.covgate` with `not-a-number`, expect `(0, false)`.

### Milestone 4 — Repo self-consistency

1. From `gotools/`: run `go run ./cmd/miru covgate` and collect any `LOOSE` rows. For each, update the corresponding `.covgate` file in place to the recommended value (or tighter).

2. If any `.covgate` needs updating, do so manually in this plan — do NOT use `covratchet` here, because covratchet overwrites to `actual` exactly, which would make the next small coverage dip re-fail. Pick a value like `floor(actual*10)/10` (one decimal, rounded down) so there's a tiny natural buffer.

3. Commit the tightened `.covgate` files in the same milestone commit as Milestone 4 (not Milestone 1) to keep the logic change and the config tightening separate in history.

## Concrete Steps

All commands run from the repository root: `/home/ben/miru/workbench1/repos/gotools` (or wherever the clone lives — use `git rev-parse --show-toplevel`).

### Milestone 1 — Service logic

1. Baseline — confirm tests currently pass:

       go test ./internal/services/covgate/... ./internal/services/gocover/...

   Expect `ok  .../covgate` and `ok  .../gocover`.

2. Add `LookupThreshold` in `internal/services/gocover/gocover.go` and refactor `GetThreshold` to call it.

3. Extend `Opts` and `checkPackageCtx` in `internal/services/covgate/covgate.go`; thread fields from `opts` into `checkPackageCtx` inside `run()`.

4. Add the tightness branch in `checkPackage()` after the existing `FAIL` branch. Use `LookupThreshold` to detect explicit `.covgate` presence.

5. Update the error line in `printResults()`.

6. Build to catch compile errors:

       go build ./...

   Expect silent success.

7. Commit:

       git add internal/services/gocover/gocover.go internal/services/covgate/covgate.go
       git commit -m "feat(covgate): add tightness check for required vs actual coverage"

### Milestone 2 — CLI flags

1. Edit `internal/commands/covgate.go` to register `--tightness` and `--tightness-tolerance` flags.

2. Verify the flags appear:

       go run ./cmd/miru covgate --help

   Expect the help output to show `--tightness` and `--tightness-tolerance` with their defaults (`true` and `0.5`).

3. Commit:

       git add internal/commands/covgate.go
       git commit -m "feat(covgate): add --tightness and --tightness-tolerance flags"

### Milestone 3 — Tests

1. Update existing `checkPackageCtx` constructions in `internal/services/covgate/covgate_test.go` to set `tightnessEnabled: false` where preservation of prior semantics is the goal.

2. Add the new test functions listed in Milestone 3 of Plan of Work.

3. Add `gocover.LookupThreshold` tests in `internal/services/gocover/gocover_test.go`.

4. Run the targeted test packages:

       go test ./internal/services/covgate/... ./internal/services/gocover/...

   Expect all prior tests still pass plus the new ones. Look for the new test names in the output when running with `-v`:

       go test -v ./internal/services/covgate/... | grep -E 'TestCheckPackage_Loose|TestRun_LooseFailsOverall'

5. Run the full test suite:

       go test ./...

   Expect all packages `ok`.

6. Commit:

       git add internal/services/covgate/covgate_test.go internal/services/gocover/gocover_test.go
       git commit -m "test(covgate): cover tightness check behavior"

### Milestone 4 — Repo self-consistency + preflight

1. Run covgate against the repo's own packages:

       go run ./cmd/miru covgate

   Read the output. For every `LOOSE` row, note the package and the recommended floor.

2. For each flagged package, update `<pkg>/.covgate`. Choose `floor(actual * 10) / 10` (one decimal, rounded down) to leave a small natural buffer. Example: actual `85.37%` → write `85.3`.

3. Re-run covgate to confirm no `LOOSE` rows remain:

       go run ./cmd/miru covgate

   Expect `All packages meet minimum coverage requirement` and exit 0.

4. Run preflight:

       ./scripts/preflight.sh

   Expect: `=== All checks passed ===` at the end and exit 0. If lint or surface lint fail for unrelated reasons, treat those as findings in Surprises & Discoveries and fix before proceeding.

5. Commit any tightened `.covgate` files:

       git add $(git diff --name-only -- '*/.covgate')
       git commit -m "chore(covgate): tighten .covgate files to match actual coverage"

## Validation and Acceptance

Acceptance criteria, phrased as observable behavior:

1. **Tightness fires above tolerance.** Given a package with `.covgate = 10.0` and actual coverage `80.0%`, `miru covgate` prints a row with status `LOOSE` including the substring `required lags actual by 70.0pp` and exits non-zero. The error summary line includes `have loose .covgate thresholds`.

2. **Tightness tolerates within-band drift.** Given `.covgate = 79.6`, actual `80.0%`, and default tolerance, the row is `PASS` and exit is 0.

3. **Exact-tolerance boundary passes.** Given `.covgate = 79.5`, actual `80.0%`, tolerance `0.5`: gap is exactly `0.5`, row is `PASS`.

4. **Zero-coverage packages are not flagged.** A package with no tests (actual `0%`) and `.covgate = 0` passes.

5. **Packages without `.covgate` are not flagged.** A package that falls back to `--default-threshold` does not get a `LOOSE` row even if actual coverage vastly exceeds the default.

6. **Opt-out works.** `miru covgate --tightness=false` never emits `LOOSE` rows.

7. **Custom tolerance.** `miru covgate --tightness-tolerance=15.0` does not flag a package with `.covgate = 70.0` and actual `80.0%`.

8. **Test suite.** `go test ./...` passes with the new tests included. The new tests (`TestCheckPackage_Loose_Fires`, `TestCheckPackage_Loose_WithinTolerance`, `TestCheckPackage_Loose_AtExactTolerance`, `TestCheckPackage_Loose_JustOverTolerance`, `TestCheckPackage_Loose_ZeroCoverageAllowed`, `TestCheckPackage_Loose_NoCovgateFile_UsesDefault`, `TestCheckPackage_Loose_Disabled`, `TestCheckPackage_CustomTolerance`, `TestRun_LooseFailsOverall`, and the three `TestLookupThreshold_*`) fail before the implementation and pass after.

9. **Preflight clean.** From `gotools/`, running `./scripts/preflight.sh` must print `=== All checks passed ===` and exit 0 before publishing the PR. If preflight is not clean, the plan is not done — fix findings and re-run until clean.

## Idempotence and Recovery

- **Service edits (Milestone 1).** Pure source edits. Safe to repeat by resetting the files and reapplying: `git checkout -- internal/services/covgate/covgate.go internal/services/gocover/gocover.go`.

- **CLI edits (Milestone 2).** Safe to repeat. `go run ./cmd/miru covgate --help` is a read-only verification.

- **Test additions (Milestone 3).** Repeatable. `go test` is idempotent.

- **`.covgate` file tightening (Milestone 4).** Potentially destructive — overwrites existing values. Recovery: these files are checked into git, so `git checkout -- '**/.covgate'` restores prior state. Do not hand-edit below actual coverage (that would make the file fail its own minimum gate immediately).

- **If preflight fails due to unrelated lint issues.** Do not mask them — record in Surprises & Discoveries, fix within this plan or spin off a separate PR. Do not merge with preflight red.
