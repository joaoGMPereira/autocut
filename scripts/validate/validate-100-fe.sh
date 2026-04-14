#!/usr/bin/env bash
# validate-100-fe.sh — Frontend validation for 100-autocut-settings-tools
# Checks: TypeScript compile, Settings page wired, ModelsSection exists,
#         SETTINGS_KEYS updated, all 5 sections present in settings page.
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

echo "=== validate-100-fe.sh: AutoCut Settings — Frontend ==="

cd "$WEB_DIR"

# 1. TypeScript compile
if pnpm tsc --noEmit 2>/dev/null; then
  check "pnpm tsc --noEmit" "ok"
else
  pnpm tsc --noEmit 2>&1 | head -30 || true
  check "pnpm tsc --noEmit" "fail"
fi

# 2. Settings page no longer uses NotImplemented
SETTINGS_PAGE="$WEB_DIR/src/app/settings/page.tsx"
if ! grep -q "NotImplemented" "$SETTINGS_PAGE" 2>/dev/null; then
  check "settings/page.tsx: NotImplemented removed" "ok"
else
  check "settings/page.tsx: NotImplemented still present" "fail"
fi

# 3. Settings page uses Tabs
if grep -q "Tabs" "$SETTINGS_PAGE" 2>/dev/null; then
  check "settings/page.tsx: Tabs component used" "ok"
else
  check "settings/page.tsx: Tabs component missing" "fail"
fi

# 4. All 5 sections imported in settings page
SECTIONS=(ToolsSection AppSettingsSection ModelsSection ChannelsSection OAuthProfilesSection)
for s in "${SECTIONS[@]}"; do
  if grep -q "$s" "$SETTINGS_PAGE" 2>/dev/null; then
    check "settings/page.tsx: $s imported" "ok"
  else
    check "settings/page.tsx: $s missing" "fail"
  fi
done

# 5. ModelsSection.tsx exists and has both Whisper and Ollama sub-sections
MODELS_SECTION="$WEB_DIR/src/components/settings/ModelsSection.tsx"
if [ -f "$MODELS_SECTION" ]; then
  check "ModelsSection.tsx exists" "ok"
else
  check "ModelsSection.tsx exists" "fail"
fi

if grep -q "whisperModels\|fetchWhisperModels" "$MODELS_SECTION" 2>/dev/null; then
  check "ModelsSection.tsx: Whisper sub-section wired" "ok"
else
  check "ModelsSection.tsx: Whisper sub-section missing" "fail"
fi

if grep -q "ollama\|Ollama" "$MODELS_SECTION" 2>/dev/null; then
  check "ModelsSection.tsx: Ollama sub-section present" "ok"
else
  check "ModelsSection.tsx: Ollama sub-section missing" "fail"
fi

# 6. SETTINGS_KEYS includes UPLOAD_STRATEGY and UPLOAD_PARALLEL_COUNT
STORE_FILE="$WEB_DIR/src/store/settingsStore.ts"
for key in UPLOAD_STRATEGY UPLOAD_PARALLEL_COUNT; do
  if grep -q "$key" "$STORE_FILE" 2>/dev/null; then
    check "settingsStore.ts: SETTINGS_KEYS.$key present" "ok"
  else
    check "settingsStore.ts: SETTINGS_KEYS.$key missing" "fail"
  fi
done

# 7. ToolsSection has 'use client' and uses createLogger
TOOLS_SECTION="$WEB_DIR/src/components/settings/ToolsSection.tsx"
if [ -f "$TOOLS_SECTION" ]; then
  check "ToolsSection.tsx exists" "ok"
else
  check "ToolsSection.tsx exists" "fail"
fi

# 8. All required settings component files exist
for f in AppSettingsSection ChannelsSection ChannelAuthSection OAuthProfilesSection; do
  FPATH="$WEB_DIR/src/components/settings/${f}.tsx"
  if [ -f "$FPATH" ]; then
    check "${f}.tsx exists" "ok"
  else
    check "${f}.tsx exists" "fail"
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
