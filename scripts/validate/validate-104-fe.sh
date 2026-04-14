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

STEPMODE="$REPO_ROOT/apps/web/src/components/pipeline/steps/StepMode.tsx"
TYPES="$REPO_ROOT/apps/web/src/types/pipeline.ts"

echo "=== validate-104-fe: Workflow Mode Selection Frontend ==="

# File existence
echo ""
echo "[Files]"
check "StepMode.tsx exists" "[ -f '$STEPMODE' ]"
check "types/pipeline.ts exists" "[ -f '$TYPES' ]"

# Types
echo ""
echo "[Types]"
check "AntiDuplicateConfig defined" "grep -q 'AntiDuplicateConfig' '$TYPES'"
check "PriorClipsResponse defined" "grep -q 'PriorClipsResponse' '$TYPES'"
check "ModeConfig has anti_duplicate" "grep -q 'anti_duplicate' '$TYPES'"
check "ModeConfig has skip_regenerate" "grep -q 'skip_regenerate' '$TYPES'"
check "AI_DEFAULTS has anti_duplicate" "grep -q 'anti_duplicate' '$TYPES'"

# StepMode feature markers
echo ""
echo "[StepMode features]"
check "AI duration: clipDurationSecs state" "grep -q 'clipDurationSecs' '$STEPMODE'"
check "AI duration: minDurationSecs state" "grep -q 'minDurationSecs' '$STEPMODE'"
check "AI duration: isMinDurationInvalid" "grep -q 'isMinDurationInvalid' '$STEPMODE'"
check "Anti-dup: antiDupEnabled state" "grep -q 'antiDupEnabled' '$STEPMODE'"
check "Anti-dup: antiDupMode state" "grep -q 'antiDupMode' '$STEPMODE'"
check "Anti-dup: panel markup" "grep -q 'Anti-Duplicate' '$STEPMODE'"
check "Reuse: priorRunId state" "grep -q 'priorRunId' '$STEPMODE'"
check "Reuse: prior-clips endpoint" "grep -q 'prior-clips' '$STEPMODE'"
check "Reuse: skip_regenerate" "grep -q 'skipRegenerate' '$STEPMODE'"
check "Persistence: ai_clip_duration_secs key" "grep -q 'ai_clip_duration_secs' '$STEPMODE'"
check "Persistence: anti_dup_enabled key" "grep -q 'anti_dup_enabled' '$STEPMODE'"
check "Persistence: Promise.allSettled" "grep -q 'Promise.allSettled' '$STEPMODE'"
check "Logging: [StepMode] prefix" "grep -q '\[StepMode\]' '$STEPMODE'"

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
echo "=== validate-104-fe: PASS=$PASS FAIL=$FAIL ==="
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo "FAIL"
  exit 1
fi
echo "PASS"
exit 0
