# Surface Linters for gotools

## Goal

Add non-Go surface linters (YAML, shell script, GitHub Actions) to the gotools repository, matching what was added to the core repository.

## Background

The core repository added surface linters in commits `3360e12`–`6ecf0d7`. The implementation consists of:
- `.yamllint.yml` — yamllint configuration
- `.github/actionlint.yml` — actionlint configuration (minimal)
- `scripts/lint-surface.sh` — runs yamllint, shellcheck, actionlint
- `scripts/install-lint-tools.sh` — installs the three linters
- `surface-lint` CI job in `.github/workflows/ci.yml`

## Scope

Repo: `gotools` (`/home/ben/miru/workbench3/gotools`)

## Steps

### 1. Add `.yamllint.yml`

Create `.yamllint.yml` at repo root, adapted from core:
- Same rules (document-start, line-length, truthy, comments)
- Ignore `.agents/` instead of `.venv/` (gotools has `.agents/` subtree, no `.venv/`)
- No `tests/testdata/` ignore needed (gotools has no such dir)

### 2. Add `.github/actionlint.yml`

Create `.github/actionlint.yml` with minimal comment (no custom config needed).

### 3. Add `scripts/lint-surface.sh`

Copy from core verbatim — checks for yamllint, shellcheck, actionlint, then runs each.
Make executable.

### 4. Add `scripts/install-lint-tools.sh`

Copy from core verbatim — installs yamllint (via pip3/.venv), shellcheck (apt/brew), actionlint (go install).
Make executable.

### 5. Update `.github/workflows/ci.yml`

Add `surface-lint` job matching core:
- Checkout
- yamllint via `ibiqlik/action-yamllint` (SHA-pinned, same as core)
- shellcheck via `ludeeus/action-shellcheck` (SHA-pinned, same as core)
- actionlint via `rhysd/actionlint` (SHA-pinned, same as core)

## Test Steps

- `yamllint -c .yamllint.yml .` passes (run locally with yamllint installed)
- `shellcheck scripts/*.sh` passes
- `actionlint` passes on the workflows
- `scripts/lint-surface.sh` runs end-to-end without error (with tools installed)
- CI `surface-lint` job is syntactically valid (actionlint validates it)

## Validation

Preflight must report `clean` before a PR is opened. Run `./scripts/preflight.sh` and confirm no failures.
