#!/bin/sh
set -e

exec go run ./cmd/miru covgate \
    --packages="./internal/..." \
    --src-prefix=internal \
    --default-threshold="${1:-80.0}"
