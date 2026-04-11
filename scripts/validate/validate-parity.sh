#!/usr/bin/env bash
# validate-parity.sh — reads constitution Feature Parity Tracker and reports status
# Exit 0 = no violations. Exit 1 = at least one DONE feature has no passing script.
#
# Usage: ./validate-parity.sh

set -e

CONSTITUTION="$(dirname "$0")/../../.specify/memory/constitution.md"
VALIDATE_DIR="$(dirname "$0")"

if [[ ! -f "$CONSTITUTION" ]]; then
  echo "ERROR: constitution not found at $CONSTITUTION" >&2
  exit 1
fi

TODO=0
IN_PROGRESS=0
NEEDS_VALIDATION=0
DONE=0
VIOLATIONS=0

# Parse table rows — format: | N | Feature | Kotlin Ref | Target | STATUS |
while IFS='|' read -r _ num feature kotlin target status _; do
  # Skip header rows and empty lines
  status="$(echo "$status" | tr -d '[:space:]')"
  [[ "$status" == "Status" || -z "$status" || "$status" == "---" ]] && continue
  num="$(echo "$num" | tr -d '[:space:]')"
  [[ ! "$num" =~ ^[0-9]+$ ]] && continue

  case "$status" in
    TODO)           TODO=$((TODO+1)) ;;
    INPROGRESS)     IN_PROGRESS=$((IN_PROGRESS+1)) ;;
    NEEDSVALIDATION) NEEDS_VALIDATION=$((NEEDS_VALIDATION+1)) ;;
    DONE)
      DONE=$((DONE+1))
      # Check that dedicated script exists and passes
      script="$VALIDATE_DIR/features/validate-$(printf '%02d' "$num").sh"
      if [[ ! -f "$script" ]]; then
        echo "VIOLATION: Feature #$num marked DONE but $script does not exist"
        VIOLATIONS=$((VIOLATIONS+1))
      elif ! bash "$script" > /dev/null 2>&1; then
        echo "VIOLATION: Feature #$num script failed: $script"
        VIOLATIONS=$((VIOLATIONS+1))
      fi
      ;;
  esac
done < "$CONSTITUTION"

echo "Feature Parity Status"
echo "  TODO:             $TODO"
echo "  IN PROGRESS:      $IN_PROGRESS"
echo "  NEEDS VALIDATION: $NEEDS_VALIDATION"
echo "  DONE:             $DONE"
echo "  VIOLATIONS:       $VIOLATIONS"

if [[ $VIOLATIONS -gt 0 ]]; then
  echo "FAIL: $VIOLATIONS violation(s) found."
  exit 1
fi

echo "PASS: No violations."
exit 0
