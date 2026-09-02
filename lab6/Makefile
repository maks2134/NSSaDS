BINARY_NAME=lab6
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
	go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/lab6

.PHONY: build-all
build-all:
	@mkdir -p $(OUTPUT_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%%/*}; \
		arch=$${platform##*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "Building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-$$os-$$arch$$ext ./cmd/lab6; \
	done

.PHONY: run
run: build
	./$(OUTPUT_DIR)/$(BINARY_NAME)

.PHONY: clean
clean:
	rm -rf $(OUTPUT_DIR)
	go clean -cache

.PHONY: help
help:
	@echo "make build       - build lab6 for current platform"
	@echo "make build-all   - build for linux/darwin/windows"
	@echo "make test        - run unit tests"
	@echo "make run         - build and run lab6"
