#!/usr/bin/env bash
# validate-feature.sh — run validation script for a single feature by number
#
# Usage: ./validate-feature.sh <feature-number>
# Example: ./validate-feature.sh 1

set -e

NUM="$1"

if [[ -z "$NUM" || ! "$NUM" =~ ^[0-9]+$ ]]; then
  echo "Usage: validate-feature.sh <feature-number>" >&2
  exit 1
fi

SCRIPT_DIR="$(dirname "$0")"
FEATURE_SCRIPT="$SCRIPT_DIR/features/validate-$(printf '%02d' "$NUM").sh"

if [[ ! -f "$FEATURE_SCRIPT" ]]; then
  echo "No validation script for feature #$NUM (expected: $FEATURE_SCRIPT)"
  echo "Status: MISSING SCRIPT"
  exit 1
fi

echo "Running validation for feature #$NUM..."
bash "$FEATURE_SCRIPT"
echo "PASS: Feature #$NUM validation passed."
