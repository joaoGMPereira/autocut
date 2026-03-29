# ─── AutoCut — Root Makefile ─────────────────────────────────────────────────
# Usage:
#   make dev              — Electron + Go (debug) + Next.js (turbopack)
#   make run              — Go + Next.js only (browser, sem Electron)
#   make wolf-build       — Compila Go server (release)
#   make wolf-build-debug — Compila Go server (debug + race detector)
#   make release          — Build completo (macOS arm64)
#   make bump-version v=patch|minor|major
#   make publish          — Build + upload GitHub Releases
#   make kill             — Mata processos nas portas dev

SHELL := /bin/bash
.DEFAULT_GOAL := dev

# ─── Versão ───────────────────────────────────────────────────────────────────
VERSION_FILE := apps/desktop/package.json
VERSION      := $(shell node -p "require('./$(VERSION_FILE)').version")

# ─── Portas ───────────────────────────────────────────────────────────────────
GO_DEV_PORT  := 4071
WEB_DEV_PORT := 3201

# ─── Desenvolvimento ──────────────────────────────────────────────────────────

.PHONY: dev
dev: wolf-build-debug
	@echo "▶ Starting AutoCut dev environment..."
	@trap 'make kill' INT TERM; \
	GO_DIR=$$HOME/.autocut-dev \
	pnpm --filter @autocut/desktop run dev & \
	wait

.PHONY: run
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

.PHONY: wolf-build
wolf-build:
	@echo "▶ Building Go server (release)..."
	@$(MAKE) -C server build

.PHONY: wolf-build-debug
wolf-build-debug:
	@echo "▶ Building Go server (debug)..."
	@cd server && go build -race -o ../apps/desktop/bin/server ./cmd/server

# ─── Release ──────────────────────────────────────────────────────────────────

.PHONY: release
release: wolf-build
	@echo "▶ Building Next.js standalone..."
	@NEXT_OUTPUT=standalone pnpm --filter @autocut/web build
	@echo "▶ Preparing standalone package..."
	@$(MAKE) _prepare-standalone
	@echo "▶ Bundling Electron..."
	@pnpm --filter @autocut/desktop run build
	@echo "▶ Building installer (macOS arm64)..."
	@pnpm --filter @autocut/desktop run dist:mac
	@echo "✓ Release built at apps/desktop/release/"

.PHONY: _prepare-standalone
_prepare-standalone:
	@rm -rf apps/web/standalone-pkg
	@rsync -aL apps/web/.next/standalone/ apps/web/standalone-pkg/
	@rsync -aL apps/web/.next/static/ apps/web/standalone-pkg/apps/web/.next/static/
	@rsync -aL apps/web/public/ apps/web/standalone-pkg/apps/web/public/ 2>/dev/null || true

# ─── Versioning ───────────────────────────────────────────────────────────────

.PHONY: bump-version
bump-version:
ifndef v
	$(error Usage: make bump-version v=patch|minor|major)
endif
	@echo "▶ Bumping version ($(v))..."
	@node scripts/bump-version.mjs $(v)
	@echo "✓ Version bumped to $$(node -p \"require('./$(VERSION_FILE)').version\")"

# ─── Publish ──────────────────────────────────────────────────────────────────

.PHONY: publish
publish: release
ifndef GH_TOKEN
	$(error GH_TOKEN env var required for publish)
endif
	@echo "▶ Publishing to GitHub Releases..."
	@GH_TOKEN=$(GH_TOKEN) pnpm --filter @autocut/desktop exec electron-builder \
		--mac \
		--publish always \
		--config electron-builder.yml

# ─── Utilities ────────────────────────────────────────────────────────────────

.PHONY: kill
kill:
	@echo "▶ Killing dev processes..."
	@node scripts/kill-ports.mjs $(GO_DEV_PORT) $(WEB_DEV_PORT) 2>/dev/null || true
	@pkill -f "apps/desktop/bin/server" 2>/dev/null || true
	@echo "✓ Done"

.PHONY: clean
clean:
	@echo "▶ Cleaning build artifacts..."
	@rm -rf apps/desktop/bin/server apps/desktop/bin/server.exe
	@rm -rf apps/desktop/compiled apps/desktop/release
	@rm -rf apps/web/.next apps/web/standalone-pkg
	@echo "✓ Clean"

.PHONY: install
install:
	@pnpm install

.PHONY: help
help:
	@echo "AutoCut Makefile targets:"
	@echo "  dev              — Electron + Go (debug) + Next.js"
	@echo "  run              — Go + Next.js only (browser)"
	@echo "  wolf-build       — Build Go server (release)"
	@echo "  wolf-build-debug — Build Go server (debug)"
	@echo "  release          — Full build (macOS)"
	@echo "  bump-version     — v=patch|minor|major"
	@echo "  publish          — Build + upload GitHub Releases"
	@echo "  kill             — Kill dev processes"
	@echo "  clean            — Remove build artifacts"
