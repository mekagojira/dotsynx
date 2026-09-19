SHELL := /bin/sh
APP_NAME := dotsynx
BIN_DIR := bin
CACHE_DIR := $(PWD)/.cache

GOPATH ?= $(CACHE_DIR)/gopath
GOCACHE ?= $(CACHE_DIR)/gocache
NPM_CACHE ?= $(CACHE_DIR)/npm
GO := $(HOME)/.local/go/bin/go
export GOROOT = $(HOME)/.local/go

# Fall back to any go on PATH if the user-space toolchain is missing
ifeq ($(wildcard $(GO)),)
GO := go
export GOROOT =
endif

.PHONY: all build build-web build-go test clean install install-service uninstall dist

all: install install-service

# 1. Build React + Vite Webview
build-web:
	@echo "==> Building React Webview..."
	@cd web && npm i && npm run build

# 2. Build Go Binary with Embedded Web Assets
build-go:
	@echo "==> Building dotsynx Go binary..."
	@mkdir -p $(BIN_DIR)
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) $(GO) build -o $(BIN_DIR)/$(APP_NAME) ./cmd/dotsynx
	@codesign -s - -f $(BIN_DIR)/$(APP_NAME) 2>/dev/null || true

# Full build: web first, then go
build: build-web build-go
	@echo "==> Binary built successfully: $(BIN_DIR)/$(APP_NAME)"

# Run Go tests
test:
	@echo "==> Running tests..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) $(GO) test -v ./internal/...

# Install binary to ~/.local/bin and ensure PATH is set
install: build
	@echo "==> Installing dotsynx binary..."
	@mkdir -p $(HOME)/.local/bin
	@rm -f $(HOME)/.local/bin/$(APP_NAME)
	@cp $(BIN_DIR)/$(APP_NAME) $(HOME)/.local/bin/$(APP_NAME)
	@codesign -s - -f $(HOME)/.local/bin/$(APP_NAME) 2>/dev/null || true
	@echo "✓ Installed binary to $(HOME)/.local/bin/$(APP_NAME)"
	@echo ""
	@echo "==> Configuring shell PATH..."
	@if [ -f $(HOME)/.bashrc ]; then \
		if ! grep -q '.local/bin' $(HOME)/.bashrc 2>/dev/null; then \
			echo '' >> $(HOME)/.bashrc; \
			echo '# Added by dotsynx installer' >> $(HOME)/.bashrc; \
			echo 'export PATH="$$HOME/.local/bin:$$PATH"' >> $(HOME)/.bashrc; \
			echo "✓ Added PATH to ~/.bashrc"; \
		else \
			echo "✓ PATH already in ~/.bashrc"; \
		fi; \
	fi
	@if [ -f $(HOME)/.zshrc ]; then \
		if ! grep -q '.local/bin' $(HOME)/.zshrc 2>/dev/null; then \
			echo '' >> $(HOME)/.zshrc; \
			echo '# Added by dotsynx installer' >> $(HOME)/.zshrc; \
			echo 'export PATH="$$HOME/.local/bin:$$PATH"' >> $(HOME)/.zshrc; \
			echo "✓ Added PATH to ~/.zshrc"; \
		else \
			echo "✓ PATH already in ~/.zshrc"; \
		fi; \
	fi
	@if [ -d $(HOME)/.config/fish ]; then \
		mkdir -p $(HOME)/.config/fish/conf.d; \
		if [ ! -f $(HOME)/.config/fish/conf.d/dotsynx-path.fish ] || ! grep -q '.local/bin' $(HOME)/.config/fish/conf.d/dotsynx-path.fish 2>/dev/null; then \
			echo '# Added by dotsynx installer' > $(HOME)/.config/fish/conf.d/dotsynx-path.fish; \
			echo 'fish_add_path $$HOME/.local/bin' >> $(HOME)/.config/fish/conf.d/dotsynx-path.fish; \
			echo "✓ Added PATH to ~/.config/fish/conf.d/dotsynx-path.fish"; \
		else \
			echo "✓ PATH already in fish config"; \
		fi; \
	fi
	@echo ""
	@echo "⚠️  Restart your shell or run: export PATH=\"\$$HOME/.local/bin:\$$PATH\""

# Install systemd user service (Linux) or launchd user agent (macOS) in stopped state
install-service:
	@echo "==> Installing dotsynx service..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		mkdir -p $(HOME)/Library/LaunchAgents; \
		echo '<?xml version="1.0" encoding="UTF-8"?>' > $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '<plist version="1.0">' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '<dict>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <key>Label</key>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <string>com.dotsynx.agent</string>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <key>ProgramArguments</key>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <array>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '    <string>$(HOME)/.local/bin/$(APP_NAME)</string>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '    <string>start</string>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '    <string>--foreground</string>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  </array>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <key>RunAtLoad</key>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <false/>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <key>KeepAlive</key>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <false/>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <key>StandardOutPath</key>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <string>$(HOME)/.local/share/dotsynx/daemon.log</string>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <key>StandardErrorPath</key>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '  <string>$(HOME)/.local/share/dotsynx/daemon.log</string>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '</dict>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo '</plist>' >> $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo "✓ Installed launchd agent to ~/Library/LaunchAgents/com.dotsynx.agent.plist"; \
		echo "  Start with: launchctl load ~/Library/LaunchAgents/com.dotsynx.agent.plist"; \
		echo "  Stop with:  launchctl unload ~/Library/LaunchAgents/com.dotsynx.agent.plist"; \
	elif command -v systemctl >/dev/null 2>&1; then \
		mkdir -p $(HOME)/.config/systemd/user; \
		echo '[Unit]' > $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo 'Description=Dotsynx dotfile synchronization daemon' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo 'After=network.target' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo '' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo '[Service]' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo 'Type=simple' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo 'ExecStart=$(HOME)/.local/bin/$(APP_NAME) start --foreground' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo 'Restart=on-failure' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo 'RestartSec=10' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo '' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo '[Install]' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		echo 'WantedBy=default.target' >> $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		systemctl --user daemon-reload; \
		echo "✓ Installed systemd user service to ~/.config/systemd/user/$(APP_NAME).service"; \
		echo "  Start with: systemctl --user start $(APP_NAME)"; \
		echo "  Stop with:  systemctl --user stop $(APP_NAME)"; \
		echo "  Enable at login: systemctl --user enable $(APP_NAME)"; \
	else \
		echo "⚠️  No systemd or launchd found. Service not installed."; \
		echo "   You can run manually: $(APP_NAME) start"; \
	fi

# Uninstall binary and service
uninstall:
	@echo "==> Uninstalling dotsynx..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		launchctl unload $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist 2>/dev/null || true; \
		rm -f $(HOME)/Library/LaunchAgents/com.dotsynx.agent.plist; \
		echo "✓ Removed launchd agent"; \
	elif command -v systemctl >/dev/null 2>&1; then \
		systemctl --user stop $(APP_NAME) 2>/dev/null || true; \
		systemctl --user disable $(APP_NAME) 2>/dev/null || true; \
		rm -f $(HOME)/.config/systemd/user/$(APP_NAME).service; \
		systemctl --user daemon-reload 2>/dev/null || true; \
		echo "✓ Removed systemd service"; \
	fi
	@rm -f $(HOME)/.local/bin/$(APP_NAME)
	@echo "✓ Removed binary from ~/.local/bin"
	@echo ""
	@echo "Config and data remain in ~/.local/share/dotsynx"
	@echo "To remove completely: rm -rf ~/.local/share/dotsynx ~/.config/dotsynx"

# Package universal macOS and Linux release binaries
dist: build-web
	@echo "==> Building cross-platform release binaries..."
	@rm -rf $(BIN_DIR)/dist $(BIN_DIR)/archives
	@mkdir -p $(BIN_DIR)/dist/darwin-arm64 $(BIN_DIR)/dist/darwin-amd64 $(BIN_DIR)/dist/darwin-universal $(BIN_DIR)/dist/linux-amd64 $(BIN_DIR)/dist/linux-arm64 $(BIN_DIR)/archives
	@echo " -> Compiling macOS ARM64..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="-s -w" -o $(BIN_DIR)/dist/darwin-arm64/$(APP_NAME) ./cmd/dotsynx
	@echo " -> Compiling macOS AMD64..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="-s -w" -o $(BIN_DIR)/dist/darwin-amd64/$(APP_NAME) ./cmd/dotsynx
	@echo " -> Creating macOS Universal Binary (Intel + Apple Silicon)..."
	@lipo -create -output $(BIN_DIR)/dist/darwin-universal/$(APP_NAME) $(BIN_DIR)/dist/darwin-amd64/$(APP_NAME) $(BIN_DIR)/dist/darwin-arm64/$(APP_NAME)
	@codesign -s - -f $(BIN_DIR)/dist/darwin-universal/$(APP_NAME) 2>/dev/null || true
	@tar -czf $(BIN_DIR)/archives/$(APP_NAME)-darwin-universal.tar.gz -C $(BIN_DIR)/dist/darwin-universal $(APP_NAME)
	@echo " -> Compiling Linux AMD64 (x86_64)..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="-s -w" -o $(BIN_DIR)/dist/linux-amd64/$(APP_NAME) ./cmd/dotsynx
	@tar -czf $(BIN_DIR)/archives/$(APP_NAME)-linux-amd64.tar.gz -C $(BIN_DIR)/dist/linux-amd64 $(APP_NAME)
	@echo " -> Compiling Linux ARM64 (aarch64)..."
	@GOPATH=$(GOPATH) GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -ldflags="-s -w" -o $(BIN_DIR)/dist/linux-arm64/$(APP_NAME) ./cmd/dotsynx
	@tar -czf $(BIN_DIR)/archives/$(APP_NAME)-linux-arm64.tar.gz -C $(BIN_DIR)/dist/linux-arm64 $(APP_NAME)
	@cd $(BIN_DIR)/archives && shasum -a 256 *.tar.gz > checksums.txt
	@echo "==> Release packages created in $(BIN_DIR)/archives/:"
	@ls -lh $(BIN_DIR)/archives/

clean:
	@rm -rf $(BIN_DIR) web/dist
