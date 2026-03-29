GOOS   ?= $(shell go env GOOS)
ARCH   ?= $(shell go env GOARCH)
PKG    := ./cmd/server
OUTDIR := ../apps/desktop/bin

.PHONY: build build-debug test clean

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build \
		-trimpath \
		-ldflags="-s -w" \
		-o $(OUTDIR)/server \
		$(PKG)

build-debug:
	go build -race -o $(OUTDIR)/server $(PKG)

test:
	go test ./...

clean:
	rm -rf $(OUTDIR)/server $(OUTDIR)/server.exe
