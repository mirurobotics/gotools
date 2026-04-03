---
name: hunt-reliability
description: Proactively scan a codebase for reliability issues, prioritize by failure impact, then hand off the most critical one to $patch for hardening. Use when asked to find reliability problems, audit error handling, or improve failure resilience.
---

# Reliability Hunt Workflow

## Inputs
- `repo` (optional): constrain scope to one repo path (e.g. `backend`, `frontend`, `core`, or `.` for root).
- `scope` (optional): files, package, module, or full repo. Defaults to full repo.
- `severity` (optional): minimum severity to act on — `critical`, `high`, `medium`. Defaults to `high`.
- `base` (optional): target branch for the fix. Defaults to `main`.

## References
Use these skills as authoritative lenses during analysis:
- `$reliability` for error handling, resource lifecycle, cancellation, timeouts, and retry analysis.
- `$review` for correctness cross-check.
- `$test` for writing failure-demonstrating tests.
- `$commit` for commit conventions.
- `$patch` for executing the fix.

## Procedure

### 1. Analyze the codebase for reliability issues
- Read project structure, entrypoints, and reliability-sensitive paths (I/O operations, external calls, resource acquisition, concurrent code, shutdown sequences).
- Systematically scan using the analysis approach from `$reliability`:
  - Resource leaks (file handles, connections, goroutines, event listeners not cleaned up).
  - Swallowed or silently dropped errors.
  - Missing or unbounded timeouts and retries.
  - Cancellation signals not propagated (context, AbortController, CancellationToken).
  - Missing graceful shutdown or cleanup on process exit.
  - Non-idempotent operations in retry paths.
  - Deadlocks, livelocks, and unbounded queue growth.
  - Missing circuit breakers on external dependencies.
- Produce a ranked issue list sorted by failure impact.

### 2. Select the most severe issue
- Pick the highest-severity issue from the list.
- Present the full issue list to the user, indicate which will be fixed, and proceed immediately.
- If no issues are found at or above the minimum severity, report that and stop.

### 3. Write failure-demonstrating tests
- Create a branch from latest `base`: `fix/<short-reliability-description>`.
- Write test cases that trigger the failure mode (error injection, forced timeout, simulated resource exhaustion).
- Tests must fail on the current code, demonstrating the reliability gap.
- Run the tests and **capture the failure output**.
- Commit the failing tests: `test: add failure tests for <reliability issue description>`.
- Follow test conventions from `$test`.

### 4. Hand off to $patch
- Invoke `$patch` with:
  - `task`: what the reliability issue is, its failure impact, root cause, file paths/line references, and what the hardening should look like.
  - `repo` and `base`: pass through from inputs.
  - `branch`: the branch created in step 3, so `$patch` continues on it.
  - `evidence`: the captured test failure output from step 3.

## Severity Definitions
- **critical**: Resource leak or error swallowing that causes cascading failure, OOM, or data loss under sustained load.
- **high**: Reliability issue that manifests under realistic production conditions — missing timeout on external call, leaked connection on error path, non-idempotent retry.
- **medium**: Reliability weakness that manifests under unusual but possible conditions — missing cleanup on rare error path, unbounded retry that could theoretically exhaust resources.

## Safety Rules
- Never force push.
- Never push directly to protected branches (`main`, `production`, `uat`, `release/*`).
- Never commit unrelated changes alongside the fix.
- Stop and report if no issues meet the severity threshold.

## Output Contract
1. **Reliability issue inventory**: ranked list with failure impact and trigger conditions for each.
2. **Selected issue**: which one is being fixed and why it ranks highest.
3. **Test evidence**: failure test output proving the issue exists.
4. **Patch result**: output from `$patch` (task summary, changes, verification, PR link).
