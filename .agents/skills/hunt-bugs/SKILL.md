---
name: hunt-bugs
description: Analyze a codebase for bugs, prioritize by severity, then hand off the most critical one to $patch for fixing. Use when asked to find bugs, hunt for issues, or proactively improve code correctness.
---

# Bug Hunt Workflow

## Inputs
- `repo` (optional): constrain scope to one repo path (e.g. `backend`, `frontend`, `core`, or `.` for root).
- `scope` (optional): files, package, module, or full repo. Defaults to full repo.
- `severity` (optional): minimum severity to act on — `critical`, `high`, `medium`. Defaults to `high`.
- `base` (optional): target branch for the fix. Defaults to `main`.

## References
Use these skills as authoritative lenses during analysis:
- `$review` for correctness and test coverage analysis.
- `$security` for security-sensitive bugs.
- `$reliability` for error handling, lifecycle, and failure-mode bugs.
- `$test` for writing failing tests.
- `$commit` for commit conventions.
- `$patch` for executing the fix.

## Procedure

### 1. Analyze the codebase for bugs
- Read project structure, entrypoints, and high-risk areas (error handling, state mutations, boundary parsing, concurrency, external I/O).
- Systematically scan for bugs using the correctness analysis approach from `$review`:
  - Logic errors, broken invariants, state corruption, data loss paths.
  - Boundary value mishandling (off-by-one, nil/null, empty collections, overflow).
  - Error paths that swallow errors, double-wrap, or silently degrade.
  - Race conditions, missing synchronization, resource leaks.
  - Security issues (injection, auth bypass, unsafe deserialization).
- Produce a ranked bug list sorted by severity.

### 2. Select the most severe bug
- Pick the highest-severity bug from the list.
- Present the full bug list to the user, indicate which bug will be fixed, and proceed immediately.
- If no bugs are found at or above the minimum severity, report that and stop.

### 3. Write failing tests
- Create a branch from latest `base`: `fix/<short-bug-description>`.
- Write test cases that **demonstrate the bug is real** — they must fail on the current code.
- Cover the exact failure mode, boundary conditions that trigger it, and at least one regression guard.
- Follow test conventions from `$test`.
- Run the tests and **capture the failure output** (command, test names, and failure messages).
- If the tests pass, the bug hypothesis is wrong — revisit step 2.
- Commit the failing tests: `test: add failing tests for <bug description>`.

### 4. Hand off to $patch
- Invoke `$patch` with:
  - `task`: what the bug is, its severity/impact, root cause, file paths/line references, and what the fix should look like.
  - `repo` and `base`: pass through from inputs.
  - `branch`: the branch already created in step 3 (so $patch continues on it rather than creating a new one).
  - `evidence`: the captured test failure output from step 3, so it appears in the PR for reviewers.

## Severity Definitions
- **critical**: Data loss, security vulnerability, crash in production, or silent data corruption.
- **high**: Incorrect behavior under realistic conditions, broken error handling, resource leaks.
- **medium**: Edge-case failures, degraded behavior under unusual but possible conditions.

## Safety Rules
- Never force push.
- Never push directly to protected branches (`main`, `production`, `uat`, `release/*`).
- Stop and report if no bugs meet the severity threshold.

## Output Contract
1. **Bug inventory**: ranked list of discovered bugs with severity, location, and impact.
2. **Selected bug**: which bug is being fixed and why it ranks highest.
3. **Test evidence**: test names, commands run, and failure output proving the bug exists.
4. **Patch result**: output from `$patch` (task summary, changes, verification, PR link).
