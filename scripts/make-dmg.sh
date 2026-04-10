#!/usr/bin/env bash
# Create a drag-to-Applications DMG for macOS.
# Uses create-dmg (brew) if available; falls back to built-in hdiutil.
#
# Usage: make-dmg.sh <Source.app> <output.dmg> <VolumeName>

set -euo pipefail

APP=${1:?"Usage: $0 <Source.app> <output.dmg> <VolumeName>"}
DMG=${2:?}
VOLNAME=${3:?}
APPNAME="WGTray.app"

rm -f "$DMG"

# Always rename to WGTray.app inside the DMG, regardless of source bundle name.
STAGING=$(mktemp -d)
trap 'rm -rf "$STAGING"' EXIT
cp -r "$APP" "$STAGING/$APPNAME"

if command -v create-dmg &>/dev/null; then
    # Produces a polished DMG with background arrow and window layout.
    create-dmg \
        --volname   "$VOLNAME" \
        --window-size 540 380 \
        --icon-size 128 \
        --icon      "$APPNAME" 130 170 \
        --app-drop-link 400 170 \
        --hide-extension "$APPNAME" \
        "$DMG" \
        "$STAGING/$APPNAME"
else
    # Built-in hdiutil fallback — no extra dependencies, works in CI.
    ln -s /Applications "$STAGING/Applications"
    hdiutil create \
        -volname    "$VOLNAME" \
        -srcfolder  "$STAGING" \
        -ov \
        -format     UDBZ \
        "$DMG"
fi

echo "✅  $(basename "$DMG")  ($(du -sh "$DMG" | cut -f1))"
