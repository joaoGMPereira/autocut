#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
SERVER_DIR="$REPO_ROOT/server"
WEB_DIR="$REPO_ROOT/apps/web"

echo "=== FP-001 Frontend Validation ==="

# CHECK 1: download_handler.go is not a stub
HANDLER="$SERVER_DIR/internal/api/handlers/download_handler.go"
if grep -q "^// TODO" "$HANDLER"; then
  echo "FAIL: download_handler.go is still a stub"
  exit 1
fi
if ! grep -q "func NewDownloadHandler" "$HANDLER"; then
  echo "FAIL: NewDownloadHandler not found in download_handler.go"
  exit 1
fi
echo "CHECK 1 PASS: download_handler.go is implemented"

# CHECK 2: handler tests pass
echo "CHECK 2: running handler tests"
cd "$SERVER_DIR" && go test ./internal/api/handlers/... -run "TestPostDownload|TestGetDownloadStream" -v
echo "CHECK 2 PASS: handler tests pass"

# CHECK 3: SSEVideoInfoPayload type exists in TS
if ! grep -q "SSEVideoInfoPayload" "$WEB_DIR/src/types/download.ts"; then
  echo "FAIL: SSEVideoInfoPayload not found in types/download.ts"
  exit 1
fi
echo "CHECK 3 PASS: SSEVideoInfoPayload type exists"

# CHECK 4: /download-test page exists
PAGE="$WEB_DIR/src/app/download-test/page.tsx"
if [ ! -f "$PAGE" ]; then
  echo "FAIL: /download-test page not found at $PAGE"
  exit 1
fi
echo "CHECK 4 PASS: /download-test page exists"

# CHECK 5: TypeScript compiles
echo "CHECK 5: running tsc --noEmit"
cd "$WEB_DIR" && pnpm tsc --noEmit
echo "CHECK 5 PASS: TypeScript compilation clean"

echo ""
echo "PASS: FP-001 Frontend Integration is complete."
