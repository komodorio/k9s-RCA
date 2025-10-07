.PHONY: build install clean test help

BINARY_NAME=k9s-rca
INSTALL_DIR=$(HOME)/.local/bin
K9S_CONFIG_DIR=$(HOME)/.config/k9s

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
	mkdir -p $(K9S_CONFIG_DIR)
	cp k9s_rca_plugin.yaml $(K9S_CONFIG_DIR)/plugins.yaml
	@echo "Plugin configuration installed to $(K9S_CONFIG_DIR)/plugins.yaml"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	@echo "Clean complete!"

test: ## Run tests
	@echo "Running tests..."
	go test ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

dev: deps build ## Development build with dependencies

release: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 .
	@echo "Release builds complete!"

uninstall: ## Remove installed binary and plugin
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Binary removed from $(INSTALL_DIR)"
	@echo "Note: Plugin configuration at $(K9S_CONFIG_DIR)/plugins.yaml must be removed manually" 