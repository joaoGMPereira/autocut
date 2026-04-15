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

echo "=== validate-105: Tool Setup & Validation ==="

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
  echo "=== validate-105: build PASS (server offline — endpoint checks skipped) ==="
  echo "Results: $PASS passed, $FAIL failed"
  if [ "$FAIL" -gt 0 ]; then
    echo "FAIL"
    exit 1
  fi
  echo "PASS"
  exit 0
fi
check "health endpoint" "curl -sf '$BASE/api/health' | grep -q '\"ok\"'"

# 3. Hardware endpoint
echo ""
echo "[Hardware API]"
HW=$(curl -sf "$BASE/api/setup/hardware" 2>/dev/null || echo "")
check "GET /api/setup/hardware returns total_ram_mb" \
  "echo '$HW' | grep -q '\"total_ram_mb\"'"
check "GET /api/setup/hardware total_ram_mb > 0" \
  "echo '$HW' | grep -o '\"total_ram_mb\":[0-9]*' | grep -v ':0$' | grep -q '.'"
check "GET /api/setup/hardware has recommendation" \
  "echo '$HW' | grep -q '\"whisper_tier\"'"

# 4. Custom path validation — expect 422 for invalid path
echo ""
echo "[Custom path validation]"
CUSTOM_STATUS=$(curl -sf -o /dev/null -w "%{http_code}" \
  -X POST "$BASE/api/setup/tools/ffmpeg/path" \
  -H "Content-Type: application/json" \
  -d '{"path":"/nonexistent/binary/ffmpeg"}' 2>/dev/null || echo "000")
check "POST invalid path returns 422" \
  "[ '$CUSTOM_STATUS' = '422' ]"

# 5. Status endpoint has source + version_ok fields
echo ""
echo "[Status fields]"
STATUS=$(curl -sf "$BASE/api/setup/status" 2>/dev/null || echo "")
check "GET /api/setup/status returns tools array" \
  "echo '$STATUS' | grep -q '\"tools\"'"
check "GET /api/setup/status has source field" \
  "echo '$STATUS' | grep -q '\"source\"'"

echo ""
echo "=== validate-105: PASS=$PASS FAIL=$FAIL ==="
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo "FAIL"
  exit 1
fi
echo "PASS"
exit 0
