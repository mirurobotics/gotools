# Fix covgate LOOSE recommendation rounding bug

This ExecPlan is a living document. The sections Progress, Surprises & Discoveries, Decision Log, and Outcomes & Retrospective must be kept up to date as work proceeds.

## Scope

| Repository | Access | Description |
|-----------|--------|-------------|
| `gotools` | read-write | `internal/services/covgate/covgate.go` and `covgate_test.go` |

This plan lives in `gotools/plans/backlog/` because the bug and fix are entirely within this repository.

## Purpose / Big Picture

When covgate's tightness check fires (the LOOSE result), it prints a recommended threshold value using `%.1f` formatting. This formatting can round the recommendation **down**, which means a user who follows the suggestion and sets their `.covgate` file to the printed value may still fail the tightness check on the next run.

Example: coverage=80.0, tolerance=0.06 produces recommended=79.94. Formatting with `%.1f` prints `79.9`. The user sets 79.9 in `.covgate`, but the gap is now 80.0 - 79.9 = 0.1, which exceeds the 0.06 tolerance. The suggestion itself is wrong.

After this fix, the recommended value is always ceiled to 1 decimal place before formatting, guaranteeing the printed suggestion passes the tightness check.

## Progress

- [ ] Move plan to `plans/active/`
- [ ] Implement fix in `covgate.go`
- [ ] Add regression test in `covgate_test.go`
- [ ] Run preflight (lint + tests)
- [ ] Commit

## Surprises & Discoveries

(Updated during execution.)

## Decision Log

- Decision: Use `math.Ceil(recommended*10) / 10` to round up to 1 decimal place.
  Rationale: This is the simplest correct fix. It guarantees the formatted value is always >= the exact recommended value, so the suggestion always passes the tightness check. It requires adding `"math"` to the import block.
  Date/Author: 2026-04-14 / agent

## Outcomes & Retrospective

(Fill in at completion.)

## Context and Orientation

The tightness check lives in `internal/services/covgate/covgate.go`, function `checkPackage`, lines 196-208. When a package has an explicit `.covgate` file and the gap between actual coverage and the threshold exceeds `tightnessTolerance`, the check fails with a LOOSE result and prints a recommended threshold.

The relevant code path:

```go
// line 196-208
if ctx.tightnessEnabled && hasExplicitCovgate && coverage > 0 {
    gap := coverage - threshold
    if gap > ctx.tightnessTolerance {
        recommended := coverage - ctx.tightnessTolerance
        _, _ = fmt.Fprintf(
            &b, "%-6s  %7.1f%%  %7.1f%%  %8s  "+
                "%s (required lags actual by %.1fpp; "+
                "update .covgate to >= %.1f)\n",
            "LOOSE", coverage, threshold, fmtDuration(elapsed),
            relPkg, gap, recommended,
        )
        return checkResult{b.String(), false, elapsed}
    }
}
```

The existing test `TestCheckPackage_Loose_Fires` (line 449) uses tolerance=0.5 with coverage=80.0, yielding recommended=79.5 -- an exact value that does not expose the rounding bug. A new test with a tolerance that produces a non-exact decimal (e.g., 0.06) is needed.

The file currently does not import `"math"`.

**Preflight**: CI runs `./scripts/lint.sh` and `./scripts/covgate.sh`. Locally, these same scripts serve as preflight checks.

## Plan of Work

This is a two-file, single-milestone fix.

### Milestone 1: Fix and test

1. **Edit `covgate.go`**: Add `"math"` to the import block. Before the `Fprintf` call, ceil `recommended` to 1 decimal place:
   ```go
   recommended = math.Ceil(recommended*10) / 10
   ```

2. **Edit `covgate_test.go`**: Add `TestCheckPackage_Loose_RecommendationRounding` that exercises the rounding edge case:
   - coverage=80.0, threshold=10.0 (from `.covgate` file), tolerance=0.06
   - gap = 70.0 > 0.06, so LOOSE fires
   - recommended = 80.0 - 0.06 = 79.94 -- without the fix, `%.1f` would print `79.9`
   - With the fix, `math.Ceil(79.94*10)/10 = 80.0`, so the output should contain `>= 80.0`
   - Assert output contains `"LOOSE"` and `">= 80.0"`

3. **Run preflight**: `./scripts/lint.sh` and `./scripts/covgate.sh` from the repo root.

4. **Commit**: Commit both files with a message describing the fix.

## Concrete Steps

All paths are relative to `/home/ben/miru/workbench1/repos/gotools/`.

1. Move plan: `mv plans/backlog/20260414-covgate-recommendation-rounding.md plans/active/`

2. Edit `internal/services/covgate/covgate.go`:
   - Add `"math"` to the import block (between `"io"` and `"os"` alphabetically).
   - After `recommended := coverage - ctx.tightnessTolerance` (line 199), add:
     ```go
     recommended = math.Ceil(recommended*10) / 10
     ```

3. Edit `internal/services/covgate/covgate_test.go`:
   - Add a new test function `TestCheckPackage_Loose_RecommendationRounding` after the existing LOOSE tests (after line ~475). Model it on `TestCheckPackage_Loose_Fires` but use `tightnessTolerance: 0.06` and assert the output contains `">= 80.0"` (not `79.9`).

4. Run preflight:
   ```bash
   cd /home/ben/miru/workbench1/repos/gotools
   ./scripts/lint.sh
   ./scripts/covgate.sh
   ```

5. Commit both changed files via `$commit`.

## Validation and Acceptance

- `go test ./internal/services/covgate/...` passes, including the new `TestCheckPackage_Loose_RecommendationRounding` test.
- `./scripts/lint.sh` exits 0.
- `./scripts/covgate.sh` exits 0.
- Preflight must report clean before changes are published (pushed or PR opened).
- Manual review: the LOOSE output for tolerance=0.06 and coverage=80.0 prints `>= 80.0`, not `>= 79.9`.

## Idempotence and Recovery

Both edits are simple file changes. Re-running is safe -- the old content can be restored from git (`git checkout -- internal/services/covgate/covgate.go internal/services/covgate/covgate_test.go`). The plan can be re-executed from any step without side effects.
