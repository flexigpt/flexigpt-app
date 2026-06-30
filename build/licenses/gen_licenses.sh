#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

die() { echo "ERROR: $*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Usage: build/licenses/gen_licenses.sh [--version vX.Y.Z]

Generates:
  - build/licenses/PROJECT_LICENSE.txt          (from repo LICENSE)
  - build/licenses/js-dependency-licenses.txt   (from frontend vite + rollup-plugin-license)
  - build/licenses/go-dependency-licenses.txt   (from go-licenses + template)
  - build/licenses/THIRD_PARTY_NOTICES.txt      (combined)

Environment:
  Uses build/buildvars.env for LICENSE_* vars and INSTALL_TOOL_GO_LICENSES etc.
EOF
}

VERSION_TAG=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION_TAG="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) die "Unknown arg: $1" ;;
  esac
done

# Load buildvars.env (for local runs; CI may already export these).
if [[ -f "build/buildvars.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "build/buildvars.env"
  set +a
fi

: "${LICENSE_TEMPLATE_PATH:?Must set LICENSE_TEMPLATE_PATH}"
: "${LICENSE_OUTPUT_DIR:?Must set LICENSE_OUTPUT_DIR}"
: "${LICENSE_JS_FILENAME:?Must set LICENSE_JS_FILENAME}"
: "${LICENSE_GO_FILENAME:?Must set LICENSE_GO_FILENAME}"
: "${COMMON_PRODUCT_GITPATH:?Must set COMMON_PRODUCT_GITPATH}"
: "${INSTALL_TOOL_GO_LICENSES:?Must set INSTALL_TOOL_GO_LICENSES}"

# Canonical extra outputs (kept here so scripts don't need extra env churn)
LICENSE_PROJECT_FILENAME="${LICENSE_PROJECT_FILENAME:-./build/licenses/PROJECT_LICENSE.txt}"
LICENSE_THIRD_PARTY_FILENAME="${LICENSE_THIRD_PARTY_FILENAME:-./build/licenses/THIRD_PARTY_NOTICES.txt}"

OUT_DIR="${ROOT_DIR}/${LICENSE_OUTPUT_DIR#./}"
JS_OUT="${ROOT_DIR}/${LICENSE_JS_FILENAME#./}"
GO_OUT="${ROOT_DIR}/${LICENSE_GO_FILENAME#./}"
PROJECT_OUT="${ROOT_DIR}/${LICENSE_PROJECT_FILENAME#./}"
THIRD_OUT="${ROOT_DIR}/${LICENSE_THIRD_PARTY_FILENAME#./}"

mkdir -p "${OUT_DIR}"

echo "==> Generating Project license..."
[[ -f "LICENSE" ]] || die "Repo root LICENSE file not found"
cp -f "LICENSE" "${PROJECT_OUT}"

echo "==> Generating JS dependency licenses..."
export GEN_LICENSES="true"
export LICENSE_JS_OUT="${JS_OUT}"

if ! command -v pnpm >/dev/null 2>&1; then
  die "pnpm not found on PATH"
fi

# Run frontend generator
pnpm -C "${ROOT_DIR}/frontend" run licenses:gen
[[ -s "${JS_OUT}" ]] || die "JS licenses output missing/empty at: ${JS_OUT}"

echo "==> Generating Go dependency licenses..."
export GOFLAGS="${GOFLAGS:-"-buildvcs=false"}"

if ! command -v go >/dev/null 2>&1; then
  die "go not found on PATH"
fi

# Ensure module cache exists (go-licenses uses go list/module resolution)
go mod download

if ! command -v go-licenses >/dev/null 2>&1; then
  echo "go-licenses not found; installing..."
  eval "${INSTALL_TOOL_GO_LICENSES}"
fi

[[ -f "${LICENSE_TEMPLATE_PATH}" ]] || die "Template not found: ${LICENSE_TEMPLATE_PATH}"

go-licenses report ./... \
  --template "${LICENSE_TEMPLATE_PATH}" \
  --ignore "${COMMON_PRODUCT_GITPATH}" \
  > "${GO_OUT}"

[[ -s "${GO_OUT}" ]] || die "Go licenses output missing/empty at: ${GO_OUT}"

echo "==> Writing combined THIRD_PARTY_NOTICES..."
{
  echo "FlexiGPT - Third Party Notices"
  echo "Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  if [[ -n "${VERSION_TAG}" ]]; then
    echo "Version: ${VERSION_TAG}"
  fi
  echo
  echo "======================"
  echo "Go dependencies"
  echo "======================"
  cat "${GO_OUT}"
  echo
  echo "======================"
  echo "JavaScript dependencies"
  echo "======================"
  cat "${JS_OUT}"
  echo
} > "${THIRD_OUT}"

echo "==> Done. Outputs:"
ls -la "${OUT_DIR}"
echo "==> Cleaning Licenses Output Dist"
pnpm -C "${ROOT_DIR}/frontend" run clean
