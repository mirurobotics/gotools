#!/bin/sh
set -e

FIX="--fix"
if [ "${LINT_FIX:-1}" = "0" ]; then
    FIX="--fix=false"
fi

exec go run ./cmd/miru lint \
    --paths=internal \
    --exclude=nofmt,bgctx \
    $FIX
