#!/bin/sh
set -e

exec go run ./cmd/miru covratchet \
    --packages="./internal/..." \
    --src-prefix=internal
