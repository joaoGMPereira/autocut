GOOS   ?= $(shell go env GOOS)
ARCH   ?= $(shell go env GOARCH)
PKG    := ./cmd/server
OUTDIR := ../apps/desktop/bin

# OAuth seed credentials — set via env vars loaded from gh variable (see scripts/load-credentials.sh)
OAUTH_CLIENT_ID_INERD          ?=
OAUTH_CLIENT_SECRET_INERD      ?=
OAUTH_CLIENT_ID_MAROMBA        ?=
OAUTH_CLIENT_SECRET_MAROMBA    ?=
OAUTH_CLIENT_ID_REACT          ?=
OAUTH_CLIENT_SECRET_REACT      ?=
OAUTH_CLIENT_ID_GEN_LANG       ?=
OAUTH_CLIENT_SECRET_GEN_LANG   ?=

DB_PKG := github.com/joaoGMPereira/autocut/server/internal/database

OAUTH_LDFLAGS := \
	-X '$(DB_PKG).seedOAuthClientIDInerd=$(OAUTH_CLIENT_ID_INERD)' \
	-X '$(DB_PKG).seedOAuthClientSecretInerd=$(OAUTH_CLIENT_SECRET_INERD)' \
	-X '$(DB_PKG).seedOAuthClientIDMaromba=$(OAUTH_CLIENT_ID_MAROMBA)' \
	-X '$(DB_PKG).seedOAuthClientSecretMaromba=$(OAUTH_CLIENT_SECRET_MAROMBA)' \
	-X '$(DB_PKG).seedOAuthClientIDReact=$(OAUTH_CLIENT_ID_REACT)' \
	-X '$(DB_PKG).seedOAuthClientSecretReact=$(OAUTH_CLIENT_SECRET_REACT)' \
	-X '$(DB_PKG).seedOAuthClientIDGenLang=$(OAUTH_CLIENT_ID_GEN_LANG)' \
	-X '$(DB_PKG).seedOAuthClientSecretGenLang=$(OAUTH_CLIENT_SECRET_GEN_LANG)'

.PHONY: build build-debug test clean

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build \
		-trimpath \
		-ldflags="-s -w $(OAUTH_LDFLAGS)" \
		-o $(OUTDIR)/server \
		$(PKG)

build-debug:
	go build -race -o $(OUTDIR)/server $(PKG)

test:
	go test ./...

clean:
	rm -rf $(OUTDIR)/server $(OUTDIR)/server.exe
