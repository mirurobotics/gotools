#!/bin/sh
set -e

# Paths below are relative to the Go module root.
cd "$(dirname "$0")/../../go"

FIX="--fix"
if [ "${LINT_FIX:-1}" = "0" ]; then
    FIX="--fix=false"
fi

# bgctx and tempdir require core (mctx, test_dirs), which this module does not
# depend on, so its own context and temp-dir calls are legitimate.
exec go run ./cmd/miru lint \
    --paths=internal \
    --exclude=nofmt,bgctx,tempdir \
    --goos=windows \
    $FIX
