---
name: hunt-docs
description: Proactively scan a codebase for missing or outdated documentation, prioritize by user impact, then hand off the most critical gap to $patch for improvement. Use when asked to find documentation gaps, audit docs freshness, or improve developer experience.
---

# Documentation Hunt Workflow

## Inputs
- `repo` (optional): constrain scope to one repo path (e.g. `backend`, `frontend`, `core`, or `.` for root).
- `scope` (optional): files, package, module, or full repo. Defaults to full repo.
- `severity` (optional): minimum severity to act on — `critical`, `high`, `medium`. Defaults to `high`.
- `base` (optional): target branch for the fix. Defaults to `main`.

## References
Use these skills as authoritative lenses during analysis:
- `$document` for documentation standards, docstring conventions, and comment quality.
- `$review` for identifying behavior that should be documented.
- `$commit` for commit conventions.
- `$patch` for executing the fix.

## Procedure

### 1. Analyze the codebase for documentation gaps
- Read project structure, public APIs, exported types, configuration options, and user-facing interfaces.
- Systematically scan for documentation issues:
  - Public APIs and exported functions/types with no docstrings or usage examples.
  - Stale or misleading comments that contradict the current implementation.
  - Outdated README sections (wrong setup steps, deprecated flags, removed features).
  - Missing usage examples for non-obvious interfaces.
  - Undocumented configuration options, environment variables, or feature flags.
  - Missing or incomplete architecture/design documentation for complex subsystems.
  - Broken or outdated links in documentation files.
- Produce a ranked gap list sorted by user impact.

### 2. Select the most critical gap
- Pick the highest-severity gap from the list.
- Present the full gap list to the user, indicate which will be addressed, and proceed immediately.
- If no gaps are found at or above the minimum severity, report that and stop.

### 3. Hand off to $patch
- Invoke `$patch` with:
  - `task`: what documentation to write or update — target symbol/file, what's missing or wrong, and what the correct content should cover.
  - `repo` and `base`: pass through from inputs.
  - `evidence`: description of the gap — what is missing or misleading, where users would encounter it, and why it matters.

## Severity Definitions
- **critical**: Actively misleading documentation — wrong API signatures, incorrect examples that would cause errors, security-relevant instructions that are outdated.
- **high**: Missing documentation on public APIs or user-facing configuration — developers cannot use the interface without reading source code.
- **medium**: Stale comments, incomplete examples, or missing docs on internal-but-shared interfaces.

## Safety Rules
- Never force push.
- Never push directly to protected branches (`main`, `production`, `uat`, `release/*`).
- Never commit unrelated changes alongside the documentation fix.
- Stop and report if no gaps meet the severity threshold.

## Output Contract
1. **Documentation gap inventory**: ranked list with user impact and location for each.
2. **Selected gap**: which one is being addressed and why it ranks highest.
3. **Evidence**: description of what is missing or misleading.
4. **Patch result**: output from `$patch` (task summary, changes, verification, PR link).
