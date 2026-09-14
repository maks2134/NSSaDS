BINARY_NAME=lab7
OUTPUT_DIR=bin
BIN=$(OUTPUT_DIR)/$(BINARY_NAME)
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -s -w"
BUILD_FLAGS=-v

# Defaults for the whole lab demo — override only if needed:
#   make demo N=3072
#   make defend HOSTS="pc1,pc2,pc3"
N ?= 2560
PANEL ?= 64
NP ?= 3
HOSTFILE ?= hostfile
HOSTS ?=

PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

MPIRUN_LOCAL=mpirun -np $(NP)
MPIRUN_CLUSTER=mpirun -np $(NP) --hostfile $(HOSTFILE)
LAB7_FLAGS=-n $(N) -panel $(PANEL)

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
	CGO_ENABLED=1 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BIN) ./cmd/lab7

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
				go build $(BUILD_FLAGS) $(LDFLAGS) -o "$$out" ./cmd/lab7 || exit 1; \
		else \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
				go build $(BUILD_FLAGS) $(LDFLAGS) -o "$$out" ./cmd/lab7 || exit 1; \
		fi; \
	done
	@echo "Note: only the native non-Windows binary links OpenMPI (CGO)."
	@echo "Build with 'make build' on each lab host for a real MPI binary."

# --- Local rehearsal (1 PC, np=3) ---

.PHONY: blocking
blocking: build
	@echo "==> blocking (local, np=$(NP), N=$(N), panel=$(PANEL))"
	$(MPIRUN_LOCAL) ./$(BIN) -mode blocking $(LAB7_FLAGS)

.PHONY: nonblocking
nonblocking: build
	@echo "==> nonblocking (local, np=$(NP), N=$(N), panel=$(PANEL))"
	$(MPIRUN_LOCAL) ./$(BIN) -mode nonblocking $(LAB7_FLAGS)

.PHONY: demo
demo: build
	@echo "==> DEMO local: blocking then nonblocking (compare time + checksum)"
	@echo "--- blocking ---"
	$(MPIRUN_LOCAL) ./$(BIN) -mode blocking $(LAB7_FLAGS)
	@echo "--- nonblocking ---"
	$(MPIRUN_LOCAL) ./$(BIN) -mode nonblocking $(LAB7_FLAGS)
	@echo "==> checksums must match; on localhost speedup may be small"

# --- Submission on 3 PCs ---

.PHONY: hostfile
hostfile:
	@if [ -n "$(HOSTS)" ]; then \
		rm -f $(HOSTFILE); \
		for h in $$(echo "$(HOSTS)" | tr ',' ' '); do \
			echo "$$h slots=1" >> $(HOSTFILE); \
		done; \
		echo "Wrote $(HOSTFILE):"; cat $(HOSTFILE); \
	elif [ ! -f $(HOSTFILE) ]; then \
		cp hostfile.example $(HOSTFILE); \
		echo "Created $(HOSTFILE) from hostfile.example — edit IPs, or:"; \
		echo "  make hostfile HOSTS=192.168.1.10,192.168.1.11,192.168.1.12"; \
		exit 1; \
	else \
		echo "Using existing $(HOSTFILE):"; cat $(HOSTFILE); \
	fi

.PHONY: blocking-cluster
blocking-cluster: build hostfile
	@echo "==> blocking (3 hosts, N=$(N), panel=$(PANEL))"
	$(MPIRUN_CLUSTER) ./$(BIN) -mode blocking $(LAB7_FLAGS)

.PHONY: nonblocking-cluster
nonblocking-cluster: build hostfile
	@echo "==> nonblocking (3 hosts, N=$(N), panel=$(PANEL))"
	$(MPIRUN_CLUSTER) ./$(BIN) -mode nonblocking $(LAB7_FLAGS)

.PHONY: defend
defend: build hostfile
	@echo "==> DEFEND on 3 PCs: blocking then nonblocking"
	@echo "--- blocking ---"
	$(MPIRUN_CLUSTER) ./$(BIN) -mode blocking $(LAB7_FLAGS)
	@echo "--- nonblocking ---"
	$(MPIRUN_CLUSTER) ./$(BIN) -mode nonblocking $(LAB7_FLAGS)
	@echo "==> Show: P=3, same checksum, nonblocking faster; then answer questions (make questions)"

.PHONY: questions
questions:
	@sed -n '/^## Контрольные вопросы/,$$p' README.md

# Back-compat aliases
.PHONY: run-blocking run-nonblocking
run-blocking: blocking
run-nonblocking: nonblocking

.PHONY: clean
clean:
	rm -rf $(OUTPUT_DIR)
	go clean -cache

.PHONY: help
help:
	@echo "Lab7 — show everything via make"
	@echo ""
	@echo "  make demo                 local rehearsal: both modes (1 PC, np=3)"
	@echo "  make defend               submission: both modes on 3 PCs (needs hostfile)"
	@echo "  make hostfile HOSTS=a,b,c write hostfile from three hostnames/IPs"
	@echo "  make blocking             local blocking only"
	@echo "  make nonblocking          local nonblocking only"
	@echo "  make blocking-cluster     cluster blocking only"
	@echo "  make nonblocking-cluster  cluster nonblocking only"
	@echo "  make questions            print lab Q&A"
	@echo "  make build / build-all / test"
	@echo ""
	@echo "Optional overrides (defaults: N=$(N) PANEL=$(PANEL) NP=$(NP) HOSTFILE=$(HOSTFILE)):"
	@echo "  make demo N=3072"
	@echo "  make hostfile HOSTS=192.168.1.10,192.168.1.11,192.168.1.12"
	@echo "  make defend N=3072"
