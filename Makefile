# ─── AutoCut — Root Makefile ─────────────────────────────────────────────────
SHELL := /bin/bash
.DEFAULT_GOAL := help

# ─── Versão ───────────────────────────────────────────────────────────────────
VERSION_FILE := apps/desktop/package.json
VERSION      := $(shell node -p "require('./$(VERSION_FILE)').version")

# ─── Portas ───────────────────────────────────────────────────────────────────
GO_DEV_PORT  := 4071
WEB_DEV_PORT := 3201

# ─── Arquitetura alvo ─────────────────────────────────────────────────────────
# Override: make release-macos-arch ARCH=x64
ARCH ?= arm64

ifeq ($(ARCH),x64)
  GOARCH        := amd64
  ELECTRON_ARCH := x64
else ifeq ($(ARCH),arm64)
  GOARCH        := arm64
  ELECTRON_ARCH := arm64
else
  $(error ARCH inválido: '$(ARCH)'. Use 'arm64' ou 'x64'.)
endif

.PHONY: help dev run wolf-build wolf-build-debug wolf-build-windows \
        release release-unsigned release-macos release-macos-arch release-windows \
        bump-version publish kill clean install

## Exibe este helper com todos os targets disponíveis
help:
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════╗"
	@echo "║             AutoCut — Makefile Help                      ║"
	@echo "╚══════════════════════════════════════════════════════════╝"
	@echo ""
	@awk '/^## /{ if(!desc) desc=substr($$0,4); next } /^[a-zA-Z_][a-zA-Z0-9_-]*:/{ gsub(/:.*/, "", $$1); if(desc) printf "  \033[36m%-22s\033[0m %s\n", $$1, desc; desc="" }' $(MAKEFILE_LIST)
	@echo ""
	@echo "  Variáveis de ambiente (override com VAR=valor):"
	@echo "    ARCH   arquitetura alvo: arm64 (default) ou x64"
	@echo ""

# ─── Desenvolvimento ──────────────────────────────────────────────────────────

## Dev com Electron — build do desktop + abre o app (que gerencia web + server)
dev: wolf-build-debug
	@echo "▶ Starting AutoCut dev environment..."
	@trap 'make kill' INT TERM; \
	GO_DIR=$$HOME/.autocut-dev \
	pnpm --filter @autocut/desktop run dev & \
	wait

## Dev sem Electron — Go + Next.js no browser
run: wolf-build-debug
	@echo "▶ Starting Go + Next.js (no Electron)..."
	@trap 'make kill' INT TERM; \
	apps/desktop/bin/server \
		-host 127.0.0.1 \
		-port $(GO_DEV_PORT) \
		-dir $$HOME/.autocut-dev & \
	pnpm --filter @autocut/web dev & \
	wait

# ─── Go Build ─────────────────────────────────────────────────────────────────

## Compilar server (release, CGO_ENABLED=0)
## Uso: make wolf-build           (arch nativa)
##      make wolf-build ARCH=x64  (Intel)
wolf-build:
	@echo "▶ Building server ($(GOARCH), release)..."
	@$(MAKE) -C server build ARCH=$(GOARCH) GOOS=darwin
	@echo "✔ server ($(GOARCH)) → apps/desktop/bin/server"
	@file apps/desktop/bin/server

## Compilar server (debug + race detector)
wolf-build-debug:
	@echo "▶ Building server (debug)..."
	@cd server && go build -race -o ../apps/desktop/bin/server ./cmd/server

## Compilar server para Windows (x64)
wolf-build-windows:
	@echo "▶ Building server (windows/amd64, release)..."
	@$(MAKE) -C server build ARCH=amd64 GOOS=windows
	@echo "✔ server.exe (windows/amd64) → apps/desktop/bin/server.exe"

# ─── Release ──────────────────────────────────────────────────────────────────

## Release local sem assinatura — build completo (standalone + electron-builder) sem notarização
## Gera .app em apps/desktop/release/ para testes locais.
release-unsigned: install wolf-build
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════╗"
	@echo "║     AutoCut — Release Unsigned (local test)              ║"
	@echo "╚══════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "→ 1/5 Build web (Next.js standalone)..."
	NEXT_OUTPUT=standalone pnpm --filter @autocut/web build
	@echo ""
	@echo "→ 2/5 Preparar pacote standalone..."
	rm -rf apps/web/standalone-pkg
	rsync -aL apps/web/.next/standalone/ apps/web/standalone-pkg/
	@for dep in styled-jsx postcss @next @swc caniuse-lite; do \
		cp -R apps/web/standalone-pkg/node_modules/.pnpm/next@*/node_modules/$$dep \
			apps/web/standalone-pkg/apps/web/node_modules/ 2>/dev/null || true; \
	done
	rm -rf apps/web/standalone-pkg/node_modules
	mkdir -p apps/web/standalone-pkg/apps/web/.next/static
	cp -R apps/web/.next/static/* apps/web/standalone-pkg/apps/web/.next/static/
	@if [ -d "apps/web/public" ]; then \
		mkdir -p apps/web/standalone-pkg/apps/web/public; \
		cp -R apps/web/public/* apps/web/standalone-pkg/apps/web/public/; \
	fi
	@echo "   Pacote standalone pronto em apps/web/standalone-pkg/"
	@echo ""
	@echo "→ 3/5 Build desktop (esbuild)..."
	pnpm --filter @autocut/desktop build
	@echo ""
	@echo "→ 4/5 Package com electron-builder ($(ELECTRON_ARCH), sem assinatura)..."
	@unset CSC_LINK CSC_NAME CSC_KEY_PASSWORD APPLE_ID APPLE_APP_SPECIFIC_PASSWORD APPLE_TEAM_ID && \
		CSC_IDENTITY_AUTO_DISCOVERY=false \
		pnpm --filter @autocut/desktop exec electron-builder --config electron-builder.yml \
			--mac --$(ELECTRON_ARCH) -c.mac.identity=null -c.mac.notarize=false
	@echo ""
	@echo "→ 5/5 Abrindo app..."
	@open apps/desktop/release/mac-$(ELECTRON_ARCH)/"AutoCut.app" 2>/dev/null || true
	@echo ""
	@echo "✔ Release unsigned pronto! App em apps/desktop/release/"

## Release build simples (sem sign/notarize) — igual release-unsigned mas sem abrir o app
release: install wolf-build
	@echo "▶ Building Next.js standalone..."
	@NEXT_OUTPUT=standalone pnpm --filter @autocut/web build
	@echo "▶ Preparing standalone package..."
	@$(MAKE) _prepare-standalone
	@echo "▶ Bundling Electron..."
	@pnpm --filter @autocut/desktop run build
	@echo "▶ Building installer (macOS $(ELECTRON_ARCH))..."
	@pnpm --filter @autocut/desktop run dist:mac
	@echo "✓ Release built at apps/desktop/release/"

.PHONY: _prepare-standalone
_prepare-standalone:
	@rm -rf apps/web/standalone-pkg
	@rsync -aL apps/web/.next/standalone/ apps/web/standalone-pkg/
	@rsync -aL apps/web/.next/static/ apps/web/standalone-pkg/apps/web/.next/static/
	@rsync -aL apps/web/public/ apps/web/standalone-pkg/apps/web/public/ 2>/dev/null || true

## Release macOS — DMG assinado e notarizado (arm64 + x64)
## Requer: certificado Developer ID + APPLE_ID, APPLE_TEAM_ID, APPLE_APP_SPECIFIC_PASSWORD
## Detalhes: scripts/release-macos.sh
release-macos:
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════╗"
	@echo "║  AutoCut — Release macOS — arm64 + x64                   ║"
	@echo "╚══════════════════════════════════════════════════════════╝"
	@echo ""
	@ARCH=arm64 ./scripts/release-macos.sh
	@ARCH=x64 ./scripts/release-macos.sh
	@echo ""
	@echo "✔ Build de ambas as arquiteturas concluído!"
	@echo "  Artefatos em apps/desktop/release/"
	@ls -1 apps/desktop/release/*.{dmg,zip,yml} 2>/dev/null | while read f; do echo "    $$(basename $$f)"; done
	@echo ""

## Release macOS — apenas uma arch específica
## Uso: make release-macos-arch ARCH=arm64
##      make release-macos-arch ARCH=x64
release-macos-arch:
	@ARCH=$(ARCH) ./scripts/release-macos.sh

## Release Windows — NSIS installer (cross-compile de macOS, sem code signing)
## Uso: make release-windows             (x64, default)
##      WIN_ARCH=arm64 make release-windows  (arm64)
release-windows:
	@./scripts/release-windows.sh

## Pipeline local completa — bump + GitHub Release + build arm64 + x64 + windows
## Lê credenciais de .env.local. Usa: GH_TOKEN, DEVELOPER_ID_APPLICATION_CERTIFICATE, etc.
## Uso: make release-local
release-local:
	@./scripts/release-local.sh

## Pipeline local paralela — shared build uma vez, builds em paralelo (16GB+ RAM)
release-local-parallel:
	@./scripts/release-local.sh --parallel

## Pipeline local — só macOS arm64
release-local-arm64:
	@./scripts/release-local.sh --arm64

## Pipeline local — só macOS x64
release-local-x64:
	@./scripts/release-local.sh --x64

## Pipeline local — só macOS (arm64 + x64)
release-local-mac:
	@./scripts/release-local.sh --mac

## Pipeline local — só Windows x64
release-local-windows:
	@./scripts/release-local.sh --windows

## Pipeline local — build sem publicar (dry run)
release-local-dry:
	@./scripts/release-local.sh --no-publish

# ─── Versioning ───────────────────────────────────────────────────────────────

## Bump de versão — patch, minor ou major (interativo se v= não especificado)
## Uso: make bump-version        (interativo)
##      make bump-version v=patch
##      make bump-version v=minor
##      make bump-version v=major
bump-version:
	@./scripts/bump-version.sh $(v)

# ─── Publish ──────────────────────────────────────────────────────────────────

## Publicar release no GitHub Releases — build + code sign + notarize + upload
## Requer: GH_TOKEN, DEVELOPER_ID_APPLICATION_CERTIFICATE, APPLE_ID, APPLE_TEAM_ID, APPLE_APP_SPECIFIC_PASSWORD
## Uso: GH_TOKEN=<token> make publish
##      GH_TOKEN=<token> ARCH=x64 make publish
publish:
	@if [ -z "$(GH_TOKEN)" ]; then echo "❌ GH_TOKEN não definido. Use: GH_TOKEN=<token> make publish"; exit 1; fi
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════╗"
	@echo "║       AutoCut — Publish to GitHub Releases               ║"
	@echo "╚══════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "→ Delegando para release-macos.sh (build + sign + notarize + publish)..."
	@echo ""
	GH_TOKEN=$(GH_TOKEN) ARCH=$(ARCH) ./scripts/release-macos.sh

# ─── Utilities ────────────────────────────────────────────────────────────────

## Matar processos nas portas dev
kill:
	@echo "▶ Killing dev processes..."
	@node scripts/kill-ports.mjs $(GO_DEV_PORT) $(WEB_DEV_PORT) 2>/dev/null || true
	@pkill -f "apps/desktop/bin/server" 2>/dev/null || true
	@echo "✓ Done"

## Limpar artefatos de build
clean:
	@echo "▶ Cleaning build artifacts..."
	@rm -rf apps/desktop/bin/server apps/desktop/bin/server.exe
	@rm -rf apps/desktop/compiled apps/desktop/release
	@rm -rf apps/web/.next apps/web/standalone-pkg
	@echo "✓ Clean"

## Instalar dependências
install:
	@pnpm install
