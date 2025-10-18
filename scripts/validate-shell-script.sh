#!/usr/bin/env bash
set -eu -o pipefail

script_dir="$( cd "$( dirname "${0}" )" && pwd -P)"

if command -v shellcheck >/dev/null 2>&1; then
    exit_code=0

    while IFS= read -r script_to_check; do
        shellcheck "$script_to_check" &
    done < <(
        find "${script_dir}/.." \
        -type f -name "*.sh" \
        ! -path "*/.git/*" \
        ! -path "*/node_modules/*" \
        ! -path "*/vendor/*" \
        ! -path "*/.pnpm/*" \
        -print0 | xargs -0 shellcheck
    )

    for p in $(jobs -p); do
        wait "$p" || exit_code=$?
    done

    exit "$exit_code"
else
    echo "== Command shellcheck not found in your PATH. No shell script checked."
    exit 1
fi
