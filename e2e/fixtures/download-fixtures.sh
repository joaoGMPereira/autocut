#!/usr/bin/env sh
# Download the 3 fixture videos used by the E2E suite.
# Requires yt-dlp on PATH.
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUT_DIR="${SCRIPT_DIR}/videos"
mkdir -p "${OUT_DIR}"

VIDEOS="bfJy1-IRa_k HsNMliaypC0 7hyc3z2WSkQ"

if ! command -v yt-dlp >/dev/null 2>&1; then
  echo "yt-dlp not found on PATH. Install with: brew install yt-dlp" >&2
  exit 1
fi

for VID in $VIDEOS; do
  OUT="${OUT_DIR}/${VID}.mp4"
  if [ -f "$OUT" ]; then
    echo "✔ ${VID}.mp4 already present — skipping"
    continue
  fi
  echo "⬇ downloading ${VID} → ${OUT}"
  yt-dlp \
    --format 'bv*[height<=720][ext=mp4]+ba[ext=m4a]/b[height<=720][ext=mp4]/b[height<=720]' \
    --merge-output-format mp4 \
    -o "${OUT}" \
    "https://www.youtube.com/watch?v=${VID}"
done

echo "done. fixtures in ${OUT_DIR}"
