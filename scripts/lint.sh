#!/bin/sh
set -e

FIX="--fix"
if [ "${LINT_FIX:-1}" = "0" ]; then
    FIX="--fix=false"
fi

# bgctx and tempdir require core (mctx, test_dirs), which gotools does not
# depend on, so its own context and temp-dir calls are legitimate.
exec go run ./cmd/miru lint \
    --paths=internal \
    --exclude=nofmt,bgctx,tempdir \
    $FIX
