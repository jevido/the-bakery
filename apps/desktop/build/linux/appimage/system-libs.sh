#!/usr/bin/env bash
# Builds an AppImage that holds only The Bakery's binary and uses the
# system's GTK 4 and WebKitGTK 6.0.
#
# Why not `wails3 generate appimage`: that bundles WebKitGTK, and WebKit
# looks for its helper processes (WebKitNetworkProcess, WebKitWebProcess) at
# the absolute path it was built with. A bundle made on Ubuntu then crashes
# on Arch, Fedora and others, where that path does not exist. The system
# WebKit always finds its own helpers.
#
# Usage: system-libs.sh <binary> <icon.png> <file.desktop> <output.AppImage>
set -euo pipefail

binary=$1 icon=$2 desktop=$3 output=$4
name=the-bakery
tools="${XDG_CACHE_HOME:-$HOME/.cache}/the-bakery-appimage"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
appdir="$work/$name.AppDir"
mkdir -p "$appdir/usr/bin"
cp "$binary" "$appdir/usr/bin/$name"
cp "$icon" "$appdir/$name.png"
cp "$desktop" "$appdir/$name.desktop"
cat > "$appdir/AppRun" <<'RUN'
#!/bin/sh
here="$(dirname "$(readlink -f "$0")")"
exec "$here/usr/bin/the-bakery" "$@"
RUN
chmod +x "$appdir/AppRun"

mkdir -p "$tools"
tool="$tools/appimagetool-x86_64.AppImage"
if [ ! -x "$tool" ]; then
	curl -fsSL -o "$tool" https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
	chmod +x "$tool"
fi

mkdir -p "$(dirname "$output")"
ARCH=x86_64 "$tool" --appimage-extract-and-run --no-appstream "$appdir" "$output"
