#!/bin/sh
set -e

# Local validation script that mirrors CI (lint + test + coverage).

git_repo_root_dir=$(git rev-parse --show-toplevel)
cd "$git_repo_root_dir"

echo "=== Lint ==="
./scripts/lint.sh
echo ""

echo "=== Test ==="
./scripts/covgate.sh
