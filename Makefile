SHELL := /bin/sh
APP_NAME := dotsynx
BIN_DIR := bin
CACHE_DIR := $(PWD)/.cache

GOPATH ?= $(CACHE_DIR)/gopath
GOCACHE ?= $(CACHE_DIR)/gocache
NPM_CACHE ?= $(CACHE_DIR)/npm

.PHONY: all build build-web build-go test clean install

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

clean:
	@rm -rf $(BIN_DIR) web/dist
