
BINARY_NAME_SERVER=lab4-server
BINARY_NAME_CLIENT=lab4-client
OUTPUT_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -s -w"

BUILD_FLAGS=-v
GO_FILES=$(shell find . -name "*.go" -type f)

PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: all
all: clean deps lint test build

.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

.PHONY: test
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-coverage
test-coverage: test
	@echo "Coverage report generated: coverage.html"

.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

.PHONY: build
build: build-server build-client

.PHONY: build-server
build-server:
	@echo "Building server for current platform..."
	@mkdir -p $(OUTPUT_DIR)
	go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME_SERVER) ./cmd/server

.PHONY: build-client
build-client:
	@echo "Building client for current platform..."
	@mkdir -p $(OUTPUT_DIR)
	go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME_CLIENT) ./cmd/client

.PHONY: build-all
build-all: build-all-server build-all-client

.PHONY: build-all-server
build-all-server:
	@echo "Building server for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	@for platform in $(PLATFORMS); do \
		echo "Building server for $$platform..."; \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		GOOS=$$os GOARCH=$$arch \
		go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME_SERVER)-$$os-$$arch$$(if [ "$$os" = "windows" ]; then echo ".exe"; fi) ./cmd/server; \
	done

.PHONY: build-all-client
build-all-client:
	@echo "Building client for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	@for platform in $(PLATFORMS); do \
		echo "Building client for $$platform..."; \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		GOOS=$$os GOARCH=$$arch \
		go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME_CLIENT)-$$os-$$arch$$(if [ "$$os" = "windows" ]; then echo ".exe"; fi) ./cmd/client; \
	done

.PHONY: build-platform
build-platform:
	@if [ -z "$(PLATFORM)" ]; then \
		echo "Usage: make build-platform PLATFORM=linux/amd64"; \
		exit 1; \
	fi
	@echo "Building for $(PLATFORM)..."
	@mkdir -p $(OUTPUT_DIR)
	@GOOS=$(word 1,$(subst /, ,$(PLATFORM))) GOARCH=$(word 2,$(subst /, ,$(PLATFORM))) \
		go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME_SERVER)-$(word 1,$(subst /, ,$(PLATFORM)))-$(word 2,$(subst /, ,$(PLATFORM)))$(if $(filter windows,$(word 1,$(subst /, ,$(PLATFORM))),.exe,) ./cmd/server
	@GOOS=$(word 1,$(subst /, ,$(PLATFORM))) GOARCH=$(word 2,$(subst /, ,$(PLATFORM))) \
		go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME_CLIENT)-$(word 1,$(subst /, ,$(PLATFORM)))-$(word 2,$(subst /, ,$(PLATFORM)))$(if $(filter windows,$(word 1,$(subst /, ,$(PLATFORM))),.exe,) ./cmd/client

.PHONY: run-server
run-server: build-server
	@echo "Starting UDP server..."
	./$(OUTPUT_DIR)/$(BINARY_NAME_SERVER)

.PHONY: run-client
run-client: build-client
	@echo "Starting UDP client..."
	./$(OUTPUT_DIR)/$(BINARY_NAME_CLIENT)

.PHONY: dev-server
dev-server:
	@echo "Starting server in development mode..."
	go run ./cmd/server

.PHONY: dev-client
dev-client:
	@echo "Starting client in development mode..."
	go run ./cmd/client

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(OUTPUT_DIR)
	rm -f coverage.out coverage.html
	go clean -cache

.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	go install golang.org/x/tools/cmd/goimports@latest
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint already installed"; \
	else \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2; \
	fi

.PHONY: docs
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server on :6060..."; \
		godoc -http=:6060; \
	else \
		echo "godoc not installed, run: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

.PHONY: security
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed, run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

.PHONY: outdated
outdated:
	@echo "Checking for outdated dependencies..."
	@if command -v go-mod-outdated >/dev/null 2>&1; then \
		go list -u -m -json all | go-mod-outdated -update -direct; \
	else \
		echo "go-mod-outdated not installed, run: go install github.com/psampaz/go-mod-outdated@latest"; \
	fi

.PHONY: package
package: build-all
	@echo "Creating release packages..."
	@mkdir -p $(OUTPUT_DIR)/release
	@for platform in $(PLATFORMS); do \
		echo "Packaging for $$platform..."; \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		mkdir -p $(OUTPUT_DIR)/release/$(BINARY_NAME)-$$os-$$arch; \
		cp $(OUTPUT_DIR)/$(BINARY_NAME_SERVER)-$$os-$$arch$$(if [ "$$os" = "windows" ]; then echo ".exe"; fi) $(OUTPUT_DIR)/release/$(BINARY_NAME)-$$os-$$arch/; \
		cp $(OUTPUT_DIR)/$(BINARY_NAME_CLIENT)-$$os-$$arch$$(if [ "$$os" = "windows" ]; then echo ".exe"; fi) $(OUTPUT_DIR)/release/$(BINARY_NAME)-$$os-$$arch/; \
		cp README.md $(OUTPUT_DIR)/release/$(BINARY_NAME)-$$os-$$arch/ 2>/dev/null || true; \
		cd $(OUTPUT_DIR)/release && tar -czf $(BINARY_NAME)-$$os-$$arch.tar.gz $(BINARY_NAME)-$$os-$$arch/; \
	done

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all              - Clean, deps, lint, test, and build"
	@echo "  deps             - Install dependencies"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  lint             - Run linter"
	@echo "  fmt              - Format code"
	@echo "  build            - Build for current platform"
	@echo "  build-server     - Build server for current platform"
	@echo "  build-client     - Build client for current platform"
	@echo "  build-all        - Build for all platforms"
	@echo "  build-platform   - Build for specific platform (PLATFORM=linux/amd64)"
	@echo "  run-server       - Build and run server"
	@echo "  run-client       - Build and run client"
	@echo "  dev-server       - Run server in development mode"
	@echo "  dev-client       - Run client in development mode"
	@echo "  clean            - Clean build artifacts"
	@echo "  install-tools    - Install development tools"
	@echo "  docs             - Generate and serve documentation"
	@echo "  security         - Run security scan"
	@echo "  benchmark        - Run benchmarks"
	@echo "  update-deps      - Update dependencies"
	@echo "  outdated         - Check for outdated dependencies"
	@echo "  package          - Create release packages"
	@echo "  help             - Show this help message"
