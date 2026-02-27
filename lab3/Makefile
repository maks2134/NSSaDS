
BINARY_NAME_SERVER=lab3-server
BINARY_NAME_CLIENT=lab3-client
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

GO=go
GOBUILD=$(GO) build $(LDFLAGS)
GOCLEAN=$(GO) clean
GOTEST=$(GO) test -v
GOMOD=$(GO) mod

PLATFORMS=linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64

DEFAULT_HOST=localhost
DEFAULT_PORT=8080
DEFAULT_MAX_CLIENTS=100
DEFAULT_PING_TIMEOUT=30s
DEFAULT_CHUNK_SIZE=1024
DEFAULT_SELECT_TIMEOUT=10ms

.PHONY: help build build-all build-server build-client clean test deps tidy run run-server run-client benchmark

help: ## Show this help message
	@echo 'Lab3: TCP Server with Select() Multiplexing'
	@echo '==============================================='
	@echo ''
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Build targets:'
	@echo '  build          Build server and client for current platform'
	@echo '  build-all      Build for all platforms'
	@echo '  build-server   Build server for current platform'
	@echo '  build-client   Build client for current platform'
	@echo '  clean          Clean build artifacts'
	@echo ''
	@echo 'Run targets:'
	@echo '  run            Build and run server'
	@echo '  run-server    Build and run server'
	@echo '  run-client    Build and run client'
	@echo ''
	@echo 'Test targets:'
	@echo '  test           Run unit tests'
	@echo '  benchmark      Run benchmarks'
	@echo '  test-server    Test server with multiple clients'
	@echo ''
	@echo 'Development targets:'
	@echo '  deps           Install dependencies'
	@echo '  tidy           Clean up dependencies'
	@echo '  fmt            Format code'
	@echo ''
	@echo 'Configuration:'
	@echo '  Host: $(DEFAULT_HOST)'
	@echo '  Port: $(DEFAULT_PORT)'
	@echo '  Max Clients: $(DEFAULT_MAX_CLIENTS)'
	@echo '  Ping Timeout: $(DEFAULT_PING_TIMEOUT)'
	@echo '  Chunk Size: $(DEFAULT_CHUNK_SIZE)'
	@echo '  Select Timeout: $(DEFAULT_SELECT_TIMEOUT)'

build: build-server build-client ## Build server and client

build-all: build-all-server build-all-client ## Build for all platforms

build-all-server: ## Build server for all platforms
	@echo "Building server for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		echo "Building server for $$platform..."; \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME_SERVER)-$$os-$$arch; \
		if [ "$$os" = "windows" ]; then output_name=$$output_name.exe; fi; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD) -o $(BUILD_DIR)/$$output_name ./cmd/server; \
	done
	@echo "All server builds completed in $(BUILD_DIR)/"

build-all-client: ## Build client for all platforms
	@echo "Building client for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		echo "Building client for $$platform..."; \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME_CLIENT)-$$os-$$arch; \
		if [ "$$os" = "windows" ]; then output_name=$$output_name.exe; fi; \
		GOOS=$$os GOARCH=$$arch $(GOBUILD) -o $(BUILD_DIR)/$$output_name ./cmd/client; \
	done
	@echo "All client builds completed in $(BUILD_DIR)/"

build-server: ## Build server
	@echo "Building server..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER) ./cmd/server
	@echo "Server built: $(BUILD_DIR)/$(BINARY_NAME_SERVER)"

build-client: ## Build client
	@echo "Building client..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT) ./cmd/client
	@echo "Client built: $(BUILD_DIR)/$(BINARY_NAME_CLIENT)"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

test: ## Run unit tests
	@echo "Running tests..."
	$(GOTEST) ./...

benchmark: ## Run performance benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

deps: ## Install dependencies
	@echo "Installing dependencies..."
	$(GOMOD) download

tidy: ## Clean up dependencies
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...

run: run-server ## Build and run server

run-server: build-server ## Build and run server
	@echo "Starting server with select() multiplexing..."
	@echo "Configuration: Host=$(DEFAULT_HOST), Port=$(DEFAULT_PORT)"
	@echo "Connect with: telnet $(DEFAULT_HOST) $(DEFAULT_PORT) or nc $(DEFAULT_HOST) $(DEFAULT_PORT)"
	@echo "Or use: make run-client"
	$(BUILD_DIR)/$(BINARY_NAME_SERVER) -host=$(DEFAULT_HOST) -port=$(DEFAULT_PORT)

run-client: build-client ## Build and run client
	@echo "Starting client..."
	@echo "Connecting to: $(DEFAULT_HOST):$(DEFAULT_PORT)"
	$(BUILD_DIR)/$(BINARY_NAME_CLIENT) -host=$(DEFAULT_HOST) -port=$(DEFAULT_PORT)

test-server: build-server ## Test server with multiple clients
	@echo "Testing server with multiple concurrent clients..."
	@echo "Starting server on port 8080..."
	@$(BUILD_DIR)/$(BINARY_NAME_SERVER) -host=localhost -port=8080 > /tmp/server.log 2>&1 & \
	sleep 2 && \
	echo "Testing with 5 concurrent clients..." && \
	for i in {1..5}; do \
		echo "ECHO Client $$i" | nc localhost 8080 & \
	done; \
	wait && \
	sleep 1 && \
	killall $(BINARY_NAME_SERVER) 2>/dev/null || true && \
	echo "Test completed. Check /tmp/server.log for results"

demo: build-server build-client ## Quick demo with server and client
	@echo "Starting Lab3 demo..."
	@echo "===================="
	@echo "1. Starting server..."
	@$(BUILD_DIR)/$(BINARY_NAME_SERVER) -host=localhost -port=8080 > /tmp/demo-server.log 2>&1 & \
	SERVER_PID=$$! && \
	sleep 2 && \
	echo "2. Testing ECHO command..." && \
	echo "ECHO Hello from Lab3!" | nc localhost 8080 && \
	sleep 1 && \
	echo "3. Testing TIME command..." && \
	echo "TIME" | nc localhost 8080 && \
	sleep 1 && \
	echo "4. Stopping server..." && \
	kill $$SERVER_PID 2>/dev/null || true && \
	echo "Demo completed successfully!" && \
	echo "Check /tmp/demo-server.log for server output"

dev-setup: deps tidy fmt ## Set up development environment
	@echo "Setting up development environment..."
	$(MAKE) test

info: ## Show server configuration and features
	@echo 'Lab3: TCP Server with Select() Multiplexing'
	@echo '==============================================='
	@echo ''
	@echo 'Key Features:'
	@echo '  ✓ Single-threaded operation'
	@echo '  ✓ select() I/O multiplexing'
	@echo '  ✓ Dynamic chunk sizing (t = ping * 10)'
	@echo '  ✓ Non-blocking file transfers'
	@echo '  ✓ Concurrent client handling'
	@echo '  ✓ Clean Architecture'
	@echo ''
	@echo 'Technical Details:'
	@echo '  • Uses syscall.Select() for I/O multiplexing'
	@echo '  • Single thread handles multiple clients'
	@echo '  • File descriptor management'
	@echo '  • Timeout-based responsiveness'
	@echo '  • Dynamic chunk size calculation'
	@echo ''
	@echo 'Commands:'
	@echo '  ECHO <text>     - Echo service'
	@echo '  TIME            - Time service'
	@echo '  HELP            - Help information'
	@echo '  CLOSE/EXIT/QUIT - Disconnect'
	@echo ''
	@echo 'Usage Examples:'
	@echo '  make run-server           # Start server'
	@echo '  make run-client           # Start client'
	@echo '  make demo                 # Quick demo'
	@echo '  make test-server          # Test with multiple clients'

.PHONY: build build-server build-client clean test benchmark deps tidy fmt run run-server run-client test-server demo dev-setup info
