BINARY_NAME=lab5
OUTPUT_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -s -w"
BUILD_FLAGS=-v
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: all
all: fmt test build

.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	go test -race ./...

.PHONY: build
build:
	@mkdir -p $(OUTPUT_DIR)
	go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/lab5

.PHONY: build-all
build-all:
	@mkdir -p $(OUTPUT_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%%/*}; \
		arch=$${platform##*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "Building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-$$os-$$arch$$ext ./cmd/lab5; \
	done

.PHONY: run-ping
run-ping: build
	./$(OUTPUT_DIR)/$(BINARY_NAME) ping $(HOSTS)

.PHONY: clean
clean:
	rm -rf $(OUTPUT_DIR)
	go clean -cache

.PHONY: help
help:
	@echo "make build       - build lab5 for current platform"
	@echo "make build-all   - build for linux/darwin/windows"
	@echo "make test        - run unit tests"
	@echo "make run-ping HOSTS='8.8.8.8 1.1.1.1'"
