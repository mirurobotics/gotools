#!/bin/sh
set -e

# Paths below are relative to the Go module root.
cd "$(dirname "$0")/../../go"

exec go run ./cmd/miru covgate \
    --packages="./internal/..." \
    --src-prefix=internal \
    --default-threshold="${1:-80.0}"
