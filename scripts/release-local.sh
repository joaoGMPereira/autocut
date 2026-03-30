#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────
# release-local.sh — Pipeline completa de release local
#
# Simula o GitHub Actions release.yml sem gastar Actions minutes.
# Lê credenciais de .env.local na raiz do monorepo.
#
# Uso:
#   ./scripts/release-local.sh                          # arm64 + x64 + windows (sequencial)
#   ./scripts/release-local.sh --parallel               # arm64 + x64 + windows (paralelo)
#   ./scripts/release-local.sh --mac                    # só macOS (arm64 + x64)
#   ./scripts/release-local.sh --arm64                  # só arm64
#   ./scripts/release-local.sh --x64                    # só x64
#   ./scripts/release-local.sh --windows                # só windows
#   ./scripts/release-local.sh --targets arm64,windows  # targets específicos
#   ./scripts/release-local.sh --resume                 # retoma: pula builds já prontos
#   ./scripts/release-local.sh --no-publish             # build sem publicar
#
# Em modo --parallel, as partes compartilhadas (Next.js, standalone-pkg,
# desktop esbuild) são compiladas uma única vez antes de lançar os workers.
# Cada target roda em background com server e electron-builder isolados.
# Recomendado em máquinas com 16GB+ RAM.
# ──────────────────────────────────────────────────────────────────

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# ─── Cores ────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

step()    { echo -e "\n${CYAN}${BOLD}▶ $*${NC}"; }
success() { echo -e "${GREEN}✔ $*${NC}"; }
warn()    { echo -e "${YELLOW}⚠ $*${NC}"; }
fail()    { echo -e "${RED}✗ $*${NC}"; exit 1; }

# ─── Streaming prefix para modo paralelo ────────────────────────
prefix_stream() {
  local tag="$1" color="$2" log="$3"
  shift 3
  local nc
  nc=$(printf '\033[0m')
  "$@" 2>&1 | tee -a "$log" | awk -v c="$color" -v t="$tag" -v n="$nc" \
    '{printf "%s[%s]%s %s\n", c, t, n, $0; fflush()}'
}

# Cores com bytes ESC reais (awk não interpreta \033 literal do echo -e)
_PFX_CYAN=$(printf '\033[0;36m')
_PFX_MAGENTA=$(printf '\033[0;35m')
_PFX_YELLOW=$(printf '\033[1;33m')

# ─── Carregar .env.local ──────────────────────────────────────────
if [[ -f "$ROOT_DIR/.env.local" ]]; then
  while IFS='=' read -r key value; do
    [[ "$key" =~ ^[[:space:]]*# ]] && continue
    [[ -z "$key" ]] && continue
    value="${value%\"}"
    value="${value#\"}"
    value="${value%\'}"
    value="${value#\'}"
    export "$key=$value"
  done < "$ROOT_DIR/.env.local"
  echo -e "${DIM}Credenciais carregadas de .env.local${NC}"
else
  warn ".env.local não encontrado — usando variáveis do ambiente"
fi

# ─── Parse args ───────────────────────────────────────────────────
BUILD_ARM64=true
BUILD_X64=true
BUILD_WINDOWS=true
NO_PUBLISH=false
RESUME=false
PARALLEL=false

for arg in "$@"; do
  case "$arg" in
    --arm64)      BUILD_ARM64=true;  BUILD_X64=false; BUILD_WINDOWS=false ;;
    --x64)        BUILD_ARM64=false; BUILD_X64=true;  BUILD_WINDOWS=false ;;
    --mac)        BUILD_ARM64=true;  BUILD_X64=true;  BUILD_WINDOWS=false ;;
    --windows)    BUILD_ARM64=false; BUILD_X64=false; BUILD_WINDOWS=true  ;;
    --no-publish) NO_PUBLISH=true ;;
    --resume)     RESUME=true ;;
    --parallel)   PARALLEL=true ;;
    --targets=*)
      # --targets=arm64,x64,windows (vírgula-separado)
      _targets="${arg#--targets=}"
      BUILD_ARM64=false; BUILD_X64=false; BUILD_WINDOWS=false
      IFS=',' read -ra _parts <<< "$_targets"
      for _t in "${_parts[@]}"; do
        case "$_t" in
          arm64)   BUILD_ARM64=true ;;
          x64)     BUILD_X64=true ;;
          windows) BUILD_WINDOWS=true ;;
          mac)     BUILD_ARM64=true; BUILD_X64=true ;;
          *) warn "Target desconhecido: '$_t' (suportados: arm64, x64, windows, mac)" ;;
        esac
      done
      ;;
  esac
done

if [[ "$NO_PUBLISH" == "true" ]]; then
  export NO_PUBLISH=true
  warn "Modo --no-publish: artifacts gerados mas não publicados"
fi

RELEASE_DIR="$ROOT_DIR/apps/desktop/release"

# ─── Contar targets ativos ───────────────────────────────────────
_active_targets=0
[[ "$BUILD_ARM64" == "true" ]] && _active_targets=$((_active_targets + 1))
[[ "$BUILD_X64" == "true" ]] && _active_targets=$((_active_targets + 1))
[[ "$BUILD_WINDOWS" == "true" ]] && _active_targets=$((_active_targets + 1))

# Desativar paralelo se só 1 target (não faz sentido)
if [[ "$PARALLEL" == "true" && "$_active_targets" -lt 2 ]]; then
  PARALLEL=false
  echo -e "${DIM}Paralelo desativado (só 1 target)${NC}"
fi

# ─── Bump de versão ───────────────────────────────────────────────
CURRENT=$(node -p "require('$ROOT_DIR/apps/desktop/package.json').version")

# Auto-resume: se a tag existe MAS o release ainda não tem os artefatos principais,
# reutiliza a versão atual (build falhou antes do upload).
if [[ "$RESUME" != "true" ]] && git rev-parse "v${CURRENT}" &>/dev/null; then
  _has_dmg=false
  if gh release view "v${CURRENT}" --repo "joaoGMPereira/autocut" --json assets --jq '.assets[].name' 2>/dev/null \
      | grep -q "AutoCut-${CURRENT}-arm64.dmg"; then
    _has_dmg=true
  fi

  if [[ "$_has_dmg" == "false" ]]; then
    warn "Tag v${CURRENT} já existe mas release incompleto (sem DMG) — entrando em modo resume automático"
    RESUME=true
  else
    echo -e "${DIM}Tag v${CURRENT} já existe e release está completo — prosseguindo com bump normal${NC}"
  fi
fi

if [[ "$RESUME" == "true" ]]; then
  NEW_VERSION="$CURRENT"
  echo -e "\n${BOLD}Modo resume — versão ${CYAN}${NEW_VERSION}${NC}"
  echo -e "${DIM}Pulando bump, commit e tag (já feitos)${NC}\n"
else
  IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT"

  echo -e "\n${BOLD}Versão atual: ${CYAN}${CURRENT}${NC}"
  echo -e "${DIM}Tipo de bump? [patch] / minor / major  (Enter = patch)${NC}"
  read -r BUMP_TYPE
  BUMP_TYPE="${BUMP_TYPE:-patch}"

  case "$BUMP_TYPE" in
    major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
    minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
    patch) PATCH=$((PATCH + 1)) ;;
    *) fail "Tipo inválido: '$BUMP_TYPE'. Use patch, minor ou major." ;;
  esac

  NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
  echo -e "${GREEN}${BOLD}→ ${CURRENT} → ${NEW_VERSION}${NC}\n"

  # Atualizar todos os package.json
  "$SCRIPT_DIR/bump-version.sh" patch <<< "" 2>/dev/null || \
  node -e "
    const fs = require('fs');
    const files = ['apps/desktop/package.json','apps/web/package.json','packages/shared/package.json'];
    files.forEach(f => {
      const p = JSON.parse(fs.readFileSync('$ROOT_DIR/' + f, 'utf8'));
      p.version = '$NEW_VERSION';
      fs.writeFileSync('$ROOT_DIR/' + f, JSON.stringify(p, null, 2) + '\n');
    });
  "

  # Commit + tag
  cd "$ROOT_DIR"
  git add apps/desktop/package.json apps/web/package.json packages/shared/package.json
  git commit -m "chore: bump version to ${NEW_VERSION}"
  git tag "v${NEW_VERSION}"
  git push origin main
  git push origin "v${NEW_VERSION}"
  success "Tag v${NEW_VERSION} criada e pushed"
fi

# ─── Criar GitHub Release antecipadamente ────────────────────────
# Cria o release com a tag correta ANTES dos workers do electron-builder.
# Se electron-builder rodar com --publish always e não encontrar um release
# existente, ele cria um "untagged-<sha>" — causando URL/tag errada.
VERSION="$NEW_VERSION"
if [[ -n "${GH_TOKEN:-}" ]] && command -v gh &>/dev/null && [[ "${NO_PUBLISH:-}" != "true" ]]; then
  if ! gh release view "v${VERSION}" --repo "joaoGMPereira/autocut" &>/dev/null; then
    if gh release create "v${VERSION}" \
        --repo "joaoGMPereira/autocut" \
        --title "v${VERSION}" \
        --notes "v${VERSION}" \
        --draft=false 2>/dev/null; then
      success "GitHub Release v${VERSION} criado antecipadamente"
    else
      warn "Falha ao criar GitHub Release antecipadamente — electron-builder tentará criar"
    fi
  else
    success "GitHub Release v${VERSION} já existe"
  fi
fi

# ─── Header ───────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}${BOLD}╔══════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}${BOLD}║${NC}  ${BOLD}AutoCut v${VERSION} — Release Local Pipeline${NC}             ${CYAN}${BOLD}║${NC}"
echo -e "${CYAN}${BOLD}╚══════════════════════════════════════════════════════════╝${NC}"
if [[ "$PARALLEL" == "true" ]]; then
  echo -e "  ${BOLD}Modo: ${CYAN}paralelo${NC} (${_active_targets} targets)"
else
  echo -e "  ${BOLD}Modo: ${DIM}sequencial${NC}"
fi
echo ""

# ─── Helper de formatação ─────────────────────────────────────────
result_label() {
  if [[ "$1" == "success" ]]; then
    echo -e "${GREEN}✔ success${NC}"
  else
    echo -e "${RED}✗ failure${NC}"
  fi
}

# ─── Tracking de resultados ───────────────────────────────────────
RESULT_ARM64="skipped"
RESULT_X64="skipped"
RESULT_WINDOWS="skipped"

# ─── Resume: detectar artefatos já existentes ─────────────────────
if [[ "$RESUME" == "true" ]]; then
  if [[ "$BUILD_ARM64" == "true" && -f "$RELEASE_DIR/AutoCut-${VERSION}-arm64.dmg" ]]; then
    RESULT_ARM64="success"
    BUILD_ARM64=false
    success "macOS arm64 — artefato já existe, pulando build"
  fi
  if [[ "$BUILD_X64" == "true" && -f "$RELEASE_DIR/AutoCut-${VERSION}.dmg" ]]; then
    RESULT_X64="success"
    BUILD_X64=false
    success "macOS x64 — artefato já existe, pulando build"
  fi
  if [[ "$BUILD_WINDOWS" == "true" ]] && ls "$RELEASE_DIR"/AutoCut-*"${VERSION}"*-win-*-setup.exe &>/dev/null 2>&1; then
    RESULT_WINDOWS="success"
    BUILD_WINDOWS=false
    success "Windows x64 — artefato já existe, pulando build"
  fi
fi

# ─── Build: Paralelo ou Sequencial ──────────────────────────────

if [[ "$PARALLEL" == "true" ]]; then
  # ═══════════════════════════════════════════════════════════════
  # MODO PARALELO
  #
  # 1. Pre-build: Next.js standalone + standalone-pkg + desktop esbuild
  #    (compartilhado entre todos os targets — compilado uma vez só)
  # 2. Cada target roda em background com:
  #    - SKIP_SHARED_BUILD=true (pula steps compartilhados)
  #    - SERVER_BINARY_NAME=server-{arch} (binários Go isolados por arch)
  #    - RELEASE_OUTPUT_DIR=release-{target} (output isolado)
  # 3. Merge: artefatos copiados para apps/desktop/release/
  #
  # Recomendado: 16GB+ RAM
  # ═══════════════════════════════════════════════════════════════

  step "Pre-build: partes compartilhadas (Next.js, standalone-pkg, desktop)..."
  SHARED_BUILD_ONLY=true ARCH=arm64 "$SCRIPT_DIR/release-macos.sh"
  success "Shared build concluído"

  # Diretório para logs de cada worker
  LOG_DIR=$(mktemp -d)
  echo -e "${DIM}Logs em: ${LOG_DIR}${NC}"

  # ── Lançar workers em background ─────────────────────────────────
  _pids=()
  _targets_launched=()

  if [[ "$BUILD_ARM64" == "true" ]]; then
    step "Lançando worker: macOS arm64"
    prefix_stream "arm64" "$_PFX_CYAN" "$LOG_DIR/arm64.log" \
      env SKIP_SHARED_BUILD=true SERVER_BINARY_NAME=server-arm64 \
          RELEASE_OUTPUT_DIR=release-arm64 ARCH=arm64 \
      "$SCRIPT_DIR/release-macos.sh" &
    _pids+=($!)
    _targets_launched+=("arm64")
    echo -e "  ${DIM}PID $!${NC}"
  fi

  if [[ "$BUILD_X64" == "true" ]]; then
    step "Lançando worker: macOS x64"
    prefix_stream "x64" "$_PFX_MAGENTA" "$LOG_DIR/x64.log" \
      env SKIP_SHARED_BUILD=true SERVER_BINARY_NAME=server-x64 \
          RELEASE_OUTPUT_DIR=release-x64 ARCH=x64 \
      "$SCRIPT_DIR/release-macos.sh" &
    _pids+=($!)
    _targets_launched+=("x64")
    echo -e "  ${DIM}PID $!${NC}"
  fi

  if [[ "$BUILD_WINDOWS" == "true" ]]; then
    step "Lançando worker: Windows x64"
    prefix_stream "win" "$_PFX_YELLOW" "$LOG_DIR/windows.log" \
      env SKIP_SHARED_BUILD=true RELEASE_OUTPUT_DIR=release-win \
          WIN_ARCH=x64 \
      "$SCRIPT_DIR/release-windows.sh" &
    _pids+=($!)
    _targets_launched+=("windows")
    echo -e "  ${DIM}PID $!${NC}"
  fi

  echo ""
  echo -e "${BOLD}Workers iniciados (${#_pids[@]}) — output em tempo real:${NC}"
  echo -e "  ${_PFX_CYAN}[arm64]${NC}  ${_PFX_MAGENTA}[x64]${NC}  ${_PFX_YELLOW}[win]${NC}"

  # ── Aguardar workers e coletar resultados ──────────────────────
  for i in "${!_pids[@]}"; do
    _pid="${_pids[$i]}"
    _target="${_targets_launched[$i]}"

    if wait "$_pid"; then
      case "$_target" in
        arm64)   RESULT_ARM64="success" ;;
        x64)     RESULT_X64="success" ;;
        windows) RESULT_WINDOWS="success" ;;
      esac
      success "${_target} concluído (PID $_pid)"
    else
      case "$_target" in
        arm64)   RESULT_ARM64="failure" ;;
        x64)     RESULT_X64="failure" ;;
        windows) RESULT_WINDOWS="failure" ;;
      esac
      warn "${_target} falhou (PID $_pid) — log completo: $LOG_DIR/${_target}.log"
    fi
  done

  # ── Merge: copiar artefatos para release/ ──────────────────────
  step "Merge: consolidando artefatos em apps/desktop/release/..."
  mkdir -p "$RELEASE_DIR"

  # Remover artefatos de versões anteriores para evitar que upload pegue
  # arquivos velhos (ex: 0.0.1 ainda em release/ quando a versão nova é 0.0.2).
  find "$RELEASE_DIR" -maxdepth 1 \
    \( -name "*.dmg" -o -name "*.exe" -o -name "*.zip" -o -name "*.blockmap" -o -name "*.yml" \) \
    ! -name "*${VERSION}*" -type f -delete 2>/dev/null || true

  for _subdir in release-arm64 release-x64 release-win; do
    _src="$ROOT_DIR/apps/desktop/$_subdir"
    if [[ -d "$_src" ]]; then
      find "$_src" -maxdepth 1 -type f -exec cp {} "$RELEASE_DIR/" \;
      rm -rf "$_src"
      success "Artefatos de $_subdir mergeados"
    fi
  done

  # Cleanup binários temporários isolados
  rm -f "$ROOT_DIR/apps/desktop/bin/server-arm64"
  rm -f "$ROOT_DIR/apps/desktop/bin/server-x64"

else
  # ═══════════════════════════════════════════════════════════════
  # MODO SEQUENCIAL (comportamento padrão)
  # ═══════════════════════════════════════════════════════════════

  # ─── macOS arm64 ─────────────────────────────────────────────────
  if [[ "$BUILD_ARM64" == "true" ]]; then
    step "macOS arm64"
    if ARCH=arm64 "$SCRIPT_DIR/release-macos.sh"; then
      RESULT_ARM64="success"
      success "macOS arm64 concluído"
    else
      RESULT_ARM64="failure"
      warn "macOS arm64 falhou — continuando com próximos targets"
    fi
  fi

  # ─── macOS x64 ───────────────────────────────────────────────────
  if [[ "$BUILD_X64" == "true" ]]; then
    step "macOS x64"
    if ARCH=x64 "$SCRIPT_DIR/release-macos.sh"; then
      RESULT_X64="success"
      success "macOS x64 concluído"
    else
      RESULT_X64="failure"
      warn "macOS x64 falhou — continuando com próximos targets"
    fi
  fi

  # ─── Windows x64 ─────────────────────────────────────────────────
  if [[ "$BUILD_WINDOWS" == "true" ]]; then
    step "Windows x64"
    if WIN_ARCH=x64 "$SCRIPT_DIR/release-windows.sh"; then
      RESULT_WINDOWS="success"
      success "Windows x64 concluído"
    else
      RESULT_WINDOWS="failure"
      warn "Windows x64 falhou"
    fi
  fi
fi

# ─── Atualizar GitHub Release ─────────────────────────────────────
if [[ -n "${GH_TOKEN:-}" ]] && command -v gh &>/dev/null && [[ "$NO_PUBLISH" != "true" ]]; then
  step "Atualizando GitHub Release v${VERSION}..."

  # Garantir que a tag existe no remoto
  if ! git ls-remote --tags origin "refs/tags/v${VERSION}" | grep -q "refs/tags/v${VERSION}"; then
    git push origin "v${VERSION}" 2>/dev/null && success "Tag v${VERSION} pushed" || warn "Falha ao fazer push da tag v${VERSION}"
  fi

  # Garantir que o release existe (fallback)
  if ! gh release view "v${VERSION}" --repo "joaoGMPereira/autocut" &>/dev/null; then
    gh release create "v${VERSION}" \
      --repo "joaoGMPereira/autocut" \
      --title "v${VERSION}" \
      --notes "v${VERSION}" \
      --draft=false 2>/dev/null || warn "Falha ao criar release"
  fi

  # Corrigir releases "untagged-<sha>" criadas por electron-builder em race condition
  if gh release edit "v${VERSION}" \
      --repo "joaoGMPereira/autocut" \
      --title "v${VERSION}" \
      --tag "v${VERSION}" \
      --draft=false 2>/dev/null; then
    success "GitHub Release v${VERSION} atualizado"
  else
    warn "Falha ao atualizar release"
  fi

  # Upload artefatos Windows (electron-builder --win não faz upload automático)
  if [[ "$RESULT_WINDOWS" == "success" ]]; then
    _win_setup=$(find "$RELEASE_DIR" -maxdepth 1 -name "AutoCut-${VERSION}-win-*-setup.exe" -type f 2>/dev/null | head -1)
    if [[ -n "$_win_setup" ]]; then
      step "Uploading artefatos Windows para GitHub Release..."
      gh release upload "v${VERSION}" "$_win_setup" \
        --repo "joaoGMPereira/autocut" --clobber 2>/dev/null && \
        success "$(basename "$_win_setup") uploaded" || \
        warn "Falha ao fazer upload de $(basename "$_win_setup")"

      _win_zip=$(find "$RELEASE_DIR" -maxdepth 1 -name "AutoCut-${VERSION}-win-*.zip" -type f 2>/dev/null | head -1)
      [[ -n "$_win_zip" ]] && gh release upload "v${VERSION}" "$_win_zip" \
        --repo "joaoGMPereira/autocut" --clobber 2>/dev/null && \
        success "$(basename "$_win_zip") uploaded" || true

      _win_yml=$(find "$RELEASE_DIR" -maxdepth 1 -name "latest-win-*.yml" -type f 2>/dev/null | head -1)
      [[ -n "$_win_yml" ]] && gh release upload "v${VERSION}" "$_win_yml" \
        --repo "joaoGMPereira/autocut" --clobber 2>/dev/null || true
    fi
  fi
fi

# ─── Notificação Telegram ─────────────────────────────────────────
if [[ -n "${TG_BOT_TOKEN:-}" && -n "${TG_CHAT_ID:-}" ]]; then
  step "Notificação Telegram"

  all_success=true
  detail=""
  _pairs=("arm64:$RESULT_ARM64" "x64:$RESULT_X64" "windows:$RESULT_WINDOWS")
  for pair in "${_pairs[@]}"; do
    target_name="${pair%%:*}"
    result="${pair##*:}"
    [[ "$result" == "skipped" ]] && continue
    [[ "$result" != "success" ]] && all_success=false
    detail+=" ${target_name}:${result}"
  done

  if [[ "$all_success" == "true" ]]; then
    STATUS="✅ Sucesso"
  else
    STATUS="❌ Falhou —${detail}"
  fi

  TAG="v${VERSION}"
  URL="https://github.com/joaoGMPereira/autocut/releases/tag/${TAG}"
  tag_esc=$(echo "$TAG" | sed 's/[!.()\-]/\\&/g')
  url_esc=$(echo "$URL" | sed 's/[!.()\-]/\\&/g')
  MSG="🚀 *AutoCut ${tag_esc} publicado\!*%0A${STATUS}%0A${url_esc}"

  response=$(curl -s -X POST "https://api.telegram.org/bot${TG_BOT_TOKEN}/sendMessage" \
    -d "chat_id=${TG_CHAT_ID}&text=${MSG}&parse_mode=MarkdownV2")

  if echo "$response" | grep -q '"ok":true'; then
    success "Telegram notificado"
  else
    warn "Telegram falhou: $response"
  fi
else
  warn "TG_BOT_TOKEN ou TG_CHAT_ID não definidos — notificação Telegram pulada"
fi

# ─── Resumo ───────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}─── Resultado ────────────────────────────────────────────────${NC}"
[[ "$RESULT_ARM64"   != "skipped" ]] && echo -e "  macOS arm64 : $(result_label "$RESULT_ARM64")"
[[ "$RESULT_X64"     != "skipped" ]] && echo -e "  macOS x64   : $(result_label "$RESULT_X64")"
[[ "$RESULT_WINDOWS" != "skipped" ]] && echo -e "  Windows x64 : $(result_label "$RESULT_WINDOWS")"
echo ""
success "Pipeline concluída — v${VERSION}"
