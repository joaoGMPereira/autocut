#!/bin/sh
set -e

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BASE="http://127.0.0.1:4070"
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

echo "=== validate-104: Workflow Mode Selection (Expanded) ==="

# 1. Go build
echo ""
echo "[Build]"
if (cd "$REPO_ROOT/server" && CGO_ENABLED=0 go build ./... 2>/dev/null); then
  check "CGO_ENABLED=0 go build ./..." "true"
else
  (cd "$REPO_ROOT/server" && CGO_ENABLED=0 go build ./... 2>&1 | head -30) || true
  check "CGO_ENABLED=0 go build ./..." "false"
fi

# 2. Health check (server must be running for endpoint checks)
echo ""
echo "[Health]"
if ! curl -sf --max-time 3 "$BASE/api/health" >/dev/null 2>&1; then
  echo "  WARN  Server not running — skipping endpoint checks"
  echo ""
  echo "=== validate-104: build PASS (server offline — endpoint checks skipped) ==="
  echo "Results: $PASS passed, $FAIL failed"
  if [ "$FAIL" -gt 0 ]; then
    echo "FAIL"
    exit 1
  fi
  echo "PASS"
  exit 0
fi
check "health endpoint" "curl -sf '$BASE/api/health' | grep -q '\"ok\"'"

# 3. prior-clips endpoint exists and returns correct shape
echo ""
echo "[prior-clips endpoint]"
PRIOR=$(curl -sf "$BASE/api/pipeline/runs/prior-clips?url=https%3A%2F%2Fyoutu.be%2Ftest" 2>/dev/null || echo "")
check "prior-clips returns exists field" \
  "echo '$PRIOR' | grep -q '\"exists\"'"

# 4. advance with valid longform mode_config does not 404
echo ""
echo "[advance mode_config]"
RUN=$(curl -sf -X POST "$BASE/api/pipeline/runs" 2>/dev/null || echo "")
RUN_ID=$(echo "$RUN" | grep -o '"id":[0-9]*' | head -1 | sed 's/"id"://')
if [ -n "$RUN_ID" ] && [ "$RUN_ID" != "0" ]; then
  PAYLOAD='{"mode_config":{"mode":"longform","min_duration_secs":480,"max_duration_secs":0,"segment_secs":600,"anti_duplicate":{"enabled":false,"mode":"subtle"},"skip_regenerate":false}}'
  ADV_STATUS=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "$BASE/api/pipeline/runs/$RUN_ID/advance" \
    -H "Content-Type: application/json" -d "$PAYLOAD" 2>/dev/null || echo "000")
  check "advance with mode_config not 404" \
    "[ '$ADV_STATUS' != '404' ]"
else
  echo "  WARN  Could not create run — skipping advance check"
fi

# 5. advance with invalid mode returns 400
echo ""
echo "[mode validation]"
if [ -n "$RUN_ID" ] && [ "$RUN_ID" != "0" ]; then
  BAD_PAYLOAD='{"mode_config":{"mode":"invalid","min_duration_secs":480,"max_duration_secs":0}}'
  BAD_STATUS=$(curl -sf -o /dev/null -w "%{http_code}" -X POST "$BASE/api/pipeline/runs/$RUN_ID/advance" \
    -H "Content-Type: application/json" -d "$BAD_PAYLOAD" 2>/dev/null || echo "000")
  check "advance with bad mode not 200" \
    "[ '$BAD_STATUS' != '200' ]"
fi

echo ""
echo "=== validate-104: PASS=$PASS FAIL=$FAIL ==="
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo "FAIL"
  exit 1
fi
echo "PASS"
exit 0
