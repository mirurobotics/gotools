---
name: hunt-style
description: Proactively scan a codebase for style inconsistencies and convention violations beyond what linters catch, prioritize by confusion risk, then hand off the most impactful one to $patch for cleanup. Use when asked to find style issues, audit naming conventions, or improve codebase consistency.
---

# Style Hunt Workflow

## Inputs
- `repo` (optional): constrain scope to one repo path (e.g. `backend`, `frontend`, `core`, or `.` for root).
- `scope` (optional): files, package, module, or full repo. Defaults to full repo.
- `severity` (optional): minimum severity to act on — `high`, `medium`. Defaults to `high`.
- `base` (optional): target branch for the fix. Defaults to `main`.

## References
Use these skills as authoritative lenses during analysis:
- `$lint` for understanding what the linter already catches (to avoid duplicating its work).
- `$design` for design-level style and naming conventions.
- `$commit` for commit conventions.
- `$patch` for executing the fix.

## Procedure

### 1. Analyze the codebase for style inconsistencies
- Read project structure, established conventions, and naming patterns across modules.
- Identify the dominant style for each concern (naming, error handling patterns, import organization, etc.) — this is the standard to measure against.
- Systematically scan for inconsistencies that linters do NOT catch:
  - Inconsistent naming conventions across modules (e.g. `camelCase` in one package, `snake_case` in another for the same concept).
  - Inconsistent error handling patterns (e.g. some functions wrap errors, others return raw errors).
  - Inconsistent import organization or module structure.
  - Magic numbers or strings that should be named constants.
  - Copy-pasted logic that should share a common implementation.
  - Dead code (unused exports, unreachable branches, commented-out code).
  - Inconsistent API surface patterns (e.g. some handlers return errors, others panic).
- **Exclude** anything auto-fixable by the project's configured linters — that's `$lint`'s job.
- Produce a ranked issue list sorted by confusion risk.

### 2. Select the most impactful issue
- Pick the highest-severity issue from the list.
- Present the full issue list to the user, indicate which will be fixed, and proceed immediately.
- If no issues are found at or above the minimum severity, report that and stop.

### 3. Hand off to $patch
- Invoke `$patch` with:
  - `task`: what the inconsistency is, which convention is dominant, what needs to change, and the specific files/symbols affected.
  - `repo` and `base`: pass through from inputs.
  - `evidence`: description of the inconsistency — the dominant pattern vs. the deviations, with file/line references.

## Severity Definitions
Style issues are never critical — they don't cause runtime failures.
- **high**: Actively confusing inconsistency that causes misunderstanding or mistakes — e.g. same concept named differently in related modules, error handling pattern that looks like a bug but is intentional.
- **medium**: Inconsistency or convention violation that degrades maintainability but is unlikely to cause active confusion — e.g. magic numbers, minor naming deviations, dead code.

## Safety Rules
- Never force push.
- Never push directly to protected branches (`main`, `production`, `uat`, `release/*`).
- Never commit unrelated changes alongside the style fix.
- Never duplicate what the project's configured linters already handle.
- Stop and report if no issues meet the severity threshold.

## Output Contract
1. **Style issue inventory**: ranked list with confusion risk and affected locations for each.
2. **Selected issue**: which one is being fixed and why it ranks highest.
3. **Evidence**: dominant convention vs. deviations, with file/line references.
4. **Patch result**: output from `$patch` (task summary, changes, verification, PR link).
