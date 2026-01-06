.PHONY: all build build-cli build-all test test-unit test-integration test-e2e test-all lint coverage clean generate help

# Build variables
BINARY_NAME := boardingpass
CLI_BINARY_NAME := boarding
OUTPUT_DIR := _output
BIN_DIR := $(OUTPUT_DIR)/bin
DIST_DIR := $(OUTPUT_DIR)/dist
COVERAGE_DIR := $(OUTPUT_DIR)/coverage

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOVET := $(GOCMD) vet
GOTESTSUM := go tool gotestsum

# Build flags
LDFLAGS := -s -w
BUILD_FLAGS := -trimpath -ldflags="$(LDFLAGS)"

# Default target
all: lint test build

## help: Display this help message
help:
	@echo "Available targets:"
	@echo "  build         - Build the BoardingPass service binary"
	@echo "  build-cli     - Build the boarding CLI tool"
	@echo "  build-all     - Build both service and CLI binaries"
	@echo "  test          - Run unit tests (short mode)"
	@echo "  test-unit     - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e      - Run end-to-end tests (requires podman/docker)"
	@echo "  test-all      - Run all tests including e2e"
	@echo "  lint          - Run linters"
	@echo "  generate      - Generate code (mocks, etc.)"
	@echo "  coverage      - Generate test coverage report"
	@echo "  clean         - Remove build artifacts"
	@echo "  help          - Display this help message"

## build: Build the BoardingPass service binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/boardingpass
	@echo "Binary built: $(BIN_DIR)/$(BINARY_NAME)"
	@ls -lh $(BIN_DIR)/$(BINARY_NAME)

## build-cli: Build the boarding CLI tool
build-cli:
	@echo "Building $(CLI_BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(CLI_BINARY_NAME) ./cmd/boarding
	@echo "Binary built: $(BIN_DIR)/$(CLI_BINARY_NAME)"
	@ls -lh $(BIN_DIR)/$(CLI_BINARY_NAME)

## build-all: Build both service and CLI binaries
build-all: build build-cli
	@echo "All binaries built successfully"

## test: Run unit tests (short mode)
test:
	@echo "Running unit tests..."
	@$(GOTESTSUM) --format pkgname -- -race -short ./...

## test-unit: Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@$(GOTESTSUM) --format pkgname -- -race -short ./...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	@$(GOTEST) -v -race -run Integration ./tests/integration/...

## test-e2e: Run end-to-end tests (requires podman/docker)
test-e2e:
	@echo "Running end-to-end tests..."
	@echo "Note: This requires podman or docker to be installed"
	@$(GOTEST) -v -timeout 10m ./tests/e2e/... ./tests/cli-e2e/...

## test-all: Run all tests including e2e
test-all:
	@echo "Running all tests (unit + integration + e2e)..."
	@$(GOTESTSUM) --format pkgname -- -race ./...
	@echo "Running e2e tests..."
	@$(GOTEST) -v -timeout 10m ./tests/e2e/... ./tests/cli-e2e/...

## lint: Run linters
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

## generate: Generate code (mocks, etc.)
generate:
	@echo "Generating code..."
	@$(GOCMD) generate ./...
	@echo "Code generation complete"

## coverage: Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTESTSUM) --format pkgname -- -race -short -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic -coverpkg=./internal/...,./pkg/... ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out | tee $(COVERAGE_DIR)/coverage.txt
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(OUTPUT_DIR)
	@echo "Clean complete"

# Development helpers
.PHONY: deps deps-update

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) verify

## deps-update: Update dependencies
deps-update:
	@echo "Updating dependencies..."
	@$(GOMOD) tidy
	@$(GOMOD) verify
