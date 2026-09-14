BINARY_NAME=lab8
OUTPUT_DIR=bin
DATA_DIR=data
A_FILE=$(DATA_DIR)/A.bin
B_FILE=$(DATA_DIR)/B.bin
HOSTFILE ?=
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -s -w"
BUILD_FLAGS=-v

# Defaults — override only what you need: make run NP=6 N=1024
N ?= 2560
PANEL ?= 64
NP ?= 4
GROUPS ?= 2
SEED ?= 1

# Local: shared-memory BTL. Cluster (HOSTFILE set): TCP between hosts.
# Override: make run MCA=
ifeq ($(HOSTFILE),)
MCA ?= --mca pml ob1 --mca btl self,vader
else
MCA ?= --mca pml ob1 --mca btl self,tcp
endif

PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

MPIRUN=mpirun $(MCA) $(if $(HOSTFILE),--hostfile $(HOSTFILE)) -np $(NP)
RUN_FLAGS=-n $(N) -panel $(PANEL) -groups $(GROUPS) -seed $(SEED) \
	-a $(A_FILE) -b $(B_FILE) -outdir $(DATA_DIR)
BIN=./$(OUTPUT_DIR)/$(BINARY_NAME)

.PHONY: all deps fmt test build build-all data \
	run run-compare run-collective run-blocking run-nonblocking \
	demo demo-quick show clean help

all: fmt test build

deps:
	go mod tidy

fmt:
	go fmt ./...

test:
	go test ./internal/domain/...

build:
	@mkdir -p $(OUTPUT_DIR) $(DATA_DIR)
	CGO_ENABLED=1 go build $(BUILD_FLAGS) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/lab8

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

# Generate A.bin / B.bin once (no multiply)
data: build
	$(MPIRUN) $(BIN) -mode collective -gen $(RUN_FLAGS)

# --- Lab demo (prefer these) ---

# Main submission run: collective vs blocking P2P, K groups, MPI-IO
run run-compare: build
	@echo "==> compare  NP=$(NP) GROUPS=$(GROUPS) N=$(N)"
	$(MPIRUN) $(BIN) -mode compare -gen $(RUN_FLAGS)

run-collective: build
	@echo "==> collective  NP=$(NP) GROUPS=$(GROUPS) N=$(N)"
	$(MPIRUN) $(BIN) -mode collective -gen $(RUN_FLAGS)

run-blocking: build
	@echo "==> blocking  NP=$(NP) GROUPS=$(GROUPS) N=$(N)"
	$(MPIRUN) $(BIN) -mode blocking -gen $(RUN_FLAGS)

run-nonblocking: build
	@echo "==> nonblocking  NP=$(NP) GROUPS=$(GROUPS) N=$(N)"
	$(MPIRUN) $(BIN) -mode nonblocking -gen $(RUN_FLAGS)

# Full local walkthrough for defense
demo: build
	@echo "======== LAB8 DEMO ========"
	@echo "1) Unit tests"
	@$(MAKE) --no-print-directory test
	@echo ""
	@echo "2) Compare collective vs P2P (groups=$(GROUPS), np=$(NP), n=$(N))"
	@$(MAKE) --no-print-directory run-compare
	@echo ""
	@echo "3) Non-blocking P2P (lab7 mode inside groups)"
	@$(MAKE) --no-print-directory run-nonblocking
	@echo ""
	@echo "4) Result files"
	@$(MAKE) --no-print-directory show
	@echo "======== DONE ========"

# Fast smoke demo (~seconds)
demo-quick:
	@$(MAKE) --no-print-directory demo N=256 NP=4 GROUPS=2

# Cluster: make run HOSTFILE=hostfile NP=6

show:
	@echo "Data files in $(DATA_DIR)/:"
	@ls -lh $(DATA_DIR)/*.bin 2>/dev/null || echo "(none — run make run first)"

clean:
	rm -rf $(OUTPUT_DIR) $(DATA_DIR)/*.bin
	go clean -cache

help:
	@echo "Lab 8 — all demo via make (override only if needed)"
	@echo ""
	@echo "  make demo-quick              full defense walkthrough (N=256, fast)"
	@echo "  make demo                    same with defaults N=$(N) NP=$(NP)"
	@echo "  make run                     compare collective vs blocking + groups + files"
	@echo "  make run-collective          collective only"
	@echo "  make run-blocking            blocking P2P only"
	@echo "  make run-nonblocking         nonblocking P2P only"
	@echo "  make show                    list A/B/group-*.bin"
	@echo "  make data                    generate A.bin/B.bin only"
	@echo "  make test / build / clean"
	@echo ""
	@echo "Cluster:"
	@echo "  make run HOSTFILE=hostfile NP=6"
	@echo ""
	@echo "Optional vars: N=$(N) PANEL=$(PANEL) NP=$(NP) GROUPS=$(GROUPS) SEED=$(SEED) HOSTFILE MCA"
	@echo "Example: make run NP=6 GROUPS=3 N=1024"
