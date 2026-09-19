SHELL := /bin/sh
APP_NAME := dotsynx
BIN_DIR := bin
CACHE_DIR := $(PWD)/.cache

GOPATH ?= $(CACHE_DIR)/gopath
GOCACHE ?= $(CACHE_DIR)/gocache
NPM_CACHE ?= $(CACHE_DIR)/npm

.PHONY: all build build-web build-go test clean install dist

all: build

# 1. Build React + Vite Webview
build-web:
	@echo "==> Building React Webview..."
	@cd web && npm run build

# 2. Build Go Binary with Embedded Web Assets
build-go:
	@echo "==> Building dotsynx Go binary..."
	@mkdir -p $(BIN_DIR)
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/dotsynx
	@codesign -s - -f $(BIN_DIR)/$(APP_NAME) 2>/dev/null || true

# Full build: web first, then go
build: build-web build-go
	@echo "==> Binary built successfully: $(BIN_DIR)/$(APP_NAME)"

# Run Go tests
test:
	@echo "==> Running tests..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) go test -v ./internal/...

# Install binary to ~/.local/bin
install: build
	@mkdir -p $(HOME)/.local/bin
	@rm -f $(HOME)/.local/bin/$(APP_NAME)
	@cp $(BIN_DIR)/$(APP_NAME) $(HOME)/.local/bin/$(APP_NAME)
	@codesign -s - -f $(HOME)/.local/bin/$(APP_NAME) 2>/dev/null || true
	@echo "==> Installed dotsynx to $(HOME)/.local/bin/$(APP_NAME)"

# Package universal macOS and Linux release binaries
dist: build-web
	@echo "==> Building cross-platform release binaries..."
	@rm -rf $(BIN_DIR)/dist $(BIN_DIR)/archives
	@mkdir -p $(BIN_DIR)/dist/darwin-arm64 $(BIN_DIR)/dist/darwin-amd64 $(BIN_DIR)/dist/darwin-universal $(BIN_DIR)/dist/linux-amd64 $(BIN_DIR)/dist/linux-arm64 $(BIN_DIR)/archives
	@echo " -> Compiling macOS ARM64..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/dist/darwin-arm64/$(APP_NAME) ./cmd/dotsynx
	@echo " -> Compiling macOS AMD64..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/dist/darwin-amd64/$(APP_NAME) ./cmd/dotsynx
	@echo " -> Creating macOS Universal Binary (Intel + Apple Silicon)..."
	@lipo -create -output $(BIN_DIR)/dist/darwin-universal/$(APP_NAME) $(BIN_DIR)/dist/darwin-amd64/$(APP_NAME) $(BIN_DIR)/dist/darwin-arm64/$(APP_NAME)
	@codesign -s - -f $(BIN_DIR)/dist/darwin-universal/$(APP_NAME) 2>/dev/null || true
	@tar -czf $(BIN_DIR)/archives/$(APP_NAME)-darwin-universal.tar.gz -C $(BIN_DIR)/dist/darwin-universal $(APP_NAME)
	@echo " -> Compiling Linux AMD64 (x86_64)..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/dist/linux-amd64/$(APP_NAME) ./cmd/dotsynx
	@tar -czf $(BIN_DIR)/archives/$(APP_NAME)-linux-amd64.tar.gz -C $(BIN_DIR)/dist/linux-amd64 $(APP_NAME)
	@echo " -> Compiling Linux ARM64 (aarch64)..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BIN_DIR)/dist/linux-arm64/$(APP_NAME) ./cmd/dotsynx
	@tar -czf $(BIN_DIR)/archives/$(APP_NAME)-linux-arm64.tar.gz -C $(BIN_DIR)/dist/linux-arm64 $(APP_NAME)
	@cd $(BIN_DIR)/archives && shasum -a 256 *.tar.gz > checksums.txt
	@echo "==> Release packages created in $(BIN_DIR)/archives/:"
	@ls -lh $(BIN_DIR)/archives/

clean:
	@rm -rf $(BIN_DIR) web/dist
