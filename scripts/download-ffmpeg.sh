#!/usr/bin/env bash
set -e

# Usage: download-ffmpeg.sh [amd64|arm64|armv7l]
ARCH="${1:-amd64}"

case "$ARCH" in
  amd64)  JVNAME="amd64" ;;
  arm64)  JVNAME="arm64" ;;
  armv7l) JVNAME="armhf" ;;
  *)
    echo "Unknown arch: $ARCH (use amd64, arm64, or armv7l)" >&2
    exit 1
    ;;
esac

mkdir -p "resources/$ARCH"

url="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-${JVNAME}-static.tar.xz"
archive="/tmp/ffmpeg-${ARCH}.tar.xz"

if command -v curl >/dev/null 2>&1; then
  curl -L --fail --silent --show-error -A "Mozilla/5.0" "$url" -o "$archive"
else
  wget -q "$url" -O "$archive"
fi

tar -xf "$archive" -C /tmp/
cp /tmp/ffmpeg-*-${JVNAME}-static/ffmpeg  "resources/$ARCH/ffmpeg"
cp /tmp/ffmpeg-*-${JVNAME}-static/ffprobe "resources/$ARCH/ffprobe"
chmod +x "resources/$ARCH/ffmpeg" "resources/$ARCH/ffprobe"
echo "Done: ffmpeg and ffprobe in resources/$ARCH/"
