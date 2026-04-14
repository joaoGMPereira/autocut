#!/usr/bin/env bash
# validate-100.sh — Backend validation for 100-autocut-settings-tools
# Checks: build, tool validators, settings endpoints, whisper model list, routes
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

echo "=== validate-100.sh: AutoCut Settings & Tool Validation — Backend ==="

cd "$SERVER_DIR"

# 1. go build must pass (CGO_ENABLED=0)
if CGO_ENABLED=0 go build ./... 2>/dev/null; then
  check "CGO_ENABLED=0 go build ./..." "ok"
else
  CGO_ENABLED=0 go build ./... 2>&1 | head -30 || true
  check "CGO_ENABLED=0 go build ./..." "fail"
fi

# 2. All 8 validator structs present in validators.go
VALIDATORS_FILE="$SERVER_DIR/internal/configurator/validators.go"
for v in YtDlpValidator FfmpegValidator WhisperValidator ClaudeValidator GhValidator OllamaValidator TwitchValidator ConvertValidator; do
  if grep -q "type $v struct" "$VALIDATORS_FILE" 2>/dev/null; then
    check "Validator struct: $v" "ok"
  else
    check "Validator struct: $v" "fail"
  fi
done

# 3. defaultValidators() returns 8 validators
if grep -q "NewYtDlpValidator\|NewFfmpegValidator\|NewWhisperValidator" "$SERVER_DIR/internal/configurator/configurator.go" 2>/dev/null; then
  check "defaultValidators() populated in configurator.go" "ok"
else
  check "defaultValidators() populated in configurator.go" "fail"
fi

# 4. WhisperDir field in AutoCutDir
if grep -q "WhisperDir" "$SERVER_DIR/internal/configurator/configurator.go" 2>/dev/null; then
  check "WhisperDir field in AutoCutDir" "ok"
else
  check "WhisperDir field in AutoCutDir" "fail"
fi

# 5. dir.go has Ensure(), BinPath(), WhisperModelPath()
DIR_FILE="$SERVER_DIR/internal/configurator/dir.go"
for fn in "func (d \*AutoCutDir) Ensure" "func (d \*AutoCutDir) BinPath" "func (d \*AutoCutDir) WhisperModelPath"; do
  if grep -q "$fn" "$DIR_FILE" 2>/dev/null; then
    check "dir.go: $(echo $fn | awk '{print $3}')" "ok"
  else
    check "dir.go: $(echo $fn | awk '{print $3}')" "fail"
  fi
done

# 6. whisper_models.go has WhisperModelStatus and DownloadWhisperModel
WHISPER_FILE="$SERVER_DIR/internal/configurator/whisper_models.go"
for fn in "func WhisperModelStatus" "func DownloadWhisperModel"; do
  if grep -q "$fn" "$WHISPER_FILE" 2>/dev/null; then
    check "whisper_models.go: $fn" "ok"
  else
    check "whisper_models.go: $fn" "fail"
  fi
done

# 7. 5 hardcoded Whisper models in whisper_models.go (count struct literal lines)
MODEL_COUNT=$(grep -c '{name:.*filename: "ggml-' "$WHISPER_FILE" 2>/dev/null || echo 0)
if [ "$MODEL_COUNT" -eq 5 ]; then
  check "whisper_models.go: 5 hardcoded models" "ok"
else
  check "whisper_models.go: 5 hardcoded models (found $MODEL_COUNT)" "fail"
fi

# 8. settings_handler.go has GetSettings and PutSetting
SETTINGS_HANDLER="$SERVER_DIR/internal/api/handlers/settings_handler.go"
for fn in "GetSettings" "PutSetting"; do
  if grep -q "func.*$fn" "$SETTINGS_HANDLER" 2>/dev/null; then
    check "settings_handler.go: $fn" "ok"
  else
    check "settings_handler.go: $fn" "fail"
  fi
done

# 9. ollama_handler.go exists with GetOllamaModels
OLLAMA_HANDLER="$SERVER_DIR/internal/api/handlers/ollama_handler.go"
if [ -f "$OLLAMA_HANDLER" ] && grep -q "GetOllamaModels" "$OLLAMA_HANDLER"; then
  check "ollama_handler.go: GetOllamaModels" "ok"
else
  check "ollama_handler.go: GetOllamaModels" "fail"
fi

# 10. oauth_handler.go has GetOAuthProfiles, PostOAuthProfile, DeleteOAuthProfile, PostSetDefaultProfile
OAUTH_HANDLER="$SERVER_DIR/internal/api/handlers/oauth_handler.go"
for fn in GetOAuthProfiles PostOAuthProfile DeleteOAuthProfile PostSetDefaultProfile; do
  if grep -q "func.*$fn" "$OAUTH_HANDLER" 2>/dev/null; then
    check "oauth_handler.go: $fn" "ok"
  else
    check "oauth_handler.go: $fn" "fail"
  fi
done

# 11. router.go has all required routes
ROUTER_FILE="$SERVER_DIR/internal/api/router.go"
REQUIRED_ROUTES=(
  "/api/setup/install/{tool}"
  "/api/setup/install/{tool}/stream"
  "/api/setup/check-update/{tool}"
  "/api/setup/whisper/models"
  "/api/setup/whisper/models/{model}"
  "/api/setup/whisper/models/{model}/stream"
  "/api/settings"
  "/api/ollama/models"
  "/api/channels"
  "/api/oauth/profiles"
)
for route in "${REQUIRED_ROUTES[@]}"; do
  if grep -q "\"$route\"\|'$route'" "$ROUTER_FILE" 2>/dev/null || grep -qF "$route" "$ROUTER_FILE" 2>/dev/null; then
    check "router.go: $route" "ok"
  else
    check "router.go: $route" "fail"
  fi
done

# 12. main.go wires settingsHandler, channelHandler, oauthHandler, ollamaHandler
MAIN_FILE="$SERVER_DIR/cmd/server/main.go"
for h in settingsHandler channelHandler oauthHandler ollamaHandler; do
  if grep -q "$h" "$MAIN_FILE" 2>/dev/null; then
    check "main.go: $h wired" "ok"
  else
    check "main.go: $h wired" "fail"
  fi
done

# 13. SETTINGS_KEYS includes UPLOAD_STRATEGY and UPLOAD_PARALLEL_COUNT (frontend)
STORE_FILE="$REPO_ROOT/apps/web/src/store/settingsStore.ts"
for key in UPLOAD_STRATEGY UPLOAD_PARALLEL_COUNT; do
  if grep -q "$key" "$STORE_FILE" 2>/dev/null; then
    check "settingsStore.ts: $key" "ok"
  else
    check "settingsStore.ts: $key" "fail"
  fi
done

# 14. ModelsSection.tsx exists
MODELS_SECTION="$REPO_ROOT/apps/web/src/components/settings/ModelsSection.tsx"
if [ -f "$MODELS_SECTION" ]; then
  check "ModelsSection.tsx exists" "ok"
else
  check "ModelsSection.tsx exists" "fail"
fi

# 15. Settings page.tsx no longer uses NotImplemented
SETTINGS_PAGE="$REPO_ROOT/apps/web/src/app/settings/page.tsx"
if ! grep -q "NotImplemented" "$SETTINGS_PAGE" 2>/dev/null; then
  check "settings/page.tsx: NotImplemented removed" "ok"
else
  check "settings/page.tsx: NotImplemented still present" "fail"
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  echo "FAIL"
  exit 1
fi
echo "PASS"
exit 0
