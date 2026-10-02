#!/usr/bin/env bash
set -euo pipefail

# Find empty dirs
# find . -mindepth 1 -type d -empty -print
# Delete empty dirs
# find . -mindepth 1 -type d -empty -exec rmdir -- {} +

OLD_IMPORT='"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"'
NEW_IMPORT='"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"'

find ./cmd ./internal -type f -name '*.go' -print0 |
while IFS= read -r -d '' file; do
  # Only change files that import the old basespec package.
  if grep -qF "$OLD_IMPORT" "$file"; then
    perl -0pi -e '
      s|"github\.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"|"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"|g;
      s/\bbasespec\./model./g;
    ' "$file"

    echo "Updated: $file"
  fi
done
