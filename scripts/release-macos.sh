#!/bin/bash
# ──────────────────────────────────────────────────────────────────
# release-macos.sh — Build, code sign e notarização do AutoCut
#
# Gera um DMG assinado e notarizado para macOS (arm64 ou x64).
# Distribuição interna (fora da App Store).
#
# Pré-requisitos:
#   1. Xcode Command Line Tools instalado
#   2. Variáveis de ambiente configuradas (ver abaixo)
#
# ─── Variáveis de ambiente ────────────────────────────────────────
#
# Code Signing (uma das opções, em ordem de prioridade):
#   CSC_NAME                              — Identity no Keychain (auto-detectado se único)
#     ou
#   DEVELOPER_ID_APPLICATION_CERTIFICATE  — .p12 em base64 (importado num Keychain temporário)
#   DEVELOPER_ID_APPLICATION_PASSWORD     — Senha do .p12 (vazio se sem senha)
#     ou
#   CSC_LINK / CSC_KEY_PASSWORD           — Nomes nativos do electron-builder (caminho .p12)
#
# Notarização:
#   APPLE_ID                              — Email do Apple ID
#   APPLE_TEAM_ID                         — Team ID (ex: S96TK27X79)
#   APPLE_APP_SPECIFIC_PASSWORD           — App-specific password
#
# ─── Uso ──────────────────────────────────────────────────────────
#
#   ./scripts/release-macos.sh
#   make release-macos
#
# ──────────────────────────────────────────────────────────────────

set -euo pipefail

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

# ─── Arquitetura alvo ───────────────────────────────────────────
# Aceita via env var: ARCH=x64 ./scripts/release-macos.sh
# Valores válidos: arm64, x64 (default: arm64)
ARCH="${ARCH:-arm64}"

case "$ARCH" in
  arm64) GOARCH="arm64"; ELECTRON_ARCH="arm64" ;;
  x64)   GOARCH="amd64"; ELECTRON_ARCH="x64"   ;;
  *)     echo "❌ ARCH inválido: $ARCH (use arm64 ou x64)"; exit 1 ;;
esac

# ─── Funções auxiliares ───────────────────────────────────────────

header() {
  echo ""
  echo -e "${CYAN}╔══════════════════════════════════════════════════════════╗${NC}"
  echo -e "${CYAN}║${NC}  ${BOLD}AutoCut — macOS Release (${ARCH})${NC}                   ${CYAN}║${NC}"
  echo -e "${CYAN}║${NC}  ${DIM}Code Sign + Notarização + DMG${NC}                          ${CYAN}║${NC}"
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

# ─── Recursos temporários (limpos no EXIT) ────────────────────────
_TEMP_CERT_PATH=""
_TEMP_KEYCHAIN_PATH=""
_ORIGINAL_KEYCHAINS=""

cleanup_temp_resources() {
  # Remover arquivo .p12 temporário
  if [[ -n "$_TEMP_CERT_PATH" && -f "$_TEMP_CERT_PATH" ]]; then
    rm -f "$_TEMP_CERT_PATH"
  fi

  # Remover Keychain temporário e restaurar a search list original
  if [[ -n "$_TEMP_KEYCHAIN_PATH" && -f "$_TEMP_KEYCHAIN_PATH" ]]; then
    security delete-keychain "$_TEMP_KEYCHAIN_PATH" 2>/dev/null || true
    if [[ -n "$_ORIGINAL_KEYCHAINS" ]]; then
      # Restaurar a search list original (eval para expandir as aspas corretamente)
      eval security list-keychains -s "$_ORIGINAL_KEYCHAINS"
    fi
  fi
}
trap cleanup_temp_resources EXIT

# ─── Mapeamento de variáveis ─────────────────────────────────────
# Aceita nomes customizados e mapeia para o que o electron-builder espera.

map_env_vars() {
  # Prioridade: CSC_NAME (Keychain direto) > DEVELOPER_ID_APPLICATION_CERTIFICATE (base64)
  # > CSC_LINK (caminho para .p12 já existente)

  if [[ -z "${CSC_NAME:-}" ]]; then
    # Tentar auto-detectar "Developer ID Application" no Keychain
    local cert_name
    cert_name=$(security find-identity -v -p codesigning 2>/dev/null \
      | grep "Developer ID Application" \
      | head -1 \
      | sed 's/.*"Developer ID Application: \(.*\)".*/\1/' || true)

    if [[ -n "$cert_name" ]]; then
      export CSC_NAME="$cert_name"
      # Limpar CSC_LINK para que o electron-builder use CSC_NAME (Keychain)
      unset CSC_LINK 2>/dev/null || true
      unset CSC_KEY_PASSWORD 2>/dev/null || true
    else
      # Fallback: decodificar o .p12 de base64 e importar num Keychain temporário.
      #
      # Por que não usar CSC_LINK diretamente?
      # O electron-builder tenta importar o .p12 via security(1), mas em ambientes CI
      # isso frequentemente falha com "MAC verification failed" por problemas na
      # detecção de encoding/senha. A solução robusta é importar o certificado nós
      # mesmos num Keychain temporário e apontar CSC_NAME para a identity — assim o
      # electron-builder encontra o certificado pronto no Keychain sem tocar no .p12.
      if [[ -n "${DEVELOPER_ID_APPLICATION_CERTIFICATE:-}" && -z "${CSC_LINK:-}" ]]; then
        # ── 1. Decodificar base64 → arquivo .p12 temporário ────────
        local _tmp_base
        _tmp_base=$(mktemp /tmp/autocut-certXXXXXX)
        _TEMP_CERT_PATH="${_tmp_base}.p12"
        mv "$_tmp_base" "$_TEMP_CERT_PATH"

        # Remover espaços, tabs e quebras de linha antes de decodificar.
        # CIs frequentemente armazenam base64 com quebras de linha (PEM-style) que
        # causam "error decoding base64 input stream" quando passadas diretamente.
        if ! printf '%s' "${DEVELOPER_ID_APPLICATION_CERTIFICATE}" \
            | tr -d ' \t\r\n' \
            | base64 --decode > "$_TEMP_CERT_PATH" 2>/dev/null; then
          rm -f "$_TEMP_CERT_PATH"
          fail "Falha ao decodificar DEVELOPER_ID_APPLICATION_CERTIFICATE — verifique se o valor é um base64 válido de um arquivo .p12"
        fi
        if [[ ! -s "$_TEMP_CERT_PATH" ]]; then
          rm -f "$_TEMP_CERT_PATH"
          fail "Certificado .p12 decodificado está vazio — DEVELOPER_ID_APPLICATION_CERTIFICATE pode estar incorreto ou truncado"
        fi
        info "Certificado .p12 decodificado ($(du -h "$_TEMP_CERT_PATH" | cut -f1))"

        # ── 2. Criar Keychain temporário ───────────────────────────
        _TEMP_KEYCHAIN_PATH="/tmp/autocut-build-$$.keychain-db"
        local keychain_pass="autocut-build-$$"

        security create-keychain -p "$keychain_pass" "$_TEMP_KEYCHAIN_PATH"
        info "Keychain temporário criado: $_TEMP_KEYCHAIN_PATH"

        # Desbloquear e configurar para não bloquear automaticamente
        # (evita "User interaction is not allowed" durante o build)
        security unlock-keychain -p "$keychain_pass" "$_TEMP_KEYCHAIN_PATH"
        security set-keychain-settings -t 3600 -u "$_TEMP_KEYCHAIN_PATH"

        # ── 3. Importar o .p12 no Keychain temporário ─────────────
        # -P "" → senha vazia (o .p12 exportado sem senha)
        # -T /usr/bin/codesign → permitir codesign acessar sem prompt
        # -T /usr/bin/security → permitir security acessar sem prompt
        # -A → permitir acesso de qualquer aplicação (fallback para CI)
        local p12_password="${DEVELOPER_ID_APPLICATION_PASSWORD:-${CSC_KEY_PASSWORD:-}}"

        if ! security import "$_TEMP_CERT_PATH" \
            -k "$_TEMP_KEYCHAIN_PATH" \
            -P "$p12_password" \
            -T /usr/bin/codesign \
            -T /usr/bin/security \
            -A 2>/dev/null; then
          fail "Falha ao importar .p12 no Keychain — verifique se o certificado e a senha estão corretos"
        fi
        info "Certificado importado no Keychain temporário (senha: ${p12_password:-(vazia)})"

        # Autorizar codesign a acessar as chaves sem prompt interativo.
        # set-key-partition-list define a ACL "partition list" que o macOS 10.12+
        # exige para acesso programático — sem isso, codesign falha com
        # "User interaction is not allowed" em CI.
        security set-key-partition-list -S "apple-tool:,apple:,codesign:" \
          -s -k "$keychain_pass" "$_TEMP_KEYCHAIN_PATH" 2>/dev/null || true

        # ── 4. Adicionar Keychain temporário à search list ─────────
        # Salvar a lista original para restaurar no cleanup.
        _ORIGINAL_KEYCHAINS=$(security list-keychains -d user | tr -d '"' | tr '\n' ' ')

        # Inserir o temporário NO INÍCIO da lista para ter prioridade.
        # Manter o login.keychain para que codesign consiga resolver a cadeia de confiança.
        security list-keychains -d user -s "$_TEMP_KEYCHAIN_PATH" $(security list-keychains -d user | tr -d '"' | tr '\n' ' ')
        info "Keychain temporário adicionado à search list"

        # ── 5. Extrair identity e setar CSC_NAME ──────────────────
        local imported_identity
        imported_identity=$(security find-identity -v -p codesigning "$_TEMP_KEYCHAIN_PATH" 2>/dev/null \
          | grep "Developer ID Application" \
          | head -1 \
          | sed 's/.*"\(Developer ID Application: .*\)".*/\1/' || true)

        if [[ -z "$imported_identity" ]]; then
          # Mostrar o que foi encontrado para diagnóstico
          warn "Identities encontradas no Keychain temporário:"
          security find-identity -v -p codesigning "$_TEMP_KEYCHAIN_PATH" 2>/dev/null || true
          fail "Nenhuma identity 'Developer ID Application' encontrada no certificado importado"
        fi

        export CSC_NAME="$imported_identity"
        # Limpar CSC_LINK/CSC_KEY_PASSWORD — o electron-builder vai usar CSC_NAME (Keychain)
        unset CSC_LINK 2>/dev/null || true
        unset CSC_KEY_PASSWORD 2>/dev/null || true
        info "Identity: ${CSC_NAME}"

      elif [[ -n "${CSC_LINK:-}" ]]; then
        # CSC_LINK já definido externamente — usar CSC_KEY_PASSWORD se já setado, senão vazio
        export CSC_KEY_PASSWORD="${CSC_KEY_PASSWORD:-}"
      fi
    fi
  fi
}

# ─── Validação de pré-requisitos ──────────────────────────────────

validate_prerequisites() {
  step "1/7 Validando pré-requisitos..."

  # macOS?
  if [[ "$(uname -s)" != "Darwin" ]]; then
    fail "Este script só roda em macOS"
  fi
  success "macOS detectado ($(sw_vers -productVersion))"

  # Xcode CLI tools?
  if ! xcode-select -p &>/dev/null; then
    fail "Xcode Command Line Tools não instalado. Rode: xcode-select --install"
  fi
  success "Xcode CLI Tools instalado"

  # codesign + notarytool?
  if ! command -v codesign &>/dev/null; then
    fail "codesign não encontrado no PATH"
  fi
  if ! xcrun notarytool --version &>/dev/null; then
    fail "notarytool não disponível. Atualize o Xcode CLI Tools."
  fi
  success "codesign + notarytool disponíveis"

  # pnpm?
  if ! command -v pnpm &>/dev/null; then
    fail "pnpm não encontrado. Instale com: npm install -g pnpm"
  fi
  success "pnpm $(pnpm --version)"

  # ─── Certificado de code signing ────────────────────────────────
  if [[ -n "${CSC_LINK:-}" ]]; then
    # .p12 via arquivo (pode ou não ter senha — ambos são válidos)
    if [[ -n "${CSC_KEY_PASSWORD:-}" ]]; then
      success "Certificado via .p12 (com senha)"
    else
      success "Certificado via .p12 (sem senha)"
    fi
  elif [[ -n "${CSC_NAME:-}" ]]; then
    # Identity no Keychain
    if ! security find-identity -v -p codesigning | grep -q "$CSC_NAME"; then
      fail "Certificado '$CSC_NAME' não encontrado no Keychain"
    fi
    success "Certificado no Keychain: ${CSC_NAME}"
  else
    # Auto-detectar Developer ID Application no Keychain
    local cert_count
    cert_count=$(security find-identity -v -p codesigning | grep "Developer ID Application" | wc -l | tr -d ' ')

    if [[ "$cert_count" -eq 0 ]]; then
      echo ""
      echo -e "  ${RED}✖ Nenhum certificado encontrado.${NC}"
      echo ""
      echo -e "  ${BOLD}Opções:${NC}"
      echo ""
      echo -e "  ${CYAN}A) .p12 em base64 (CI/CD ou sem Keychain):${NC}"
      echo -e "     ${DIM}export DEVELOPER_ID_APPLICATION_CERTIFICATE=\"<base64>\"${NC}"
      echo -e "     ${DIM}export DEVELOPER_ID_APPLICATION_PASSWORD=\"<senha>\"${NC}"
      echo ""
      echo -e "  ${CYAN}B) Certificado no Keychain:${NC}"
      echo -e "     ${DIM}export CSC_NAME=\"Developer ID Application: Seu Nome (TEAMID)\"${NC}"
      echo ""
      exit 1
    elif [[ "$cert_count" -eq 1 ]]; then
      export CSC_NAME
      CSC_NAME=$(security find-identity -v -p codesigning | grep "Developer ID Application" | sed 's/.*"Developer ID Application: \(.*\)".*/\1/')
      success "Certificado auto-detectado: ${CSC_NAME}"
    else
      echo ""
      echo -e "  ${YELLOW}⚠ Múltiplos certificados encontrados:${NC}"
      security find-identity -v -p codesigning | grep "Developer ID Application" | while read -r line; do
        echo -e "    ${DIM}$line${NC}"
      done
      echo ""
      echo -e "  Defina qual usar:"
      echo -e "  ${DIM}export CSC_NAME=\"Developer ID Application: Seu Nome (TEAMID)\"${NC}"
      echo ""
      exit 1
    fi
  fi

  # ─── Sanitizar CSC_NAME ──────────────────────────────────────────
  # electron-builder exige que CSC_NAME NÃO tenha o prefixo "Developer ID Application:".
  # Se o usuário (ou CI) definiu com o prefixo, removemos automaticamente.
  if [[ "${CSC_NAME:-}" == "Developer ID Application:"* ]]; then
    CSC_NAME="${CSC_NAME#Developer ID Application: }"
    CSC_NAME="${CSC_NAME#Developer ID Application:}"
    export CSC_NAME
    success "Prefixo 'Developer ID Application:' removido de CSC_NAME → ${CSC_NAME}"
  fi

  # ─── Credenciais de notarização ─────────────────────────────────
  local notary_errors=0
  local notary_missing=""

  if [[ -z "${APPLE_ID:-}" ]]; then
    notary_missing+="     APPLE_ID\n"
    notary_errors=$((notary_errors + 1))
  fi

  if [[ -z "${APPLE_APP_SPECIFIC_PASSWORD:-}" ]]; then
    notary_missing+="     APPLE_APP_SPECIFIC_PASSWORD\n"
    notary_errors=$((notary_errors + 1))
  fi

  if [[ -z "${APPLE_TEAM_ID:-}" ]]; then
    notary_missing+="     APPLE_TEAM_ID\n"
    notary_errors=$((notary_errors + 1))
  fi

  if [[ "$notary_errors" -gt 0 ]]; then
    echo ""
    echo -e "  ${RED}✖ Credenciais de notarização faltando:${NC}"
    echo -e "$notary_missing"
    echo -e "  ${BOLD}Configure:${NC}"
    echo -e "  ${DIM}export APPLE_ID=\"apple@example.com\"${NC}"
    echo -e "  ${DIM}export APPLE_TEAM_ID=\"S96TK27X79\"${NC}"
    echo -e "  ${DIM}export APPLE_APP_SPECIFIC_PASSWORD=\"xxxx-xxxx-xxxx-xxxx\"${NC}"
    echo ""
    echo -e "  ${DIM}A app-specific password é gerada em:${NC}"
    echo -e "  ${DIM}https://account.apple.com/sign-in → Sign-In and Security → App-Specific Passwords${NC}"
    echo ""
    exit 1
  fi
  success "Credenciais de notarização OK (APPLE_ID, APPLE_TEAM_ID)"

  echo ""
}

# ─── Build pipeline ───────────────────────────────────────────────

build_go_server() {
  step "2/7 Build server (Go, ${GOARCH})..."
  local server_src="$ROOT_DIR/server"
  # SERVER_BINARY_NAME: em builds paralelos cada worker usa um nome isolado
  # (ex: server-arm64, server-x64) para evitar race condition no mesmo binário.
  local binary_name="${SERVER_BINARY_NAME:-server}"

  if [[ ! -f "$server_src/go.mod" ]]; then
    fail "Source do server não encontrado em $server_src/"
  fi

  if ! command -v go &>/dev/null; then
    fail "Go não está instalado. Instale em: https://go.dev/dl/"
  fi

  info "Go $(go version | awk '{print $3}')"

  mkdir -p "$ROOT_DIR/apps/desktop/bin"
  rm -f "$ROOT_DIR/apps/desktop/bin/$binary_name"

  # Compilar com -C (Go 1.21+) para suportar output path customizado
  CGO_ENABLED=0 GOOS=darwin GOARCH="$GOARCH" go build \
    -C "$server_src" \
    -trimpath -ldflags="-s -w" \
    -o "$ROOT_DIR/apps/desktop/bin/$binary_name" \
    ./cmd/server

  # ── Validar que o binário é da arch esperada ──────────────────────
  # IMPORTANTE: validar $binary_name (caminho isolado), não 'server' (caminho
  # compartilhado) — em builds paralelos outro worker pode sobrescrever 'server'
  # entre a compilação e a verificação (race condition).
  local expected_file_arch
  case "$GOARCH" in
    arm64) expected_file_arch="arm64" ;;
    amd64) expected_file_arch="x86_64" ;;
  esac

  local actual_file_arch
  actual_file_arch=$(file "$ROOT_DIR/apps/desktop/bin/$binary_name" | grep -oE '(arm64|x86_64)' | head -1 || true)

  if [[ -z "$actual_file_arch" ]]; then
    fail "Não foi possível detectar a arquitetura do server (output de 'file' inesperado)."
  fi

  if [[ "$actual_file_arch" != "$expected_file_arch" ]]; then
    fail "server compilado com arch errada! Esperado: ${expected_file_arch}, obtido: ${actual_file_arch}"
  fi

  # Em builds paralelos, copiar o binário verificado para 'server' DEPOIS da
  # verificação de arch — garante que 'server' é sempre o binário correto desta arch.
  if [[ "$binary_name" != "server" ]]; then
    cp "$ROOT_DIR/apps/desktop/bin/$binary_name" "$ROOT_DIR/apps/desktop/bin/server"
  fi

  local bin_size
  bin_size=$(du -h "$ROOT_DIR/apps/desktop/bin/$binary_name" | cut -f1)
  success "server compilado ($bin_size, ${GOARCH}, verificado: ${actual_file_arch})"
  echo ""
}

build_web_standalone() {
  step "3/7 Build web (Next.js standalone)..."
  NEXT_OUTPUT=standalone pnpm --filter @autocut/web build
  success "Next.js build concluído"
  echo ""
}

prepare_standalone_package() {
  step "4/7 Preparar pacote standalone..."

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
  step "5/7 Build desktop (esbuild)..."
  pnpm --filter @autocut/desktop build
  success "Desktop build concluído"
  echo ""
}

package_and_sign() {
  step "6/7 Package + Code Sign + Notarização (electron-builder, ${ELECTRON_ARCH})..."
  info "Isso pode levar alguns minutos (upload para Apple + verificação)..."
  echo ""

  # ── Limpar artefatos da arch anterior ─────────────────────────────
  # O electron-builder gera latest-mac.yml listando TODOS os artefatos presentes
  # no release/. Se o build anterior (ex: arm64) deixou seus artefatos lá, o
  # electron-builder no build atual (ex: x64) vai incluir entries de ambas as
  # archs no yml — causando arch mismatch no auto-update.
  # Solução: mover artefatos da arch anterior para um subdir temporário antes
  # de rodar o electron-builder, e restaurar depois.
  local release_dir="$ROOT_DIR/apps/desktop/${RELEASE_OUTPUT_DIR:-release}"
  mkdir -p "$release_dir"
  local stash_dir="$release_dir/.stash-$$"
  if [[ -d "$release_dir" ]]; then
    # Usar find em vez de brace expansion (*.{dmg,zip,...} 2>/dev/null é syntax error em bash).
    # Single-pass: find → move em um único traversal (evita TOCTOU entre count e move).
    local _found_files
    _found_files=$(find "$release_dir" -maxdepth 1 \( -name "*.dmg" -o -name "*.zip" -o -name "*.yml" -o -name "*.blockmap" \) -type f 2>/dev/null)
    if [[ -n "$_found_files" ]]; then
      mkdir -p "$stash_dir"
      local _moved=0
      while IFS= read -r f; do
        mv "$f" "$stash_dir/"
        _moved=$((_moved + 1))
      done <<< "$_found_files"
      info "Artefatos anteriores ($_moved) movidos para .stash/ (evitar contaminação entre archs)"
    fi
  fi

  # electron-builder lê automaticamente:
  #   CSC_NAME ou CSC_LINK + CSC_KEY_PASSWORD → code signing
  #   APPLE_ID + APPLE_APP_SPECIFIC_PASSWORD + APPLE_TEAM_ID → notarização
  # electron-builder.yml tem: hardenedRuntime: true, notarize: true
  #
  # IMPORTANTE: NÃO usar `pnpm dist -- --mac --x64` aqui!
  # O pnpm passa o `--` literalmente para o script, resultando em:
  #   electron-builder --config electron-builder.yml -- --mac --x64
  # O `--` faz o electron-builder ignorar --mac e --x64 como flags, causando
  # build com a arch nativa (arm64) em vez da arch solicitada.
  # Solução: chamar electron-builder diretamente com todas as flags.
  #
  # Se GH_TOKEN estiver definido, adiciona --publish always para upload automático
  # no GitHub Releases. Em builds locais sem GH_TOKEN, apenas empacota e assina.
  local _publish_flag=""
  if [[ -n "${GH_TOKEN:-}" ]]; then
    _publish_flag="--publish always"
    info "GH_TOKEN detectado — publicando artefatos no GitHub Releases"
  fi

  # Se o electron-builder falhar, garantir que o stash é restaurado antes de
  # propagar o erro — caso contrário artefatos da arch anterior são perdidos.
  # Configuração isolada para builds paralelos (output dir e binary path customizados)
  local _eb_config="electron-builder.yml"
  local _output_dir="${RELEASE_OUTPUT_DIR:-release}"
  local _binary_name="${SERVER_BINARY_NAME:-server}"
  if [[ "$_output_dir" != "release" ]]; then
    # mktemp sem sufixo — BSD mktemp do macOS não suporta XXXXXX.yml
    local _tmp_config
    _tmp_config=$(mktemp /tmp/eb-mac-XXXXXX)
    # Substituir output dir E o caminho do server binary:
    # - "from: bin/server" → "from: bin/server-arm64" (ou server-x64)
    # Isso evita race condition onde arm64 sobrescreve bin/server enquanto x64 está
    # empacotando — cada worker aponta para seu próprio binário isolado.
    sed -e "s|output: release\$|output: ${_output_dir}|" \
        -e "s|from: bin/server\$|from: bin/${_binary_name}|" \
        "$ROOT_DIR/apps/desktop/electron-builder.yml" > "$_tmp_config"
    _eb_config="$_tmp_config"
    info "Config isolado: output=${_output_dir}, server=bin/${_binary_name}"
  fi

  local dist_exit=0
  pnpm --filter @autocut/desktop exec electron-builder \
    --config "$_eb_config" --mac --${ELECTRON_ARCH} ${_publish_flag} || dist_exit=$?

  if [[ "$dist_exit" -ne 0 ]]; then
    warn "electron-builder falhou (exit $dist_exit)"
    # Restaurar stash antes de abortar
    if [[ -d "$stash_dir" ]]; then
      for f in "$stash_dir"/*; do
        [[ -f "$f" ]] && mv "$f" "$release_dir/"
      done
      rmdir "$stash_dir" 2>/dev/null || true
      info "Artefatos anteriores restaurados após falha"
    fi
    fail "electron-builder falhou — verifique os logs acima"
  fi

  # Renomear o auto-update YAML para incluir a arch — permite builds sequenciais
  # (arm64 depois x64) sem que o segundo sobrescreva o yml do primeiro.
  #
  # NOTA: O electron-builder gera o nome do YAML baseado no canal da versão:
  #   - versão stable (1.2.3)       → latest-mac.yml
  #   - versão prerelease (1.2.3-alpha.1) → alpha-mac.yml
  #   - versão prerelease (1.2.3-beta.1)  → beta-mac.yml
  # Normalizamos SEMPRE para latest-mac-{ARCH}.yml, independente do canal.
  local _yml_src=""
  if [[ -f "$release_dir/latest-mac.yml" ]]; then
    _yml_src="$release_dir/latest-mac.yml"
  else
    # Buscar qualquer *-mac.yml que não seja latest-mac-{arch}.yml (de build anterior)
    _yml_src=$(find "$release_dir" -maxdepth 1 -name "*-mac.yml" ! -name "latest-mac-*.yml" -type f 2>/dev/null | head -1)
  fi
  if [[ -n "$_yml_src" ]]; then
    mv "$_yml_src" "$release_dir/latest-mac-${ARCH}.yml"
    success "$(basename "$_yml_src") → latest-mac-${ARCH}.yml"
  else
    warn "Nenhum auto-update YAML (*-mac.yml) encontrado em $release_dir/ — auto-update pode não funcionar"
  fi

  # Restaurar artefatos anteriores do stash
  if [[ -d "$stash_dir" ]]; then
    for f in "$stash_dir"/*; do
      [[ -f "$f" ]] && mv "$f" "$release_dir/"
    done
    rmdir "$stash_dir" 2>/dev/null || true
    info "Artefatos anteriores restaurados"
  fi

  echo ""
  success "Package + signing + notarização concluídos"
  echo ""
}

# ─── Verificação pós-build ────────────────────────────────────────

verify_output() {
  step "7/7 Verificando artefatos..."

  local release_dir="apps/desktop/${RELEASE_OUTPUT_DIR:-release}"
  local dmg_file=""

  # Filtrar DMG pela arch correta — após stash restore, múltiplos DMGs coexistem.
  # electron-builder naming: arm64 → *-arm64.dmg, x64 → *.dmg (sem arm64)
  if [[ "$ARCH" == "arm64" ]]; then
    dmg_file=$(find "$release_dir" -maxdepth 1 -name "*-arm64.dmg" 2>/dev/null | head -1)
  else
    dmg_file=$(find "$release_dir" -maxdepth 1 -name "*.dmg" ! -name "*-arm64.dmg" 2>/dev/null | head -1)
  fi

  if [[ -z "$dmg_file" ]]; then
    fail "Nenhum .dmg encontrado em $release_dir/"
  fi

  local dmg_size
  dmg_size=$(du -h "$dmg_file" | cut -f1)
  success "DMG gerado: $(basename "$dmg_file") ($dmg_size)"

  # Montar DMG e verificar o .app
  local mount_point
  mount_point=$(mktemp -d)

  if hdiutil attach "$dmg_file" -mountpoint "$mount_point" -nobrowse -quiet 2>/dev/null; then
    local app_path
    app_path=$(find "$mount_point" -name "*.app" -maxdepth 1 | head -1)

    if [[ -n "$app_path" ]]; then
      # ── Verificar arch do server dentro do .app ────────────────
      local server_in_app="$app_path/Contents/Resources/server"
      if [[ -f "$server_in_app" ]]; then
        local expected_file_arch
        case "$GOARCH" in
          arm64) expected_file_arch="arm64" ;;
          amd64) expected_file_arch="x86_64" ;;
        esac

        local packaged_arch
        packaged_arch=$(file "$server_in_app" | grep -oE '(arm64|x86_64)' | head -1 || true)

        if [[ -z "$packaged_arch" ]]; then
          hdiutil detach "$mount_point" -quiet 2>/dev/null || true
          rmdir "$mount_point" 2>/dev/null || true
          fail "Não foi possível detectar a arquitetura do server dentro do .app (output de 'file' inesperado)."
        fi

        if [[ "$packaged_arch" != "$expected_file_arch" ]]; then
          hdiutil detach "$mount_point" -quiet 2>/dev/null || true
          rmdir "$mount_point" 2>/dev/null || true
          fail "server DENTRO do .app tem arch errada! Esperado: ${expected_file_arch}, empacotado: ${packaged_arch}"
        fi
        success "server no .app: arch ${packaged_arch} ✔"
      else
        warn "server não encontrado em ${server_in_app}"
      fi

      # Code signing (output vai para stderr)
      if codesign --verify --deep --strict --verbose=2 "$app_path" 2>&1 | grep -q "valid on disk"; then
        success "Code signing verificado (valid on disk)"
      else
        warn "Code signing pode ter issues — verifique manualmente"
      fi

      # Notarização (staple)
      if xcrun stapler validate "$app_path" 2>&1 | grep -q "The validate action worked!"; then
        success "Notarização verificada (staple OK)"
      else
        warn "Staple não encontrado — verifique com: xcrun stapler validate \"$app_path\""
      fi

      # Gatekeeper (output vai para stderr)
      if spctl --assess --type execute --verbose=2 "$app_path" 2>&1 | grep -qi "accepted\|source=Notarized"; then
        success "Gatekeeper: aceito para distribuição"
      else
        warn "Gatekeeper assessment falhou — pode ser esperado em máquinas de dev"
      fi
    fi

    hdiutil detach "$mount_point" -quiet 2>/dev/null || true
  fi
  rmdir "$mount_point" 2>/dev/null || true

  echo ""
  info "Artefatos em $release_dir/:"
  ls -lh "$release_dir"/*.dmg 2>/dev/null | while read -r line; do
    info "  $line"
  done
}

# ─── Main ─────────────────────────────────────────────────────────

main() {
  cd "$ROOT_DIR"

  header
  map_env_vars
  validate_prerequisites

  pnpm install --frozen-lockfile 2>/dev/null || pnpm install

  # SHARED_BUILD_ONLY: usado em modo paralelo para compilar apenas as partes
  # compartilhadas (Next.js + standalone-pkg + desktop) uma única vez antes
  # de lançar os workers paralelos.
  if [[ "${SHARED_BUILD_ONLY:-}" == "true" ]]; then
    build_web_standalone
    prepare_standalone_package
    build_desktop
    return 0
  fi

  build_go_server

  # SKIP_SHARED_BUILD: usado pelos workers paralelos — as partes compartilhadas
  # já foram compiladas pelo SHARED_BUILD_ONLY antes de lançar os workers.
  if [[ "${SKIP_SHARED_BUILD:-}" != "true" ]]; then
    build_web_standalone
    prepare_standalone_package
    build_desktop
  fi

  package_and_sign
  verify_output

  echo ""
  echo -e "${GREEN}╔══════════════════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║${NC}  ${BOLD}Release macOS concluído com sucesso!${NC}                    ${GREEN}║${NC}"
  echo -e "${GREEN}╚══════════════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "  Artefatos em: ${BOLD}apps/desktop/release/${NC}"
  echo ""
}

main "$@"
