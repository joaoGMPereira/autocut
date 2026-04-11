#!/usr/bin/env bash
# validate-01.sh — Feature #1: Download YouTube
#
# Validates that the YouTube downloader is fully implemented (not a stub) and
# that all downloader package tests pass.
#
# Checks:
#   1. server/internal/downloader/youtube.go contains func NewYouTubeDownloader
#      (proof it is not the TODO stub left by clean-to-shell.sh)
#   2. go test ./internal/downloader/... passes in the server directory
#
# Exit 0 = feature properly implemented.
# Exit 1 = stub not replaced or tests failing.
#
# Usage: ./validate-01.sh
#   (safe to call from repo root or any path — resolves paths via SCRIPT_DIR)

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
AUTOCUT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
SERVER_DIR="$AUTOCUT_DIR/server"
YOUTUBE_GO="$SERVER_DIR/internal/downloader/youtube.go"

# ── Check 1: youtube.go must not be a stub ──────────────────────────────────
if [[ ! -f "$YOUTUBE_GO" ]]; then
  echo "ERROR: $YOUTUBE_GO does not exist" >&2
  exit 1
fi

if ! grep -q "func NewYouTubeDownloader" "$YOUTUBE_GO"; then
  echo "FAIL: Feature #1 — youtube.go is still a stub (func NewYouTubeDownloader not found)" >&2
  echo "      File: $YOUTUBE_GO" >&2
  exit 1
fi

echo "CHECK 1 PASS: youtube.go contains func NewYouTubeDownloader"

# ── Check 2: downloader package tests must pass ─────────────────────────────
if [[ ! -d "$SERVER_DIR" ]]; then
  echo "ERROR: server directory not found at $SERVER_DIR" >&2
  exit 1
fi

echo "CHECK 2: running go test ./internal/downloader/... in $SERVER_DIR"
if ! (cd "$SERVER_DIR" && go test ./internal/downloader/...); then
  echo "FAIL: Feature #1 — downloader tests failed" >&2
  exit 1
fi

echo "CHECK 2 PASS: all downloader tests passed"
echo "PASS: Feature #1 (Download YouTube) is fully implemented."
exit 0
