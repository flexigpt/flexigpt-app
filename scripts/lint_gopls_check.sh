#!/usr/bin/env bash
#
# lint_gopls_check.sh  [PATH] [JOBS]
#
# Runs "gopls check" on every *.go file under PATH (default ".")
# using JOBS parallel workers (default 8).  Aggregates the results
# and fails the build if any file reports diagnostics.

set -euo pipefail

TARGET=${1:-'.'}
JOBS=${2:-8}

echo "==> gopls check (up to $JOBS parallel jobs)"

# Collect Go files
mapfile -d '' FILES < <(find "$TARGET" -type f -name '*.go' -print0)

if ((${#FILES[@]} == 0)); then
	echo "No Go files found under $TARGET"
	exit 0
fi

# Temp file to record failures
FAIL_FILE=$(mktemp)
trap 'rm -f "$FAIL_FILE"' EXIT
export FAIL_FILE # so that the subshells can use it

# One gopls invocation per file
printf '%s\0' "${FILES[@]}" |
	xargs -0 -n1 -P"$JOBS" bash -c '
    file="$0"                    # first arg after -c is $0
    if ! out=$(gopls check -- "$file" 2>&1); then
        printf "✗ %s\n%s\n\n" "$file" "$out"
        echo "$file" >> "$FAIL_FILE"
    else
				cat
        # printf "✓ %s\n" "$file"
    fi
'

# Summary
if [[ -s $FAIL_FILE ]]; then
	echo
	echo "✗ gopls check failed on $(wc -l <"$FAIL_FILE") file(s):"
	cat "$FAIL_FILE"
	exit 1
fi

echo "✓ gopls check passed"
