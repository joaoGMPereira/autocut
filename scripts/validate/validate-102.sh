#!/usr/bin/env bash
# validate-102.sh — Backend validation for 102-pipeline-nav-back
# No Go changes in this feature; script confirms build still passes.
#
# Exit 0 = all checks pass. Exit 1 = at least one check failed.

set -e
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SERVER_DIR="$REPO_ROOT/server"
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

echo "=== validate-102.sh: Pipeline Navigation Back — Backend ==="

cd "$SERVER_DIR"

# 1. go build must pass (CGO_ENABLED=0) — no Go changes, just verify build stays green
if CGO_ENABLED=0 go build ./... 2>/dev/null; then
  check "CGO_ENABLED=0 go build ./..." "ok"
else
  CGO_ENABLED=0 go build ./... 2>&1 | head -30 || true
  check "CGO_ENABLED=0 go build ./..." "fail"
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo "FAIL"
  exit 1
fi
echo "PASS"
exit 0
