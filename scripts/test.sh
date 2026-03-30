#!/bin/sh
set -e

exec go run ./cmd/miru test -- "$@"
