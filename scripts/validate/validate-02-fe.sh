#!/usr/bin/env bash
# validate-02-fe.sh — Frontend validation for 002-download-pipeline-validation
# Checks: TypeScript compile, no test pages, DownloadInfoCard wired correctly,
#         speed_kbs/eta_sec in types, pipelineStore handles phase_progress.
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

echo "=== validate-02-fe.sh: Download Pipeline — Frontend ==="

cd "$WEB_DIR"

# 1. TypeScript compile
if pnpm tsc --noEmit 2>/dev/null; then
  check "pnpm tsc --noEmit" "ok"
else
  check "pnpm tsc --noEmit" "fail"
fi

# 2. No /download-test page route
if [ ! -f "$WEB_DIR/src/app/download-test/page.tsx" ]; then
  check "No /download-test route (throwaway page removed)" "ok"
else
  check "No /download-test route (throwaway page removed)" "fail"
fi

# 3. No DownloadTestClient component
if [ ! -f "$WEB_DIR/src/components/download-test/DownloadTestClient.tsx" ]; then
  check "No DownloadTestClient component (throwaway removed)" "ok"
else
  check "No DownloadTestClient component (throwaway removed)" "fail"
fi

# 4. DownloadInfoCard exists in pipeline UI (not a test page)
if [ -f "$WEB_DIR/src/components/pipeline/DownloadInfoCard.tsx" ]; then
  check "DownloadInfoCard.tsx exists in pipeline components" "ok"
else
  check "DownloadInfoCard.tsx exists in pipeline components" "fail"
fi

# 5. speed_kbs and eta_sec in SSEPhaseProgressPayload type
TYPES_FILE="$WEB_DIR/src/types/pipeline.ts"
if grep -q "speed_kbs" "$TYPES_FILE" && grep -q "eta_sec" "$TYPES_FILE"; then
  check "speed_kbs and eta_sec in SSEPhaseProgressPayload (pipeline.ts)" "ok"
else
  check "speed_kbs and eta_sec in SSEPhaseProgressPayload (pipeline.ts)" "fail"
fi

# 6. pipelineStore handles speedKbs/etaSec from phase_progress
STORE_FILE="$WEB_DIR/src/store/pipelineStore.ts"
if grep -q "speedKbs" "$STORE_FILE" && grep -q "etaSec" "$STORE_FILE"; then
  check "pipelineStore.ts wires speedKbs and etaSec" "ok"
else
  check "pipelineStore.ts wires speedKbs and etaSec" "fail"
fi

# 7. DownloadInfoCard renders speed and ETA
CARD_FILE="$WEB_DIR/src/components/pipeline/DownloadInfoCard.tsx"
if grep -q "speed" "$CARD_FILE" && grep -q "eta" "$CARD_FILE"; then
  check "DownloadInfoCard renders speed and ETA fields" "ok"
else
  check "DownloadInfoCard renders speed and ETA fields" "fail"
fi

# 8. DownloadInfoCard is rendered from the real pipeline UI (ContextPanel)
CONTEXT_PANEL="$WEB_DIR/src/components/pipeline/ContextPanel.tsx"
if [ -f "$CONTEXT_PANEL" ] && grep -q "DownloadInfoCard" "$CONTEXT_PANEL"; then
  check "DownloadInfoCard referenced in ContextPanel (real pipeline UI)" "ok"
else
  check "DownloadInfoCard referenced in ContextPanel (real pipeline UI)" "fail"
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo "FAIL"
  exit 1
fi
echo "PASS"
exit 0
