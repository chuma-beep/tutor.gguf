#!/usr/bin/env bash
# Build a Linux .AppImage for the Wails desktop binary.
# Usage: bash packaging/appimage.sh [output-path]
set -euo pipefail

out="${1:-dist/tutor-desktop-linux-amd64.AppImage}"
bin="build/bin/tutor-desktop"

[ -x "$bin" ] || { echo "missing desktop binary: $bin" >&2; exit 1; }
mkdir -p "$(dirname "$out")"

workdir="$(mktemp -d)"
appdir="$workdir/Tutor.gguf.AppDir"
mkdir -p "$appdir/usr/bin"

cp "$bin" "$appdir/usr/bin/tutor-desktop"
cp build/linux/icon.png "$appdir/tutor-gguf.png"
cp packaging/tutor-gguf.desktop "$appdir/tutor-gguf.desktop"
ln -sf usr/bin/tutor-desktop "$appdir/AppRun"

tool="${APPIMAGETOOL:-/tmp/appimagetool}"
if [ ! -x "$tool" ]; then
  curl -fsSL -o "$tool" \
    "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage"
  chmod +x "$tool"
fi

ARCH=x86_64 "$tool" --appimage-extract-and-run "$appdir" "$out"
echo "wrote $out"
