#!/bin/sh
set -e

command -v yamllint >/dev/null 2>&1 || { echo "error: yamllint not installed" >&2; exit 1; }
command -v shellcheck >/dev/null 2>&1 || { echo "error: shellcheck not installed" >&2; exit 1; }
command -v actionlint >/dev/null 2>&1 || { echo "error: actionlint not installed" >&2; exit 1; }

yamllint -c .yamllint.yml .

find . -name '*.sh' ! -path './.agents/*' -exec shellcheck {} +

actionlint
