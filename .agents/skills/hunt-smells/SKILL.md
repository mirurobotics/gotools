---
name: hunt-smells
description: Proactively scan a codebase for code smells and structural complexity, prioritize by impact on maintainability, then hand off the most critical one to $patch for simplification. Use when asked to find code smells, reduce complexity, or improve code structure.
---

# Code Smell Hunt Workflow

## Inputs
- `repo` (optional): constrain scope to one repo path (e.g. `backend`, `frontend`, `core`, or `.` for root).
- `scope` (optional): files, package, module, or full repo. Defaults to full repo.
- `severity` (optional): minimum severity to act on — `high`, `medium`. Defaults to `high`.
- `base` (optional): target branch for the fix. Defaults to `main`.

## References
Use these skills as authoritative lenses during analysis:
- `$redesign` for complexity reduction, function splitting, and boundary clarification.
- `$design` for structural quality and single-purpose composition.
- `$test` for verifying behavior is preserved after simplification.
- `$commit` for commit conventions.
- `$patch` for executing the fix.

## Procedure

### 1. Analyze the codebase for code smells
- Read project structure, module boundaries, and high-churn or high-complexity areas.
- Systematically scan using the analysis approach from `$redesign`:
  - God functions/modules (functions > ~100 lines or with > 3 responsibilities).
  - Deep nesting (> 3 levels of indentation in control flow).
  - Mixed abstraction levels within a single function or module.
  - Redundant indirection (wrappers that add no value, pass-through functions).
  - Tight coupling between modules that should be independent.
  - Overly complex control flow (long if/else chains, nested switches, convoluted state machines).
  - Dead code paths (unreachable branches, unused internal functions).
  - Premature or unnecessary abstractions (generics/interfaces with a single implementation).
- Produce a ranked smell list sorted by maintainability impact.

### 2. Select the most impactful smell
- Pick the highest-severity smell from the list.
- Present the full smell list to the user, indicate which will be addressed, and proceed immediately.
- If no smells are found at or above the minimum severity, report that and stop.

### 3. Verify baseline test coverage
- Before handing off, check whether the target code has existing test coverage.
- If tests exist: note them in the hand-off so `$patch` can verify behavior is preserved.
- If no tests cover the target code: note that in the hand-off so `$patch` writes tests first before simplifying.

### 4. Hand off to $patch
- Invoke `$patch` with:
  - `task`: what the smell is, where it lives, why it's a problem, and a concrete simplification approach (e.g. "split into 3 functions", "extract interface", "flatten nesting with early returns").
  - `repo` and `base`: pass through from inputs.
  - `evidence`: description of the smell — current complexity metrics (line count, nesting depth, responsibility count), and the target state after simplification.

## Severity Definitions
Code smells are never critical — they don't cause runtime failures.
- **high**: Actively impedes understanding or safe modification — e.g. 200+ line function with 5+ nesting levels, module with circular dependencies, function that mixes I/O with business logic and error handling.
- **medium**: Unnecessary complexity that degrades maintainability — e.g. redundant wrapper, premature abstraction, moderate nesting that could be flattened.

## Safety Rules
- Never force push.
- Never push directly to protected branches (`main`, `production`, `uat`, `release/*`).
- Never commit unrelated changes alongside the simplification.
- Never change external behavior — simplifications must be behavior-preserving.
- Stop and report if no smells meet the severity threshold.

## Output Contract
1. **Code smell inventory**: ranked list with maintainability impact, complexity metrics, and location for each.
2. **Selected smell**: which one is being addressed and why it ranks highest.
3. **Baseline coverage**: whether existing tests cover the target code.
4. **Evidence**: current complexity vs. target state after simplification.
5. **Patch result**: output from `$patch` (task summary, changes, verification, PR link).
