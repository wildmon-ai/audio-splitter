#!/usr/bin/env bash
# Downloads static macOS ffmpeg/ffprobe (ffmpeg.martin-riedl.de) into
# resources/<arch>/. Versions are pinned and checked against the SHA-256
# below: to update, take the new download links and checksums from
# https://ffmpeg.martin-riedl.de and replace them here.
#
# Homebrew's ffmpeg must not be bundled: it needs Homebrew's libraries and
# fails on Macs that do not have them.
#
# Usage: scripts/download-ffmpeg-macos.sh [arm64|x64]   (default arm64)
set -euo pipefail

ARCH="${1:-arm64}"
BASE="https://ffmpeg.martin-riedl.de/download/macos"
case "$ARCH" in
  arm64)
    BUILD="$BASE/arm64/1789931890_9.0.2"
    FFMPEG_SHA256=c8ed4c4e6978a03c485edbfe4e0a5dc2380f8a30bba5150531b31b094492d924
    FFPROBE_SHA256=fcbe839537485eaee7a7a8bc5cbc0f90d53617e80943e8a5b2e31cb851197ea6
    ;;
  x64 | amd64)
    ARCH=x64
    BUILD="$BASE/amd64/1789931006_9.0.2"
    FFMPEG_SHA256=7c6b4125b191cbf773832dc51f424cf2b6bb7da43007d1e066f95909e47cacd4
    FFPROBE_SHA256=2322438ed2f6319a691291b247d09c69dcaa3a982460d1f269a7e1af335cfdfd
    ;;
  *)
    echo "Unknown arch: $ARCH (use arm64 or x64)" >&2
    exit 1
    ;;
esac

cd "$(dirname "$0")/.."
DEST="resources/$ARCH"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$DEST"

for tool in ffmpeg ffprobe; do
  if [ "$tool" = ffmpeg ]; then expected=$FFMPEG_SHA256; else expected=$FFPROBE_SHA256; fi
  curl -fsSL "$BUILD/$tool.zip" -o "$TMP/$tool.zip"
  actual="$(shasum -a 256 "$TMP/$tool.zip" 2>/dev/null || sha256sum "$TMP/$tool.zip")"
  actual="${actual%% *}"
  if [ "$actual" != "$expected" ]; then
    echo "Checksum mismatch for $BUILD/$tool.zip (expected $expected, got $actual)" >&2
    exit 1
  fi
  unzip -oq "$TMP/$tool.zip" -d "$TMP/$tool"
  install -m 755 "$TMP/$tool/$tool" "$DEST/$tool"
done
echo "Done: ffmpeg and ffprobe in $DEST/"
