#!/bin/sh
set -e

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PASS=0
FAIL=0

check() {
  label="$1"
  if eval "$2"; then
    echo "  PASS  $label"
    PASS=$((PASS + 1))
  else
    echo "  FAIL  $label"
    FAIL=$((FAIL + 1))
  fi
}

WEB="$REPO_ROOT/apps/web/src"
HARDWARE_CARD="$WEB/components/setup/HardwareCard.tsx"
CUSTOM_DIALOG="$WEB/components/setup/CustomPathDialog.tsx"
SETUP_STORE="$WEB/store/setupStore.ts"
SETUP_GATE="$WEB/components/setup/SetupGate.tsx"
SETUP_TYPES="$WEB/types/setup.ts"

echo "=== validate-105-fe: Tool Setup & Validation (Frontend) ==="

# File existence
echo ""
echo "[Files]"
check "HardwareCard.tsx exists" "[ -f '$HARDWARE_CARD' ]"
check "CustomPathDialog.tsx exists" "[ -f '$CUSTOM_DIALOG' ]"

# Content checks
echo ""
echo "[Content]"
check "setupStore has fetchHardware" "grep -q 'fetchHardware' '$SETUP_STORE'"
check "SetupGate has versionOk check" "grep -q 'versionOk' '$SETUP_GATE'"
check "types/setup.ts has HardwareProfile" "grep -q 'HardwareProfile' '$SETUP_TYPES'"
check "types/setup.ts has ModelRecommendation" "grep -q 'ModelRecommendation' '$SETUP_TYPES'"
check "types/setup.ts ToolStatus has source field" "grep -q 'source' '$SETUP_TYPES'"
check "HardwareCard reads hardware from store" "grep -q 'hardware' '$HARDWARE_CARD'"
check "CustomPathDialog calls POST tools path" "grep -q '/api/setup/tools' '$CUSTOM_DIALOG'"

# TypeScript compilation
echo ""
echo "[TypeScript]"
if (cd "$REPO_ROOT/apps/web" && pnpm tsc --noEmit 2>/dev/null); then
  check "pnpm tsc --noEmit" "true"
else
  (cd "$REPO_ROOT/apps/web" && pnpm tsc --noEmit 2>&1 | head -40) || true
  check "pnpm tsc --noEmit" "false"
fi

echo ""
echo "=== validate-105-fe: PASS=$PASS FAIL=$FAIL ==="
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo "FAIL"
  exit 1
fi
echo "PASS"
exit 0
