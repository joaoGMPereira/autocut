#!/usr/bin/env bash
# validate-02.sh — Backend validation for 002-download-pipeline-validation
# Checks: build, tests, stub removal, SetDownloadResult exists, URL validation
#
# Exit 0 = all checks pass. Exit 1 = at least one check failed.

set -e
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SERVER_DIR="$REPO_ROOT/server"
PASS=0
FAIL=0

check() {
  local desc="$1"
  local result="$2"  # "ok" or "fail"
  if [ "$result" = "ok" ]; then
    echo "  PASS  $desc"
    PASS=$((PASS+1))
  else
    echo "  FAIL  $desc"
    FAIL=$((FAIL+1))
  fi
}

echo "=== validate-02.sh: Download Pipeline — Backend ==="

# 1. go build must pass
if CGO_ENABLED=0 go build ./... 2>/dev/null; then
  check "CGO_ENABLED=0 go build ./..." "ok"
else
  (cd "$SERVER_DIR" && CGO_ENABLED=0 go build ./... 2>&1 | head -20) || true
  check "CGO_ENABLED=0 go build ./..." "fail"
fi
cd "$SERVER_DIR"

# 2. go test must pass for in-scope packages (other packages have pre-existing stub failures)
IN_SCOPE="./internal/downloader/... ./internal/api/... ./internal/api/handlers/... ./internal/pipeline/... ./internal/database/..."
if CGO_ENABLED=0 go test $IN_SCOPE -timeout 30s 2>/dev/null; then
  check "go test (downloader, api, pipeline, database)" "ok"
else
  check "go test (downloader, api, pipeline, database)" "fail"
fi

# 3. No TODO stubs remain in pipeline/ package
STUB_COUNT=$(grep -r "TODO: Transcribe from Kotlin source" "$SERVER_DIR/internal/pipeline/" 2>/dev/null | wc -l | tr -d ' ')
if [ "$STUB_COUNT" -eq 0 ]; then
  check "No TODO stub files in pipeline/" "ok"
else
  check "No TODO stub files in pipeline/ (found $STUB_COUNT)" "fail"
fi

# 4. SetDownloadResult exists in repo
if grep -q "SetDownloadResult" "$SERVER_DIR/internal/database/pipeline_run_repo.go"; then
  check "SetDownloadResult exists in pipeline_run_repo.go" "ok"
else
  check "SetDownloadResult exists in pipeline_run_repo.go" "fail"
fi

# 5. Migration v10 exists
if grep -q "version: 10" "$SERVER_DIR/internal/database/migrations.go"; then
  check "Migration v10 declared in migrations.go" "ok"
else
  check "Migration v10 declared in migrations.go" "fail"
fi

# 6. video_title and duration_sec in v10 migration
if grep -q "video_title" "$SERVER_DIR/internal/database/migrations.go" && \
   grep -q "duration_sec" "$SERVER_DIR/internal/database/migrations.go"; then
  check "Migration v10 adds video_title and duration_sec" "ok"
else
  check "Migration v10 adds video_title and duration_sec" "fail"
fi

# 7. validateDownloadURL exists in service
if grep -q "validateDownloadURL" "$SERVER_DIR/internal/pipeline/service.go"; then
  check "validateDownloadURL function exists in service.go" "ok"
else
  check "validateDownloadURL function exists in service.go" "fail"
fi

# 8. Context cancel map (sync.Map) used in service
if grep -q "cancelMap" "$SERVER_DIR/internal/pipeline/service.go"; then
  check "cancelMap (context cancellation) in service.go" "ok"
else
  check "cancelMap (context cancellation) in service.go" "fail"
fi

# 9. StreamingExecutor interface exists
if grep -q "StreamingExecutor" "$SERVER_DIR/internal/downloader/executor.go"; then
  check "StreamingExecutor interface in executor.go" "ok"
else
  check "StreamingExecutor interface in executor.go" "fail"
fi

# 10. DownloadWithContext exists
if grep -q "DownloadWithContext" "$SERVER_DIR/internal/downloader/youtube.go"; then
  check "DownloadWithContext method in youtube.go" "ok"
else
  check "DownloadWithContext method in youtube.go" "fail"
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo "FAIL"
  exit 1
fi
echo "PASS"
exit 0
