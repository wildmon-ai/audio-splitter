#!/usr/bin/env bash
# Downloads static Linux ffmpeg/ffprobe into resources/<arch>/, where
# electron-builder (extraResources) and the backend in development look.
#
# Usage: scripts/download-ffmpeg.sh [x64|arm64|armv7l]   (default x64; "amd64" means x64)
set -euo pipefail

ARCH="${1:-x64}"
case "$ARCH" in
  x64 | amd64) ARCH=x64 JV=amd64 ;;
  arm64) JV=arm64 ;;
  armv7l | armhf) ARCH=armv7l JV=armhf ;;
  *)
    echo "Unknown arch: $ARCH (use x64, arm64 or armv7l)" >&2
    exit 1
    ;;
esac

cd "$(dirname "$0")/.."
DEST="resources/$ARCH"
URL="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-${JV}-static.tar.xz"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -A "Mozilla/5.0" "$1" -o "$2"
  else
    wget -q "$1" -O "$2"
  fi
}

fetch "$URL" "$TMP/ffmpeg.tar.xz"
fetch "$URL.md5" "$TMP/ffmpeg.tar.xz.md5"
expected="$(cut -d' ' -f1 <"$TMP/ffmpeg.tar.xz.md5")"
actual="$(md5sum "$TMP/ffmpeg.tar.xz" | cut -d' ' -f1)"
if [ "$expected" != "$actual" ]; then
  echo "Checksum mismatch for $URL (expected $expected, got $actual)" >&2
  exit 1
fi

tar -xJf "$TMP/ffmpeg.tar.xz" -C "$TMP"
mkdir -p "$DEST"
install -m 755 "$TMP"/ffmpeg-*-static/ffmpeg "$TMP"/ffmpeg-*-static/ffprobe "$DEST/"
echo "Done: ffmpeg and ffprobe in $DEST/"
