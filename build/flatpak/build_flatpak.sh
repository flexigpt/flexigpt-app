#!/usr/bin/env bash
set -euo pipefail

# Export all variables from buildvars.env
set -a
. build/buildvars.env
set +a

echo "==== ENVIRONMENT VARIABLES ===="
env | sort
echo "==============================="

chmod +x build/flatpak/render_templates.sh
./build/flatpak/render_templates.sh --version $VERSION_TAG

mkdir -p "$XDG_CONFIG_HOME"
mkdir -p "$XDG_DATA_HOME"

PATH="$PATH:$GOROOT/bin:$GOBIN:$NODEROOT/bin" pnpm install --frozen-lockfile

PATH="$PATH:$GOROOT/bin:$GOBIN:$NODEROOT/bin" "$GOROOT/bin/go" mod download

###############################################################################
# Licenses (Go + JS) - generated at build time, packaged into Flatpak
###############################################################################
chmod +x build/licenses/gen_licenses.sh
PATH="$PATH:$GOROOT/bin:$GOBIN:$NODEROOT/bin" build/licenses/gen_licenses.sh --version "$VERSION_TAG"
# Verify expected outputs exist (canonical names)
LIC_GO="build/licenses/go-dependency-licenses.txt"
LIC_JS="build/licenses/js-dependency-licenses.txt"
for f in "${LIC_GO}" "${LIC_JS}"; do
  if [[ ! -f "$f" ]]; then
    echo "ERROR: Missing generated license file: $f"
    exit 1
  fi
  if [[ ! -s "$f" ]]; then
    echo "ERROR: Generated license file is empty: $f"
    exit 1
  fi
done

PATH="$PATH:$GOROOT/bin:$GOBIN:$NODEROOT/bin" pnpm run build:linux

# Cleanup unneeded files
rm -rf node_modules
rm -rf frontend/node_modules
rm -rf ~/.cache/go-build
rm -rf "$GOROOT/pkg"

# Continue with installation
install -Dm00755 build/bin/FlexiGPT "$FLATPAK_DEST/bin/FlexiGPT"
install -Dm00644 frontend/public/icon_256x256.png "$FLATPAK_DEST/share/icons/hicolor/256x256/apps/$FLATPAK_ID.png"
install -Dm00644 build/flatpak/"$FLATPAK_ID".desktop "$FLATPAK_DEST/share/applications/$FLATPAK_ID.desktop"
install -Dm00644 build/flatpak/"$FLATPAK_ID".appdata.xml "$FLATPAK_DEST/share/appdata/$FLATPAK_ID.appdata.xml"

# Standard Flatpak license location
LICENSES_DEST_DIR="$FLATPAK_DEST/share/licenses/$FLATPAK_ID"
mkdir -p "$LICENSES_DEST_DIR"
install -Dm00644 build/licenses/PROJECT_LICENSE.txt "$LICENSES_DEST_DIR/PROJECT_LICENSE.txt"
install -Dm00644 build/licenses/THIRD_PARTY_NOTICES.txt "$LICENSES_DEST_DIR/THIRD_PARTY_NOTICES.txt"
install -Dm00644 build/licenses/go-dependency-licenses.txt "$LICENSES_DEST_DIR/go-dependency-licenses.txt"
install -Dm00644 build/licenses/js-dependency-licenses.txt "$LICENSES_DEST_DIR/js-dependency-licenses.txt"
