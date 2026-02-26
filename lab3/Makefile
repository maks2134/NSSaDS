# Makefile for cross-platform building of Multiplexed UDP Server (Lab 3)

# Variables
BINARY_NAME_SERVER=server
BINARY_NAME_CLIENT=client
BUILD_DIR=bin
PKG_NAME=NSSaDS-lab3

# Version info
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Go build flags
BUILD_FLAGS=-v $(LDFLAGS)

# Platforms
PLATFORMS=linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64

# Help target
.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Clean build directory
.PHONY: clean
clean: ## Clean build directory
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)

# Install dependencies
.PHONY: deps
deps: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Build for current platform
.PHONY: build
build: clean deps ## Build for current platform
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	@go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER) ./cmd/server
	@echo "Build completed for current platform"

# Build for all platforms
.PHONY: build-all
build-all: clean deps ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		echo "Building for $$platform..."; \
		GOOS=$$os GOARCH=$$arch go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-$$os-$$arch$$(if [ "$$os" = "windows" ]; then echo ".exe"; fi) ./cmd/server; \
	done
	@echo "Cross-platform build completed"

# Build for Linux
.PHONY: build-linux
build-linux: clean deps ## Build for Linux (amd64)
	@echo "Building for Linux amd64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-linux-amd64 ./cmd/server
	@echo "Linux build completed"

# Build for Windows
.PHONY: build-windows
build-windows: clean deps ## Build for Windows (amd64)
	@echo "Building for Windows amd64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-windows-amd64.exe ./cmd/server
	@echo "Windows build completed"

# Build for macOS
.PHONY: build-darwin
build-darwin: clean deps ## Build for macOS (amd64)
	@echo "Building for macOS amd64..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-darwin-amd64 ./cmd/server
	@echo "macOS build completed"

# Build for ARM64
.PHONY: build-arm64
build-arm64: clean deps ## Build for ARM64 platforms
	@echo "Building for ARM64 platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-linux-arm64 ./cmd/server
	@GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-darwin-arm64 ./cmd/server
	@echo "ARM64 build completed"

# Run tests
.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
.PHONY: lint
lint: ## Lint code
	@echo "Linting code..."
	@golangci-lint run

# Security check
.PHONY: security
security: ## Run security check
	@echo "Running security check..."
	@gosec ./...

# Create release package
.PHONY: package
package: build-all ## Create release packages
	@echo "Creating release packages..."
	@mkdir -p $(BUILD_DIR)/release
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		mkdir -p $(BUILD_DIR)/release/$$os-$$arch; \
		cp $(BUILD_DIR)/$(BINARY_NAME_SERVER)-$$os-$$arch$$(if [ "$$os" = "windows" ]; then echo ".exe"; fi) $(BUILD_DIR)/release/$$os-$$arch/; \
		cp README.md $(BUILD_DIR)/release/$$os-$$arch/ 2>/dev/null || true; \
		cd $(BUILD_DIR)/release && tar -czf $(PKG_NAME)-$$os-$$arch.tar.gz $$os-$$arch/; \
	done
	@echo "Release packages created in $(BUILD_DIR)/release/"

# Development server
.PHONY: run-server
run-server: ## Run server in development mode
	@echo "Starting multiplexed server in development mode..."
	@go run ./cmd/server

# Performance tests
.PHONY: test-performance
test-performance: ## Run multiplexer performance tests
	@echo "Running multiplexer performance tests..."
	@go run ./cmd/server -test

# Show build info
.PHONY: info
info: ## Show build information
	@echo "Build Information:"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Go Version: $(shell go version)"
	@echo "  Go OS/Arch: $(shell go env GOOS)/$(shell go env GOARCH)"

# Default target
.DEFAULT_GOAL := help
