#!/usr/bin/env bash
# validate-102-fe.sh — Frontend validation for 102-pipeline-nav-back
# Checks: TypeScript compile, key source files exist, store/types correct.
#
# Exit 0 = all checks pass. Exit 1 = at least one check failed.

set -e
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
WEB_DIR="$REPO_ROOT/apps/web"
PASS=0
FAIL=0

check() {
  local desc="$1"
  local result="$2"
  if [ "$result" = "ok" ]; then
    echo "  PASS  $desc"
    PASS=$((PASS+1))
  else
    echo "  FAIL  $desc"
    FAIL=$((FAIL+1))
  fi
}

echo "=== validate-102-fe.sh: Pipeline Navigation Back — Frontend ==="

cd "$WEB_DIR"

# 1. TypeScript compile
if pnpm tsc --noEmit 2>/dev/null; then
  check "pnpm tsc --noEmit" "ok"
else
  pnpm tsc --noEmit 2>&1 | head -40 || true
  check "pnpm tsc --noEmit" "fail"
fi

# 2. GatePayload type defined in pipeline.ts
if grep -q "type GatePayload" "$WEB_DIR/src/types/pipeline.ts" 2>/dev/null; then
  check "pipeline.ts: GatePayload type" "ok"
else
  check "pipeline.ts: GatePayload type missing" "fail"
fi

# 3. GATE_STATE_ORDER constant in pipeline.ts
if grep -q "GATE_STATE_ORDER" "$WEB_DIR/src/types/pipeline.ts" 2>/dev/null; then
  check "pipeline.ts: GATE_STATE_ORDER constant" "ok"
else
  check "pipeline.ts: GATE_STATE_ORDER constant missing" "fail"
fi

# 4. pipelineStore has navHistory, navIndex, gateHistory
STORE="$WEB_DIR/src/store/pipelineStore.ts"
for field in navHistory navIndex gateHistory displayedState isNavigatedBack navigateBack navigateForward recordGateSubmission resubmitFromBacked; do
  if grep -q "$field" "$STORE" 2>/dev/null; then
    check "pipelineStore.ts: $field" "ok"
  else
    check "pipelineStore.ts: $field missing" "fail"
  fi
done

# 5. ActiveStepArea uses displayedState
ACTIVE="$WEB_DIR/src/components/pipeline/ActiveStepArea.tsx"
if grep -q "displayedState" "$ACTIVE" 2>/dev/null; then
  check "ActiveStepArea.tsx: uses displayedState" "ok"
else
  check "ActiveStepArea.tsx: displayedState missing" "fail"
fi

# 6. GateActionBar has Back button
GATE_BAR="$WEB_DIR/src/components/pipeline/GateActionBar.tsx"
if grep -q "navigateBack\|Back" "$GATE_BAR" 2>/dev/null; then
  check "GateActionBar.tsx: Back button wired" "ok"
else
  check "GateActionBar.tsx: Back button missing" "fail"
fi

# 7. BackConfirmDialog exists
DIALOG="$WEB_DIR/src/components/pipeline/BackConfirmDialog.tsx"
if [ -f "$DIALOG" ]; then
  check "BackConfirmDialog.tsx exists" "ok"
else
  check "BackConfirmDialog.tsx missing" "fail"
fi

# 8. useBackNavigation hook exists
HOOK="$WEB_DIR/src/hooks/useBackNavigation.ts"
if [ -f "$HOOK" ]; then
  check "useBackNavigation.ts exists" "ok"
else
  check "useBackNavigation.ts missing" "fail"
fi

# 9. PipelineShell uses useBackNavigation and BackConfirmDialog
SHELL="$WEB_DIR/src/components/pipeline/PipelineShell.tsx"
if grep -q "useBackNavigation" "$SHELL" 2>/dev/null; then
  check "PipelineShell.tsx: useBackNavigation wired" "ok"
else
  check "PipelineShell.tsx: useBackNavigation missing" "fail"
fi
if grep -q "BackConfirmDialog" "$SHELL" 2>/dev/null; then
  check "PipelineShell.tsx: BackConfirmDialog used" "ok"
else
  check "PipelineShell.tsx: BackConfirmDialog missing" "fail"
fi

# 10. Step components accept historical prop
for step in StepUrl StepMode StepReviewHighlights StepThumbnailConfig StepReviewMetadata StepReviewClips; do
  FILE="$WEB_DIR/src/components/pipeline/steps/${step}.tsx"
  if grep -q "historical" "$FILE" 2>/dev/null; then
    check "${step}.tsx: historical prop" "ok"
  else
    check "${step}.tsx: historical prop missing" "fail"
  fi
done

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo "FAIL"
  exit 1
fi
echo "PASS"
exit 0
