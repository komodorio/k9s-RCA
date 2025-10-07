.PHONY: build install clean test help

BINARY_NAME=k9s-rca
INSTALL_DIR=$(HOME)/.local/bin
K9S_CONFIG_DIR=$(or $(XDG_CONFIG_HOME),$(HOME)/.config)/k9s

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the k9s-rca binary
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .
	@echo "Build complete!"

install: build ## Build and install the binary
	@echo "Installing $(BINARY_NAME) to ~/.local/bin..."
	mkdir -p ~/.local/bin
	cp $(BINARY_NAME) ~/.local/bin/
	@echo "Installation complete!"

install-plugin: install ## Install binary and plugin configuration
	@echo "Installing k9s plugin configuration..."
	@if [ -z "$(XDG_CONFIG_HOME)" ]; then \
		echo "⚠️  XDG_CONFIG_HOME is not set. K9s will use ~/.config"; \
		echo "   To set it permanently, add to your shell profile:"; \
		echo "   export XDG_CONFIG_HOME=\"\$$HOME/.config\""; \
		echo ""; \
	fi
	@echo "Installing plugin configuration to: $(K9S_CONFIG_DIR)/plugins.yaml"
	mkdir -p $(K9S_CONFIG_DIR)
	cp k9s_rca_plugin.yaml $(K9S_CONFIG_DIR)/plugins.yaml
	@echo ""
	@echo "✅ Installation complete!"
	@echo ""
	@echo "📋 Required setup for the plugin to work:"
	@echo "   1. Set your Komodor API key:"
	@echo "      export KOMODOR_API_KEY=\"your-api-key\""
	@echo ""
	@echo "   2. If XDG_CONFIG_HOME is not set, add to ~/.bashrc or ~/.zshrc:"
	@echo "      export XDG_CONFIG_HOME=\"\$$HOME/.config\""
	@echo ""
	@echo "   3. Restart k9s (if running): pkill k9s && k9s"
	@echo ""
	@echo "   4. In k9s, press Shift-K on any resource to trigger RCA"
	@echo ""
	@echo "⚠️  Without these steps, the plugin will NOT work!"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf dist/
	@echo "Clean complete!"

test: ## Run tests
	@echo "Running tests..."
	go test ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

dev: deps build ## Development build with dependencies

release: ## Create a release using GoReleaser
	@echo "Creating release with GoReleaser..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "GoReleaser not found. Install with: brew install goreleaser"; \
		exit 1; \
	fi
	goreleaser release --clean

release-snapshot: ## Create a snapshot release for testing
	@echo "Creating snapshot release..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "GoReleaser not found. Install with: brew install goreleaser"; \
		exit 1; \
	fi
	goreleaser release --snapshot --clean

uninstall: ## Remove installed binary and plugin
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Binary removed from $(INSTALL_DIR)"
	@echo "Note: Plugin configuration at $(K9S_CONFIG_DIR)/plugins.yaml must be removed manually" 