#!/bin/sh
set -e

usage() {
	cat <<EOF
Usage: $(basename "$0") [options]

Install the surface linters required by scripts/lint-surface.sh:
  - yamllint    (YAML linter)
  - shellcheck  (shell script linter)
  - actionlint  (GitHub Actions workflow linter)

yamllint is installed into .venv if one exists, otherwise via system pip3.
shellcheck is installed via apt-get or brew.
actionlint is installed via go install.

Options:
  -h, --help   Show this help message and exit
EOF
}

while [ $# -gt 0 ]; do
	case "$1" in
		-h|--help)
			usage
			exit 0
			;;
		*)
			echo "error: unknown option: $1" >&2
			usage >&2
			exit 1
			;;
	esac
done

git_repo_root_dir=$(git rev-parse --show-toplevel)
cd "$git_repo_root_dir"

# --- yamllint ---
echo "==> Installing yamllint..."
if [ -d ".venv" ]; then
	.venv/bin/pip install --quiet yamllint
else
	pip3 install --quiet yamllint
fi
echo "    done."

# --- shellcheck ---
echo "==> Installing shellcheck..."
if command -v apt-get >/dev/null 2>&1; then
	sudo apt-get install -y shellcheck
elif command -v brew >/dev/null 2>&1; then
	brew install shellcheck
else
	echo "error: no supported package manager found (apt-get, brew)." >&2
	echo "Install shellcheck manually: https://github.com/koalaman/shellcheck#installing" >&2
	exit 1
fi
echo "    done."

# --- actionlint ---
echo "==> Installing actionlint..."
go install github.com/rhysd/actionlint/cmd/actionlint@latest
echo "    done."

echo ""
echo "All surface linters installed."
