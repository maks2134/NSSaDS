BINARY_NAME=lab8
OUTPUT_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -s -w"
BUILD_FLAGS=-v
N ?= 2560
PANEL ?= 64
NP ?= 4
GROUPS ?= 2
SEED ?= 1
MODE ?= compare
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: all
all: fmt test build

.PHONY: deps
deps:
	go mod tidy

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	go test ./internal/domain/...

.PHONY: build
build:
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=1 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/lab8

.PHONY: build-all
build-all:
	@mkdir -p $(OUTPUT_DIR)
	@native_os=$$(go env GOOS); \
	native_arch=$$(go env GOARCH); \
	for platform in $(PLATFORMS); do \
		os=$${platform%%/*}; \
		arch=$${platform##*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out="$(OUTPUT_DIR)/$(BINARY_NAME)-$$os-$$arch$$ext"; \
		echo "Building $$os/$$arch -> $$out"; \
		if [ "$$os" = "$$native_os" ] && [ "$$arch" = "$$native_arch" ] && [ "$$os" != "windows" ]; then \
			CGO_ENABLED=1 GOOS=$$os GOARCH=$$arch \
				go build $(BUILD_FLAGS) $(LDFLAGS) -o "$$out" ./cmd/lab8 || exit 1; \
		else \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
				go build $(BUILD_FLAGS) $(LDFLAGS) -o "$$out" ./cmd/lab8 || exit 1; \
		fi; \
	done
	@echo "Note: only the native non-Windows binary links OpenMPI (CGO)."
	@echo "Build with 'make build' on each lab host for a real MPI binary."

.PHONY: data
data: build
	mpirun -np 1 ./$(OUTPUT_DIR)/$(BINARY_NAME) -gen -n $(N) -groups 1 -mode collective \
		-a data/A.bin -b data/B.bin -outdir data

.PHONY: run-compare
run-compare: build
	mpirun -np $(NP) ./$(OUTPUT_DIR)/$(BINARY_NAME) -mode compare -groups $(GROUPS) \
		-seed $(SEED) -n $(N) -panel $(PANEL) -gen \
		-a data/A.bin -b data/B.bin -outdir data

.PHONY: run-collective
run-collective: build
	mpirun -np $(NP) ./$(OUTPUT_DIR)/$(BINARY_NAME) -mode collective -groups $(GROUPS) \
		-seed $(SEED) -n $(N) -panel $(PANEL) -gen \
		-a data/A.bin -b data/B.bin -outdir data

.PHONY: run-blocking
run-blocking: build
	mpirun -np $(NP) ./$(OUTPUT_DIR)/$(BINARY_NAME) -mode blocking -groups $(GROUPS) \
		-seed $(SEED) -n $(N) -panel $(PANEL) -gen \
		-a data/A.bin -b data/B.bin -outdir data

.PHONY: run-nonblocking
run-nonblocking: build
	mpirun -np $(NP) ./$(OUTPUT_DIR)/$(BINARY_NAME) -mode nonblocking -groups $(GROUPS) \
		-seed $(SEED) -n $(N) -panel $(PANEL) -gen \
		-a data/A.bin -b data/B.bin -outdir data

.PHONY: clean
clean:
	rm -rf $(OUTPUT_DIR) data/*.bin
	go clean -cache

.PHONY: help
help:
	@echo "make build             - build lab8 for current platform (OpenMPI + CGO)"
	@echo "make build-all         - build for linux/darwin/windows amd64+arm64"
	@echo "make test              - unit tests (domain, no MPI)"
	@echo "make data              - generate A.bin/B.bin"
	@echo "make run-compare       - mpirun compare collective vs blocking"
	@echo "make run-collective    - mpirun collective only"
	@echo "make run-blocking      - mpirun blocking P2P only"
	@echo "make run-nonblocking   - mpirun nonblocking P2P only"
	@echo "Variables: N=$(N) PANEL=$(PANEL) NP=$(NP) GROUPS=$(GROUPS) SEED=$(SEED)"
