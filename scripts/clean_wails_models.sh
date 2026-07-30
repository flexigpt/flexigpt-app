#!/usr/bin/env bash
set -euo pipefail

#./scripts/clean_wails_models.sh ./frontend/app/apis/wailsjs/go/models.ts

file="${1:?Usage: ./clean_wails_models.sh models.ts}"

perl -0777 -i.bak -pe '
  # Remove static createFrom(...)
  s/^([ \t]*)static\s+createFrom\(source:\s*any\s*=\s*\{\}\)\s*(?::[^{\r\n]+)?\s*\{.*?^\1\}[ \t]*\r?\n?//gms;

  # Remove constructor(source: any = {})
  s/^([ \t]*)constructor\(source:\s*any\s*=\s*\{\}\)\s*\{.*?^\1\}[ \t]*\r?\n?//gms;

  # Remove convertValues(...)
  s/^([ \t]*)convertValues\(a:\s*any\s*,\s*classs:\s*any\s*,\s*asMap:\s*boolean\s*=\s*false\s*\)\s*:\s*any\s*\{.*?^\1\}[ \t]*\r?\n?//gms;
' "$file"

echo "Cleaned: $file"
echo "Backup:  $file.bak"
