#!/bin/bash
set -euo pipefail

# Usage:
#   render_templates.sh --version <version> [--info-plist] [--entitlements] [--gon-sign] [--gon-notarize]
#
# Renders the requested templates using environment variables.

function usage() {
    cat <<EOF
Usage: $(basename "$0") --version <version> [--info-plist] [--entitlements] [--gon-sign] [--gon-notarize]

  --version <version>      Version string (required)
  --info-plist             Render Info.plist
  --entitlements           Render entitlements plist
  --gon-sign               Render gon-sign.json
  --gon-notarize           Render gon-notarize.json
  -h, --help               Show this help
EOF
}

RENDER_INFO_PLIST=false
RENDER_ENTITLEMENTS=false
RENDER_GON_SIGN=false
RENDER_GON_NOTARIZE=false
VERSION_TAG=""

while [[ $# -gt 0 ]]; do
    case "$1" in
    --version)
        VERSION_TAG="$2"
        shift 2
        ;;
    --info-plist)
        RENDER_INFO_PLIST=true
        shift
        ;;
    --entitlements)
        RENDER_ENTITLEMENTS=true
        shift
        ;;
    --gon-sign)
        RENDER_GON_SIGN=true
        shift
        ;;
    --gon-notarize)
        RENDER_GON_NOTARIZE=true
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

render_template() {
    local template_file="$1"
    local dest_file="$2"
    # For plist and appdata we want a plain numeric version without leading "v".
    local VERSION_NO_V="${VERSION_TAG#v}"

    node build/desktop.mjs render "$template_file" "$dest_file" "$VERSION_NO_V"
}

echo "Rendering templates:"
$RENDER_INFO_PLIST && echo "  - Info.plist"
$RENDER_ENTITLEMENTS && echo "  - entitlements.plist"
$RENDER_GON_SIGN && echo "  - gon-sign.json"
$RENDER_GON_NOTARIZE && echo "  - gon-notarize.json"

if [[ "$RENDER_INFO_PLIST" == true ]]; then
    : "${MACOS_INFO_PLIST_TEMPLATE_PATH:?Must set MACOS_INFO_PLIST_TEMPLATE_PATH}"
    : "${MACOS_INFO_PLIST_PATH:?Must set MACOS_INFO_PLIST_PATH}"
    : "${COMMON_PRODUCT_NAME:?Must set COMMON_PRODUCT_NAME for Info.plist}"
    : "${COMMON_BUILD_NAME:?Must set COMMON_BUILD_NAME for Info.plist}"
    : "${COMMON_PRODUCT_DESCRIPTION:?Must set COMMON_PRODUCT_DESCRIPTION for Info.plist}"
    : "${COMMON_PROJECT_LICENSE:?Must set COMMON_PROJECT_LICENSE for Info.plist}"
    : "${MACOS_BUNDLE_ID:?Must set MACOS_BUNDLE_ID for Info.plist}"
    mkdir -p "$(dirname "$MACOS_INFO_PLIST_PATH")"
    render_template "$MACOS_INFO_PLIST_TEMPLATE_PATH" "$MACOS_INFO_PLIST_PATH"
    echo "Rendered Info.plist"
fi

if [[ "$RENDER_ENTITLEMENTS" == true ]]; then
    : "${MACOS_ENTITLEMENTS_TEMPLATE_PATH:?Must set MACOS_ENTITLEMENTS_TEMPLATE_PATH}"
    : "${MACOS_ENTITLEMENTS_PATH:?Must set MACOS_ENTITLEMENTS_PATH}"
    mkdir -p "$(dirname "$MACOS_ENTITLEMENTS_PATH")"
    render_template "$MACOS_ENTITLEMENTS_TEMPLATE_PATH" "$MACOS_ENTITLEMENTS_PATH"
    echo "Rendered entitlements"
fi

if [[ "$RENDER_GON_SIGN" == true ]]; then
    : "${MACOS_GON_SIGN_JSON_TEMPLATE_PATH:?Must set MACOS_GON_SIGN_JSON_TEMPLATE_PATH}"
    : "${MACOS_GON_SIGN_JSON_PATH:?Must set MACOS_GON_SIGN_JSON_PATH}"
    : "${MACOS_APP_BUNDLE_PATH:?Must set MACOS_APP_BUNDLE_PATH for gon-sign.json}"
    : "${MACOS_BUNDLE_ID:?Must set MACOS_BUNDLE_ID for gon-sign.json}"
    : "${MACOS_SIGN_APPLE_USERNAME:?Must set MACOS_SIGN_APPLE_USERNAME for gon-sign.json}"
    : "${MACOS_SIGN_APPLE_APP_PASSWORD:?Must set MACOS_SIGN_APPLE_APP_PASSWORD for gon-sign.json}"
    : "${MACOS_SIGN_APPLE_TEAM_ID:?Must set MACOS_SIGN_APPLE_TEAM_ID for gon-sign.json}"
    : "${MACOS_SIGN_APPLE_DEVELOPER_IDENTITY:?Must set MACOS_SIGN_APPLE_DEVELOPER_IDENTITY for gon-sign.json}"
    : "${MACOS_ENTITLEMENTS_PATH:?Must set MACOS_ENTITLEMENTS_PATH for gon-sign.json}"
    mkdir -p "$(dirname "$MACOS_GON_SIGN_JSON_PATH")"
    render_template "$MACOS_GON_SIGN_JSON_TEMPLATE_PATH" "$MACOS_GON_SIGN_JSON_PATH"
    echo "Rendered gon-sign.json"
fi

if [[ "$RENDER_GON_NOTARIZE" == true ]]; then
    : "${MACOS_GON_NOTARIZE_JSON_TEMPLATE_PATH:?Must set MACOS_GON_NOTARIZE_JSON_TEMPLATE_PATH}"
    : "${MACOS_GON_NOTARIZE_JSON_PATH:?Must set MACOS_GON_NOTARIZE_JSON_PATH}"
    : "${MACOS_PKG_BUNDLE_PATH:?Must set MACOS_PKG_BUNDLE_PATH for gon-notarize.json}"
    : "${MACOS_BUNDLE_ID:?Must set MACOS_BUNDLE_ID for gon-notarize.json}"
    : "${MACOS_SIGN_APPLE_USERNAME:?Must set MACOS_SIGN_APPLE_USERNAME for gon-notarize.json}"
    : "${MACOS_SIGN_APPLE_APP_PASSWORD:?Must set MACOS_SIGN_APPLE_APP_PASSWORD for gon-notarize.json}"
    : "${MACOS_SIGN_APPLE_TEAM_ID:?Must set MACOS_SIGN_APPLE_TEAM_ID for gon-notarize.json}"
    mkdir -p "$(dirname "$MACOS_GON_NOTARIZE_JSON_PATH")"
    render_template "$MACOS_GON_NOTARIZE_JSON_TEMPLATE_PATH" "$MACOS_GON_NOTARIZE_JSON_PATH"
    echo "Rendered gon-notarize.json"
fi
