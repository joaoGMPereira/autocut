#!/bin/bash
# ──────────────────────────────────────────────────────────────────
# release-windows.sh — Cross-compile AutoCut para Windows
#                       a partir do macOS (sem code signing)
#
# Gera um instalador NSIS (.exe) + ZIP portátil.
# Distribuição interna — sem necessidade de code signing.
#
# Suporta x64 (padrão) e arm64:
#   ./scripts/release-windows.sh            # x64 (default)
#   ./scripts/release-windows.sh --arm64    # arm64
#   ./scripts/release-windows.sh --x64      # x64 (explícito)
#   WIN_ARCH=arm64 make release-windows     # via Makefile
#
# Pré-requisitos:
#   1. Go instalado (cross-compile server para windows)
#   2. pnpm instalado
#
# electron-builder suporta cross-build Windows de macOS nativamente
# (NSIS compiler funciona em todas as plataformas, sem Wine).
# ──────────────────────────────────────────────────────────────────

set -euo pipefail

# ─── Parse args ───────────────────────────────────────────────────
WIN_ARCH="${WIN_ARCH:-x64}"
for arg in "$@"; do
  case "$arg" in
    --arm64) WIN_ARCH="arm64" ;;
    --x64)   WIN_ARCH="x64" ;;
  esac
done

# Map arch: x64→amd64 (Go), arm64→arm64 (Go); electron-builder flags
case "$WIN_ARCH" in
  x64)   GO_ARCH="amd64"; EB_ARCH="--x64" ;;
  arm64) GO_ARCH="arm64"; EB_ARCH="--arm64" ;;
  *)     echo "❌ Arch desconhecida: $WIN_ARCH (suportadas: x64, arm64)"; exit 1 ;;
esac

# ─── Cores ────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RELEASE_DIR="$ROOT_DIR/apps/desktop/${RELEASE_OUTPUT_DIR:-release}"

# ─── Funções auxiliares ───────────────────────────────────────────

header() {
  echo ""
  echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
  echo -e "${CYAN}║${NC}  ${BOLD}AutoCut — Windows Release (${WIN_ARCH}, cross-compile)${NC}  ${CYAN}║${NC}"
  echo -e "${CYAN}║${NC}  ${DIM}NSIS installer + ZIP portátil (sem code signing)${NC}       ${CYAN}║${NC}"
  echo -e "${CYAN}╚══════════════════════════════════════════════════════════╝${NC}"
  echo ""
}

step() {
  echo -e "${CYAN}→${NC} ${BOLD}$1${NC}"
}

success() {
  echo -e "  ${GREEN}✔${NC} $1"
}

warn() {
  echo -e "  ${YELLOW}⚠${NC} $1"
}

fail() {
  echo -e "  ${RED}✖ $1${NC}"
  exit 1
}

info() {
  echo -e "  ${DIM}$1${NC}"
}

# ─── Validação de pré-requisitos ──────────────────────────────────

validate() {
  step "1/6 Validando pré-requisitos..."

  if ! command -v go &>/dev/null; then
    fail "Go não está instalado. Instale em: https://go.dev/dl/"
  fi
  success "Go $(go version | awk '{print $3}')"

  if ! command -v pnpm &>/dev/null; then
    fail "pnpm não encontrado. Instale com: npm install -g pnpm"
  fi
  success "pnpm $(pnpm --version)"

  if [[ ! -f "$ROOT_DIR/server/go.mod" ]]; then
    fail "Source do server não encontrado em $ROOT_DIR/server/"
  fi
  success "server source encontrado"

  info "CGO_ENABLED=0 (cross-compile Go puro, sem dependências C)"

  echo ""
}

# ─── Build pipeline ───────────────────────────────────────────────

build_go_server_windows() {
  step "2/6 Cross-compile server (windows/${GO_ARCH})..."
  local server_src="$ROOT_DIR/server"

  mkdir -p "$ROOT_DIR/apps/desktop/bin"
  rm -f "$ROOT_DIR/apps/desktop/bin/server.exe"

  # Cross-compile com CGO_ENABLED=0 (trivial para Go puro)
  CGO_ENABLED=0 GOOS=windows GOARCH="$GO_ARCH" go build \
    -C "$server_src" \
    -trimpath -ldflags="-s -w" \
    -o "$ROOT_DIR/apps/desktop/bin/server.exe" \
    ./cmd/server

  # Verificar que é PE executable (macOS `file` reconhece PE)
  local file_info
  file_info=$(file "$ROOT_DIR/apps/desktop/bin/server.exe" 2>/dev/null || echo "unknown")
  if [[ "$GO_ARCH" == "amd64" ]]; then
    if echo "$file_info" | grep -qi "PE32+.*x86-64"; then
      success "server.exe verificado: PE32+ x86-64 (Windows amd64)"
    elif echo "$file_info" | grep -qi "PE32"; then
      warn "server.exe detectado como PE32 (32-bit) — esperado PE32+ (64-bit)"
    else
      warn "Formato não verificável automaticamente: $file_info"
      info "O binário pode estar correto — verificar manualmente em Windows"
    fi
  elif [[ "$GO_ARCH" == "arm64" ]]; then
    if echo "$file_info" | grep -qi "Aarch64\|ARM64"; then
      success "server.exe verificado: PE32+ ARM64 (Windows arm64)"
    else
      warn "Formato: $file_info"
      info "Go cross-compile arm64 — verificar em Windows ARM se necessário"
    fi
  fi

  local bin_size
  bin_size=$(du -h "$ROOT_DIR/apps/desktop/bin/server.exe" | cut -f1)
  success "server.exe compilado ($bin_size, windows/${GO_ARCH})"
  echo ""
}

build_web_standalone() {
  step "3/6 Build web (Next.js standalone)..."
  NEXT_OUTPUT=standalone pnpm --filter @autocut/web build
  success "Next.js build concluído"
  echo ""
}

prepare_standalone_package() {
  step "4/6 Preparar pacote standalone..."

  rm -rf apps/web/standalone-pkg
  rsync -aL apps/web/.next/standalone/ apps/web/standalone-pkg/

  for dep in react react-dom styled-jsx postcss @next @swc caniuse-lite; do
    cp -R apps/web/standalone-pkg/node_modules/.pnpm/next@*/node_modules/$dep \
      apps/web/standalone-pkg/apps/web/node_modules/ 2>/dev/null || true
  done

  rm -rf apps/web/standalone-pkg/node_modules

  mkdir -p apps/web/standalone-pkg/apps/web/.next/static
  cp -R apps/web/.next/static/* apps/web/standalone-pkg/apps/web/.next/static/

  if [ -d "apps/web/public" ]; then
    mkdir -p apps/web/standalone-pkg/apps/web/public
    cp -R apps/web/public/* apps/web/standalone-pkg/apps/web/public/
  fi

  success "Pacote standalone pronto (apps/web/standalone-pkg/)"
  echo ""
}

build_desktop() {
  step "5/6 Build desktop (esbuild)..."
  pnpm --filter @autocut/desktop build
  success "Desktop build concluído"
  echo ""
}

package_windows() {
  step "6/6 Package com electron-builder (Windows NSIS + ZIP, ${WIN_ARCH})..."
  info "Cross-building Windows installer de macOS (NSIS nativo, sem Wine)..."

  mkdir -p "$RELEASE_DIR"

  # Limpar artefatos Windows anteriores para evitar contaminação
  local _found_win_files
  _found_win_files=$(find "$RELEASE_DIR" -maxdepth 1 \( -name "*-win-*" -o -name "latest.yml" -o -name "latest-win-*.yml" -o -name "alpha.yml" -o -name "beta.yml" \) -type f 2>/dev/null)
  if [[ -n "$_found_win_files" ]]; then
    local _stash_dir="$RELEASE_DIR/.stash-win-$$"
    mkdir -p "$_stash_dir"
    local _moved=0
    while IFS= read -r f; do
      mv "$f" "$_stash_dir/"
      _moved=$((_moved + 1))
    done <<< "$_found_win_files"
    info "Artefatos Windows anteriores ($_moved) movidos para .stash-win/"
  fi

  # Configuração isolada para builds paralelos (output dir customizado)
  local _eb_config="electron-builder.yml"
  local _output_dir="${RELEASE_OUTPUT_DIR:-release}"
  if [[ "$_output_dir" != "release" ]]; then
    # mktemp sem sufixo — BSD mktemp do macOS não suporta XXXXXX.yml
    local _tmp_config
    _tmp_config=$(mktemp /tmp/eb-win-XXXXXX)
    sed "s|output: release\$|output: ${_output_dir}|" \
        "$ROOT_DIR/apps/desktop/electron-builder.yml" > "$_tmp_config"
    _eb_config="$_tmp_config"
    info "Config isolado: output=${_output_dir}"
  fi

  local dist_exit=0
  pnpm --filter @autocut/desktop exec electron-builder \
    --config "$_eb_config" --win ${EB_ARCH} || dist_exit=$?

  if [[ "$dist_exit" -ne 0 ]]; then
    warn "electron-builder falhou (exit $dist_exit)"
    fail "electron-builder falhou — verifique os logs acima"
  fi

  # Renomear YAML → latest-win-{arch}.yml (consistente com macOS: latest-mac-{arch}.yml)
  # O electron-builder gera: stable→latest.yml, prerelease→alpha.yml/beta.yml
  # Normalizamos SEMPRE para latest-win-{ARCH}.yml.
  local _win_yml_src=""
  if [[ -f "$RELEASE_DIR/latest.yml" ]]; then
    _win_yml_src="$RELEASE_DIR/latest.yml"
  else
    _win_yml_src=$(find "$RELEASE_DIR" -maxdepth 1 -name "*.yml" ! -name "latest-win-*.yml" ! -name "latest-mac-*.yml" ! -name "latest-mac.yml" -type f 2>/dev/null | head -1)
  fi
  if [[ -n "$_win_yml_src" ]]; then
    cp "$_win_yml_src" "$RELEASE_DIR/latest-win-${WIN_ARCH}.yml"
    success "$(basename "$_win_yml_src") → latest-win-${WIN_ARCH}.yml"
  else
    warn "Nenhum auto-update YAML encontrado em $RELEASE_DIR/ — auto-update Windows pode não funcionar"
  fi

  echo ""
  success "Package Windows concluído!"
  echo ""
}

# ─── Verificação pós-build ────────────────────────────────────────

verify_output() {
  echo -e "${CYAN}→${NC} ${BOLD}Verificando artefatos...${NC}"

  # NSIS installer
  local exe_file
  exe_file=$(find "$RELEASE_DIR" -maxdepth 1 -name "*-win-*-setup.exe" -o -name "*-win-setup.exe" | head -1)
  if [[ -n "$exe_file" ]]; then
    local exe_size
    exe_size=$(du -h "$exe_file" | cut -f1)
    success "NSIS installer: $(basename "$exe_file") ($exe_size)"
  else
    warn "Nenhum NSIS installer (.exe) encontrado"
  fi

  # ZIP portátil
  local zip_file
  zip_file=$(find "$RELEASE_DIR" -maxdepth 1 -name "*-win-*.zip" | head -1)
  if [[ -n "$zip_file" ]]; then
    local zip_size
    zip_size=$(du -h "$zip_file" | cut -f1)
    success "ZIP portátil: $(basename "$zip_file") ($zip_size)"
  fi

  # Blockmap files
  local blockmap_count
  blockmap_count=$(find "$RELEASE_DIR" -maxdepth 1 -name "*-win-*.blockmap" 2>/dev/null | wc -l | tr -d ' ')
  if [[ "$blockmap_count" -gt 0 ]]; then
    success "Blockmap files: ${blockmap_count} (differential updates habilitados)"
  fi

  # YAML de update
  if [[ -f "$RELEASE_DIR/latest-win-${WIN_ARCH}.yml" ]]; then
    local win_version
    win_version=$(grep '^version:' "$RELEASE_DIR/latest-win-${WIN_ARCH}.yml" | awk '{print $2}' | tr -d '[:space:]"'"'"'' || true)
    success "latest-win-${WIN_ARCH}.yml (v${win_version})"
  fi

  echo ""
  info "Artefatos Windows em $RELEASE_DIR/:"
  find "$RELEASE_DIR" -maxdepth 1 \( -name "*-win-*" -o -name "latest-win-*.yml" \) -type f -exec ls -lh {} \; 2>/dev/null | while read -r line; do
    info "  $line"
  done
}

# ─── Main ─────────────────────────────────────────────────────────

main() {
  cd "$ROOT_DIR"

  header
  validate

  # SKIP_SHARED_BUILD: usado pelos workers paralelos — partes compartilhadas
  # já foram compiladas pelo SHARED_BUILD_ONLY antes de lançar os workers.
  if [[ "${SKIP_SHARED_BUILD:-}" != "true" ]]; then
    pnpm install --frozen-lockfile 2>/dev/null || pnpm install
    build_web_standalone
    prepare_standalone_package
    build_desktop
  fi

  build_go_server_windows
  package_windows
  verify_output

  local _output_dir="${RELEASE_OUTPUT_DIR:-release}"
  echo ""
  echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║${NC}  ${BOLD}Release Windows concluído com sucesso!${NC}                 ${GREEN}║${NC}"
  echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "  Artefatos em: ${BOLD}apps/desktop/${_output_dir}/${NC}"
  echo -e "  ${DIM}(Sem code signing — distribuição interna)${NC}"
  echo ""
}

main "$@"
