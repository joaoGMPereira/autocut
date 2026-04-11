#!/usr/bin/env bash
# clean-to-shell.sh — clears AutoCut v2 implementations, leaving only the shell
#
# DESTRUCTIVE. Requires --confirm flag. Run from YoutubeProjects/ root.
# Creates a git commit before clearing so the work is recoverable.
#
# Usage: ./AutoCut/scripts/clean-to-shell.sh [--dry-run] [--confirm]

set -e

DRY_RUN=false
CONFIRMED=false

for arg in "$@"; do
  case "$arg" in
    --dry-run)  DRY_RUN=true ;;
    --confirm)  CONFIRMED=true ;;
  esac
done

AUTOCUT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Directories whose .go files will have implementation bodies cleared
IMPL_DIRS=(
  "server/internal/downloader"
  "server/internal/transcript"
  "server/internal/ai"
  "server/internal/processor"
  "server/internal/thumbnail"
  "server/internal/uploader"
  "server/internal/pipeline"
  "server/internal/effects"
  "server/internal/configurator"
)

# Count files
total=0
for dir in "${IMPL_DIRS[@]}"; do
  count=$(find "$AUTOCUT_ROOT/$dir" -name "*.go" ! -name "*_test.go" 2>/dev/null | wc -l | tr -d ' ')
  echo "  $dir: $count .go files"
  total=$((total + count))
done
echo ""
echo "Total files to process: $total"
echo "Test files: preserved"
echo "Database schema: preserved"
echo "Router/config/types: preserved"
echo ""

if $DRY_RUN; then
  echo "DRY RUN — no changes made."
  exit 0
fi

if ! $CONFIRMED; then
  echo "ERROR: This is a destructive operation." >&2
  echo "Review the file list above, then re-run with --confirm to proceed." >&2
  echo "A git commit will be created before clearing so work is recoverable." >&2
  exit 1
fi

# Commit current state before clearing
echo "Creating safety commit before clearing..."
cd "$AUTOCUT_ROOT"
git add -A
git commit -m "chore: snapshot before clean-to-shell (recoverable)" || echo "Nothing to commit, proceeding."

echo "Clearing implementation bodies..."
for dir in "${IMPL_DIRS[@]}"; do
  find "$AUTOCUT_ROOT/$dir" -name "*.go" ! -name "*_test.go" | while read -r file; do
    echo "  Clearing: $file"
    pkg=$(grep -m1 "^package " "$file" | awk '{print $2}')
    printf 'package %s\n\n// TODO: Transcribe from Kotlin source — see AutoCut/.specify/memory/constitution.md\n// This file was cleared by clean-to-shell.sh as part of the constitution bootstrap.\n' "$pkg" > "$file"
  done
done

echo ""
echo "Shell cleanup complete."
echo "Next: run 'go build ./...' to verify the shell compiles."
echo "Then start feature-by-feature transcription from the Kotlin source."
