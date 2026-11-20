#!/bin/sh
set -e

# Paths below are relative to the Go module root.
cd "$(dirname "$0")/../../go"

exec go run ./cmd/miru covratchet \
    --packages="./internal/..." \
    --src-prefix=internal
