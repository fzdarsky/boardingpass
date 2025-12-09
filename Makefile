.PHONY: all build test lint coverage clean generate help

# Build variables
BINARY_NAME := boardingpass
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
	@echo "  build       - Build the binary"
	@echo "  test        - Run all tests"
	@echo "  lint        - Run linters"
	@echo "  generate    - Generate code (mocks, etc.)"
	@echo "  coverage    - Generate test coverage report"
	@echo "  clean       - Remove build artifacts"
	@echo "  help        - Display this help message"

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/boardingpass
	@echo "Binary built: $(BIN_DIR)/$(BINARY_NAME)"
	@ls -lh $(BIN_DIR)/$(BINARY_NAME)

## test: Run all tests
test:
	@echo "Running tests..."
	@$(GOTESTSUM) --format pkgname -- -race -short ./...

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
