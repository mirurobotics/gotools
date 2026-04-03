---
name: hunt-tests
description: Proactively scan a codebase for test coverage gaps, prioritize by risk of undetected regression, then hand off the most critical gap to $patch for coverage. Use when asked to find untested code, improve coverage, or audit test quality.
---

# Test Coverage Hunt Workflow

## Inputs
- `repo` (optional): constrain scope to one repo path (e.g. `backend`, `frontend`, `core`, or `.` for root).
- `scope` (optional): files, package, module, or full repo. Defaults to full repo.
- `severity` (optional): minimum severity to act on — `critical`, `high`, `medium`. Defaults to `high`.
- `base` (optional): target branch for the fix. Defaults to `main`.

## References
Use these skills as authoritative lenses during analysis:
- `$test` for coverage philosophy, test design, and language-specific conventions.
- `$review` for identifying which behaviors are most important to test.
- `$security` and `$reliability` for identifying high-risk untested paths.
- `$commit` for commit conventions.
- `$patch` for executing the fix.

## Procedure

### 1. Analyze the codebase for test coverage gaps
- Read project structure, identify core business logic, critical paths, and high-risk areas.
- Systematically scan for coverage gaps using the analysis approach from `$test`:
  - Public APIs and exported functions with no corresponding test file or test cases.
  - Error and failure paths with no test coverage (error returns, exception handlers, fallback logic).
  - Boundary conditions without tests (empty input, nil/null, max values, concurrent access).
  - Complex conditional logic with only happy-path tests.
  - Recent changes or high-churn files with weak or no test coverage.
  - Weak assertions (testing implementation details instead of observable behavior).
- Produce a ranked gap list sorted by regression risk.

### 2. Select the most critical gap
- Pick the highest-severity gap from the list.
- Present the full gap list to the user, indicate which will be addressed, and proceed immediately.
- If no gaps are found at or above the minimum severity, report that and stop.

### 3. Hand off to $patch
- Invoke `$patch` with:
  - `task`: a concrete test plan describing what tests to write — target behavior, test cases (happy path, error path, boundary cases), expected assertions, and file placement. The tests themselves are the deliverable.
  - `repo` and `base`: pass through from inputs.
  - `evidence`: description of the coverage gap — what code is untested, why it matters, and what could regress silently.

## Severity Definitions
- **critical**: Untested code that handles money, authentication, authorization, data mutations, or cryptographic operations.
- **high**: Untested error paths or boundary conditions in core business logic — a regression here would cause incorrect behavior under realistic conditions.
- **medium**: Untested happy-path variations or edge cases in non-critical code — a regression here would cause degraded behavior under unusual conditions.

## Safety Rules
- Never force push.
- Never push directly to protected branches (`main`, `production`, `uat`, `release/*`).
- Never commit unrelated changes alongside the tests.
- Stop and report if no gaps meet the severity threshold.

## Output Contract
1. **Coverage gap inventory**: ranked list with regression risk, affected code, and impact for each.
2. **Selected gap**: which one is being addressed and why it ranks highest.
3. **Test plan**: concrete description of tests to write (passed to `$patch`).
4. **Patch result**: output from `$patch` (task summary, changes, verification, PR link).
