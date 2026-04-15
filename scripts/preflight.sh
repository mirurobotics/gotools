#!/usr/bin/env bash
# shellcheck shell=bash
set -euo pipefail

# Local validation script that mirrors CI (lint + covgate + surface lint).
# All three run in parallel to minimize wall-clock time.

git_repo_root_dir=$(git rev-parse --show-toplevel)
cd "$git_repo_root_dir"

echo "=== Running lint and covgate in parallel... ==="

# LINT_FIX=0: check-only mode. Auto-fix is disabled to prevent source
# file writes racing with covgate's concurrent compilation of the same module.
export LINT_FIX=0

./scripts/lint.sh &
lint_pid=$!

./scripts/covgate.sh &
covgate_pid=$!

./scripts/lint-surface.sh &
surface_pid=$!

# Wait for all processes regardless of individual exit codes.
# set -e must not short-circuit here — all must run to completion.
lint_rc=0
covgate_rc=0
surface_rc=0
wait "$lint_pid"    || lint_rc=$?
wait "$covgate_pid" || covgate_rc=$?
wait "$surface_pid" || surface_rc=$?

echo ""

if [[ $lint_rc -ne 0 ]]; then
	echo "=== Lint FAILED ==="
fi
if [[ $covgate_rc -ne 0 ]]; then
	echo "=== Covgate FAILED ==="
fi
if [[ $surface_rc -ne 0 ]]; then
	echo "=== Surface lint FAILED ==="
fi
if [[ $lint_rc -eq 0 && $covgate_rc -eq 0 && $surface_rc -eq 0 ]]; then
	echo "=== All checks passed ==="
fi

exit $(( lint_rc | covgate_rc | surface_rc ))
