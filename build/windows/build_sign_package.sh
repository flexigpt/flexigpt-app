#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: build/windows/build_sign_package.sh --version vX.Y.Z [--sign]

Builds the Windows Wails app + NSIS installer (installer bundles build/licenses).
Optionally signs binaries if SIGN_WINDOWS_CERT/SIGN_WINDOWS_CERT_PASSWORD are set.
EOF
}

VERSION_TAG=""
DO_SIGN="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION_TAG="${2:-}"; shift 2 ;;
    --sign) DO_SIGN="true"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown arg: $1"; usage; exit 1 ;;
  esac
done

[[ -n "${VERSION_TAG}" ]] || { echo "ERROR: --version required"; usage; exit 1; }
[[ "${VERSION_TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "ERROR: version must be vX.Y.Z"; exit 1; }

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

REQUESTED_VERSION_TAG="${VERSION_TAG}"
if [[ -f "build/buildvars.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "build/buildvars.env"
  set +a
fi

export VERSION_TAG="${REQUESTED_VERSION_TAG}"

: "${WIN_INSTALLER_PATH:?Must set WIN_INSTALLER_PATH}"
: "${COMMON_BUILD_NAME:?Must set COMMON_BUILD_NAME}"


echo "==> Generating licenses..."
chmod +x build/licenses/gen_licenses.sh
task licenses:gen

echo "==> Verifying generated license files..."
LICENSE_GO="build/licenses/go-dependency-licenses.txt"
LICENSE_JS="build/licenses/js-dependency-licenses.txt"

for f in "${LICENSE_GO}" "${LICENSE_JS}"; do
  if [[ ! -f "$f" ]]; then
    echo "ERROR: Missing generated license file: $f"
    exit 1
  fi
  if [[ ! -s "$f" ]]; then
    echo "ERROR: Generated license file is empty: $f"
    exit 1
  fi
done

echo "==> Building Windows application..."
task build-win-binary

if [[ "${DO_SIGN}" == "true" ]]; then
  echo "==> Signing requested..."
  if [[ -z "${SIGN_WINDOWS_CERT:-}" || -z "${SIGN_WINDOWS_CERT_PASSWORD:-}" ]]; then
    echo "ERROR: --sign requested but SIGN_WINDOWS_CERT or SIGN_WINDOWS_CERT_PASSWORD not set"
    exit 1
  fi

  if command -v powershell.exe >/dev/null 2>&1; then
    powershell.exe -NoProfile -ExecutionPolicy Bypass -File build/windows/sign_windows.ps1 -Target app
  else
    echo "ERROR: powershell.exe not found; cannot sign"
    exit 1
  fi
fi

echo "==> Building Windows installer..."
task package-win-installer

if [[ "${DO_SIGN}" == "true" ]]; then
  powershell.exe -NoProfile -ExecutionPolicy Bypass -File build/windows/sign_windows.ps1 -Target installer
fi

echo "==> Done."
echo "Installer: ${WIN_INSTALLER_PATH}"
