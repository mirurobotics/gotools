#!/bin/sh
set -e

exec go run ./cmd/miru coverage \
    --src-prefix=./internal \
    "$@"
