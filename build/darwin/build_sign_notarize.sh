#!/bin/bash
set -euo pipefail

################################################################################
# Usage: build_sign_notarize.sh [--version VERSION] [--sign] [--notarize]
################################################################################

function usage() {
  cat <<EOF
Usage: $(basename "$0") [--version <version>] [--sign] [--notarize]

  --version <version>   Specify the app version (required, must be vX.Y.Z).
  --sign                Sign the .app (and sign the .pkg if MACOS_SIGN_INSTALLER_ID is set).
  --notarize            Notarize the resulting .pkg (requires --sign).
  -h, --help            Show this help.
EOF
}

SIGN_APP=false
NOTARIZE_APP=false
VERSION_TAG=""

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

# Load buildvars for local runs (CI may have already exported env vars).
if [[ -f "build/buildvars.env" ]]; then set -a; source build/buildvars.env; set +a; fi

while [[ $# -gt 0 ]]; do
  case "$1" in
  --version)
    VERSION_TAG="$2"
    shift 2
    ;;
  --sign)
    SIGN_APP=true
    shift
    ;;
  --notarize)
    NOTARIZE_APP=true
    shift
    ;;
  -h | --help)
    usage
    exit 0
    ;;
  *)
    echo "Unknown argument: $1"
    usage
    exit 1
    ;;
  esac
done

if [[ -z "$VERSION_TAG" ]]; then
  echo "Error: --version is required."
  usage
  exit 1
fi

if ! [[ "$VERSION_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Error: --version must be in format vX.Y.Z (e.g., v1.2.3)"
  exit 1
fi

if [[ "$NOTARIZE_APP" == true && "$SIGN_APP" != true ]]; then
  echo "Error: --notarize requires --sign. Please specify both."
  usage
  exit 1
fi

export VERSION_TAG

################################################################################
# Step 1: Render all needed templates at the start
################################################################################

RENDER_FLAGS="--version $VERSION_TAG --info-plist"
if [[ "$SIGN_APP" == true ]]; then
  RENDER_FLAGS="$RENDER_FLAGS --entitlements --gon-sign"
fi
if [[ "$NOTARIZE_APP" == true ]]; then
  RENDER_FLAGS="$RENDER_FLAGS --gon-notarize"
fi

echo "Rendering templates..."
chmod +x build/darwin/render_templates.sh
build/darwin/render_templates.sh $RENDER_FLAGS

################################################################################
# Step 1.5: Generate licenses (Go + JS) into build/licenses
################################################################################
echo "Generating licenses..."
chmod +x build/licenses/gen_licenses.sh
task licenses:gen
LIC_GO="./build/licenses/go-dependency-licenses.txt"
LIC_JS="./build/licenses/js-dependency-licenses.txt"
for f in "${LIC_GO}" "${LIC_JS}"; do
  [[ -f "$f" ]] || { echo "ERROR: Missing generated license file: $f"; exit 1; }
  [[ -s "$f" ]] || { echo "ERROR: Empty generated license file: $f"; exit 1; }
done

################################################################################
# Step 2: Build .app
################################################################################

: "${MACOS_APP_BUNDLE_PATH:?Must set MACOS_APP_BUNDLE_PATH}"

echo "Building .app through the root Taskfile..."
task build-mac

################################################################################
# Step 2.5: Include licenses into .app bundle (MUST happen before signing)
################################################################################
: "${LICENSE_OUTPUT_DIR:=./build/licenses}"
LICENSES_DST_DIR="${MACOS_APP_BUNDLE_PATH}/Contents/Resources/licenses"

echo "Including licenses into app bundle..."
rm -rf "${LICENSES_DST_DIR}"
mkdir -p "${LICENSES_DST_DIR}"
cp -f "${LIC_GO}" "${LICENSES_DST_DIR}/$(basename "${LIC_GO}")"
cp -f "${LIC_JS}" "${LICENSES_DST_DIR}/$(basename "${LIC_JS}")"
echo "Licenses copied to: ${LICENSES_DST_DIR}"

################################################################################
# Step 3: Sign .app (if requested)
################################################################################

if [[ "$SIGN_APP" == true ]]; then
  : "${MACOS_GON_SIGN_JSON_PATH:?Must set MACOS_GON_SIGN_JSON_PATH for signing}"
  echo "Signing the app bundle with gon..."
  gon -log-level=info "$MACOS_GON_SIGN_JSON_PATH"
else
  echo "No signing requested. Skipping signing steps."
  codesign --force --deep --sign - "${MACOS_APP_BUNDLE_PATH}"
fi

################################################################################
# Step 4: Build .pkg (signed if requested)
################################################################################

: "${MACOS_PKG_BUNDLE_PATH:?Must set MACOS_PKG_BUNDLE_PATH}"
mkdir -p "$(dirname "$MACOS_PKG_BUNDLE_PATH")"

if [[ "$SIGN_APP" == true && -n "${MACOS_SIGN_INSTALLER_ID:-}" ]]; then
  : "${MACOS_SIGN_INSTALLER_ID:?Must set MACOS_SIGN_INSTALLER_ID to sign the installer pkg}"
  echo "Building signed installer pkg at: ${MACOS_PKG_BUNDLE_PATH}"
  productbuild \
    --sign "${MACOS_SIGN_INSTALLER_ID}" \
    --component "${MACOS_APP_BUNDLE_PATH}" /Applications \
    "${MACOS_PKG_BUNDLE_PATH}"
else
  echo "Building unsigned installer pkg at: ${MACOS_PKG_BUNDLE_PATH}"
  productbuild \
    --component "${MACOS_APP_BUNDLE_PATH}" /Applications \
    "${MACOS_PKG_BUNDLE_PATH}"
fi

################################################################################
# Step 5: Notarize (if requested)
################################################################################

if [[ "$SIGN_APP" == true && "$NOTARIZE_APP" == true ]]; then
  : "${MACOS_GON_NOTARIZE_JSON_PATH:?Must set MACOS_GON_NOTARIZE_JSON_PATH for notarization}"
  echo "Notarizing .pkg with gon..."
  gon -log-level=info "$MACOS_GON_NOTARIZE_JSON_PATH"
else
  echo "No notarization requested. Skipping notarization steps."
fi

echo "Done! Final pkg is at: ${MACOS_PKG_BUNDLE_PATH}"
if [[ "$SIGN_APP" == true ]]; then
  echo "App was signed. Pkg may also be signed if MACOS_SIGN_INSTALLER_ID was set."
fi
if [[ "$NOTARIZE_APP" == true ]]; then
  echo "Pkg was notarized."
fi
