.PHONY: all help
.PHONY: install-deps-service install-deps-cli install-deps-app install-deps-all
.PHONY: generate-service generate-app generate-all
.PHONY: typecheck-app
.PHONY: lint-service lint-cli lint-app lint-all
.PHONY: validate-spec-app
.PHONY: build-service build-cli build-app-ios build-app-android build-app build-all
.PHONY: test-unit-service test-unit-cli test-unit-app test-unit-all
.PHONY: test-integration-service test-integration-cli test-integration-app test-integration-all
.PHONY: test-e2e-service test-e2e-cli test-e2e-app-ios test-e2e-app-android test-e2e-app test-e2e-all
.PHONY: test-contract-service test-contract-app test-contract-all
.PHONY: test-service test-cli test-app test-all
.PHONY: run-service run-cli run-app-ios run-app-ios-device run-app-android run-app
.PHONY: list-ios-devices
.PHONY: clean-service clean-cli clean-cache-app clean-native-app clean-app clean-all
.PHONY: clean-service-full clean-cli-full clean-app-full clean-all-full
.PHONY: rebuild-app-ios rebuild-app-android fix-app
.PHONY: release deploy undeploy coverage deps deps-update

# ============================================================================
# Build Variables
# ============================================================================

# Service & CLI binaries
SERVICE_BINARY_NAME := boardingpass
CLI_BINARY_NAME := boarding
OUTPUT_DIR := _output
BIN_DIR := $(OUTPUT_DIR)/bin
DIST_DIR := $(OUTPUT_DIR)/dist
COVERAGE_DIR := $(OUTPUT_DIR)/coverage

# Architecture detection for RPM deployment
UNAME_ARCH := $(shell uname -m)
RPM_ARCH := $(shell echo $(UNAME_ARCH) | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g')

# Container name for deployment (can be overridden for tests)
CONTAINER_NAME ?= boardingpass-bootc
IMAGE_NAME ?= boardingpass-bootc:latest
CONTAINERFILE ?= build/Containerfile.bootc

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

# Mobile app (npm ci in CI, npm install locally)
NPM_INSTALL := $(if $(CI),npm ci,npm install)
MOBILE_DIR := mobile
MOBILE_IOS_DIR := $(MOBILE_DIR)/ios
MOBILE_ANDROID_DIR := $(MOBILE_DIR)/android
MOBILE_EXPO_CACHE := $(MOBILE_DIR)/.expo
MOBILE_NODE_MODULES := $(MOBILE_DIR)/node_modules
MOBILE_CACHE_DIRS := $(MOBILE_EXPO_CACHE) $(MOBILE_NODE_MODULES)/.cache

# Platform detection (for default app targets)
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	DEFAULT_PLATFORM := ios
else
	DEFAULT_PLATFORM := android
endif

# iOS simulator default (can be overridden with IOS_DEVICE env var)
IOS_DEVICE ?= iPhone 17 Pro

# Extra args for app unit tests (e.g., APP_TEST_ARGS="-- --coverage --maxWorkers=2")
APP_TEST_ARGS ?=

# Physical iOS device name (can be overridden with IOS_PHYSICAL_DEVICE env var)
# To find your device name, run: xcrun devicectl list devices
# Use the exact name shown in the "Name" column
IOS_PHYSICAL_DEVICE ?= a phone

# Enable mDNS entitlement (requires paid Apple Developer account)
# Set to "false" to disable: make build-app-ios ENABLE_MDNS_ENTITLEMENT=false
ENABLE_MDNS_ENTITLEMENT ?= true

# ============================================================================
# Default Target
# ============================================================================

## all: Fast feedback loop - lint, unit tests, build (no slow integration/e2e)
all: lint-all test-unit-all build-all

# ============================================================================
# Help
# ============================================================================

## help: Display this help message
help:
	@echo ""
	@echo "BoardingPass Build System"
	@echo ""
	@echo "Components: service, cli, app"
	@echo ""
	@echo "Common Patterns:"
	@echo "  {action}-service        - Action for BoardingPass service"
	@echo "  {action}-cli            - Action for boarding CLI tool"
	@echo "  {action}-app            - Action for mobile app"
	@echo "  {action}-all            - Action for all components"
	@echo ""
	@echo "Installation:"
	@echo "  install-deps-service    - Install Go dependencies"
	@echo "  install-deps-cli        - Install Go dependencies (same as service)"
	@echo "  install-deps-app        - Install npm dependencies"
	@echo "  install-deps-all        - Install all dependencies"
	@echo ""
	@echo "Code Generation:"
	@echo "  generate-service        - Generate mocks (go generate)"
	@echo "  generate-app            - Generate TypeScript types from OpenAPI"
	@echo "  generate-all            - Generate all code"
	@echo ""
	@echo "Type Checking:"
	@echo "  typecheck-app           - Run TypeScript type check (generates types first)"
	@echo ""
	@echo "Linting:"
	@echo "  lint-service            - Run golangci-lint on service"
	@echo "  lint-cli                - Run golangci-lint on CLI"
	@echo "  lint-app                - Generate types + typecheck + ESLint + react-doctor"
	@echo "  lint-all                - Lint all components"
	@echo ""
	@echo "Building:"
	@echo "  build-service           - Build service binary"
	@echo "  build-cli               - Build CLI binary"
	@echo "  build-app-ios           - Generate iOS native project (expo prebuild)"
	@echo "                            Use ENABLE_MDNS_ENTITLEMENT=false without paid Apple Developer account"
	@echo "  build-app-android       - Generate Android native project"
	@echo "  build-app               - Generate both iOS and Android projects"
	@echo "  build-all               - Build all components"
	@echo ""
	@echo "Testing:"
	@echo "  test-unit-{component}   - Run unit tests"
	@echo "  test-integration-{component} - Run integration tests"
	@echo "  test-e2e-{component}    - Run end-to-end tests"
	@echo "  validate-spec-app       - Validate OpenAPI spec"
	@echo "  test-contract-{service|app} - Run contract tests"
	@echo "  test-{component}        - Run all tests for component"
	@echo "  test-all                - Run all tests for all components"
	@echo ""
	@echo "Running:"
	@echo "  run-service             - Start BoardingPass service"
	@echo "  run-cli                 - Run boarding CLI (interactive)"
	@echo "  run-app-ios             - Run app on iOS simulator (default: $(IOS_DEVICE))"
	@echo "  run-app-ios-device      - Run app on physical iOS device (default: $(IOS_PHYSICAL_DEVICE))"
	@echo "  run-app-android         - Run app on Android emulator"
	@echo "  run-app                 - Run app on default platform ($(DEFAULT_PLATFORM))"
	@echo "  list-ios-devices        - List all available iOS devices and simulators"
	@echo ""
	@echo "Cleaning (build artifacts only):"
	@echo "  clean-service           - Remove service binary"
	@echo "  clean-cli               - Remove CLI binary"
	@echo "  clean-cache-app         - Clear Metro cache (.expo, node_modules/.cache)"
	@echo "  clean-native-app        - Remove native projects (ios/, android/)"
	@echo "  clean-app               - Clean cache + native projects"
	@echo "  clean-all               - Clean all components"
	@echo ""
	@echo "Cleaning (build + dependencies):"
	@echo "  clean-service-full      - Same as clean-service (Go downloads as needed)"
	@echo "  clean-cli-full          - Same as clean-cli"
	@echo "  clean-app-full          - Clean app + remove node_modules"
	@echo "  clean-all-full          - Deep clean everything"
	@echo ""
	@echo "App-Specific Workflows:"
	@echo "  clean-cache-app         - Clear Metro cache only (Issue #2)"
	@echo "  clean-native-app        - Remove ios/android only (Issue #1, #3)"
	@echo "  rebuild-app-ios         - Full rebuild: clean + build + run (iOS)"
	@echo "  rebuild-app-android     - Full rebuild: clean + build + run (Android)"
	@echo "  fix-app                 - Fix common issues: expo install --fix + clean + prebuild"
	@echo ""
	@echo "Other:"
	@echo "  release                 - Build release packages (RPM, DEB, archives)"
	@echo "  deploy                  - Deploy service to bootc container"
	@echo "  undeploy                - Stop and remove bootc container"
	@echo "  coverage                - Generate test coverage report"
	@echo "  deps                    - Download and verify Go dependencies"
	@echo "  deps-update             - Update Go dependencies"
	@echo ""

# ============================================================================
# Installation
# ============================================================================

## install-deps-service: Download Go dependencies
install-deps-service:
	@echo "Downloading Go dependencies..."
	@$(GOMOD) download
	@$(GOMOD) verify

## install-deps-cli: Download Go dependencies (same as service)
install-deps-cli: install-deps-service

## install-deps-app: Install npm dependencies for mobile app
install-deps-app:
	@echo "Installing mobile app dependencies..."
	@cd $(MOBILE_DIR) && $(NPM_INSTALL)

## install-deps-all: Install all dependencies
install-deps-all: install-deps-service install-deps-app
	@echo "All dependencies installed"

# ============================================================================
# Code Generation
# ============================================================================

## generate-service: Generate Go code (mocks, etc.)
generate-service:
	@echo "Generating Go code (mocks)..."
	@$(GOCMD) generate ./...
	@echo "Go code generation complete"

## generate-app: Generate TypeScript types from OpenAPI spec
generate-app:
	@echo "Generating TypeScript types from OpenAPI spec..."
	@cd $(MOBILE_DIR) && npm run generate:types
	@echo "TypeScript type generation complete"

## generate-all: Generate all code
generate-all: generate-service generate-app
	@echo "All code generation complete"

# ============================================================================
# Type Checking
# ============================================================================

## typecheck-app: Run TypeScript type check on mobile app (generates types first)
typecheck-app: generate-app
	@echo "Running TypeScript type check..."
	@cd $(MOBILE_DIR) && npm run typecheck

# ============================================================================
# Linting
# ============================================================================

## lint-service: Run golangci-lint on service code
lint-service:
	@echo "Running golangci-lint on service..."
	@golangci-lint run

## lint-cli: Run golangci-lint on CLI code
lint-cli:
	@echo "Running golangci-lint on CLI..."
	@golangci-lint run

## lint-app: Run typecheck + ESLint + react-doctor on mobile app
lint-app: typecheck-app
	@echo "Running ESLint on mobile app..."
	@cd $(MOBILE_DIR) && npm run lint
	@echo "Running react-doctor on mobile app..."
	@cd $(MOBILE_DIR) && npx -y react-doctor@latest . --verbose --diff

## lint-all: Lint all components
lint-all: lint-service lint-app
	@echo "All linting complete"

# ============================================================================
# Building
# ============================================================================

## build-service: Build the BoardingPass service binary
build-service:
	@echo "Building $(SERVICE_BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(SERVICE_BINARY_NAME) ./cmd/boardingpass
	@echo "Binary built: $(BIN_DIR)/$(SERVICE_BINARY_NAME)"
	@ls -lh $(BIN_DIR)/$(SERVICE_BINARY_NAME)

## build-cli: Build the boarding CLI tool
build-cli:
	@echo "Building $(CLI_BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(CLI_BINARY_NAME) ./cmd/boarding
	@echo "Binary built: $(BIN_DIR)/$(CLI_BINARY_NAME)"
	@ls -lh $(BIN_DIR)/$(CLI_BINARY_NAME)

## build-app-ios: Generate iOS native project
build-app-ios: generate-app
	@echo "Generating iOS native project..."
	@if [ "$(ENABLE_MDNS_ENTITLEMENT)" = "false" ]; then \
		echo "Note: mDNS entitlement disabled (no paid Apple Developer account)"; \
	fi
	@cd $(MOBILE_DIR) && ENABLE_MDNS_ENTITLEMENT=$(ENABLE_MDNS_ENTITLEMENT) npx expo prebuild --platform ios $(if $(CI),--clean,)
	@echo "iOS project generated: $(MOBILE_IOS_DIR)/"

## build-app-android: Generate Android native project
build-app-android: generate-app
	@echo "Generating Android native project..."
	@cd $(MOBILE_DIR) && npx expo prebuild --platform android $(if $(CI),--clean,)
	@echo "Android project generated: $(MOBILE_ANDROID_DIR)/"

## build-app: Generate both iOS and Android native projects
build-app: build-app-ios build-app-android
	@echo "All native projects generated"

## build-all: Build all components
build-all: build-service build-cli build-app
	@echo "All components built"

# ============================================================================
# Testing - Unit
# ============================================================================

## test-unit-service: Run service unit tests
test-unit-service:
	@echo "Running service unit tests..."
	@$(GOTESTSUM) --format pkgname -- -race -short ./...

## test-unit-cli: Run CLI unit tests
test-unit-cli:
	@echo "Running CLI unit tests..."
	@$(GOTESTSUM) --format pkgname -- -race -short ./...

## test-unit-app: Run mobile app unit tests
test-unit-app:
	@echo "Running mobile app unit tests..."
	@cd $(MOBILE_DIR) && npm test $(APP_TEST_ARGS)

## test-unit-all: Run all unit tests
test-unit-all: test-unit-service test-unit-app
	@echo "All unit tests complete"

# ============================================================================
# Testing - Integration
# ============================================================================

## test-integration-service: Run service integration tests
test-integration-service:
	@echo "Running service integration tests..."
	@$(GOTEST) -v -race -run Integration ./tests/integration/...

## test-integration-cli: Run CLI integration tests
test-integration-cli:
	@echo "Running CLI integration tests..."
	@$(GOTEST) -v -race -run Integration ./tests/integration/...

## test-integration-app: Run mobile app integration tests
test-integration-app:
	@echo "Running mobile app integration tests..."
	@cd $(MOBILE_DIR) && npm run test:integration

## test-integration-all: Run all integration tests
test-integration-all: test-integration-service test-integration-app
	@echo "All integration tests complete"

# ============================================================================
# Testing - E2E
# ============================================================================

## test-e2e-service: Run service end-to-end tests
test-e2e-service:
	@echo "Running service E2E tests..."
	@echo "Note: This requires podman or docker to be installed"
	@$(GOTEST) -v -timeout 10m ./tests/e2e/...

## test-e2e-cli: Run CLI end-to-end tests
test-e2e-cli:
	@echo "Running CLI E2E tests..."
	@echo "Note: This requires podman or docker to be installed"
	@$(GOTEST) -v -timeout 10m ./tests/cli-e2e/...

## test-e2e-app-ios: Run mobile app E2E tests on iOS
test-e2e-app-ios:
	@echo "Running mobile app E2E tests on iOS..."
	@cd $(MOBILE_DIR) && npm run e2e:test:ios

## test-e2e-app-android: Run mobile app E2E tests on Android
test-e2e-app-android:
	@echo "Running mobile app E2E tests on Android..."
	@cd $(MOBILE_DIR) && npm run e2e:test:android

## test-e2e-app: Run mobile app E2E tests on default platform
test-e2e-app: test-e2e-app-$(DEFAULT_PLATFORM)

## test-e2e-all: Run all E2E tests
test-e2e-all: test-e2e-service test-e2e-cli test-e2e-app
	@echo "All E2E tests complete"

# ============================================================================
# Testing - Contract
# ============================================================================

## test-contract-service: Run service contract tests
test-contract-service:
	@echo "Running service contract tests..."
	@$(GOTEST) -v ./tests/contract/...

## validate-spec-app: Validate OpenAPI spec
validate-spec-app:
	@echo "Validating OpenAPI spec..."
	@cd $(MOBILE_DIR) && npm run validate:spec

## test-contract-app: Run mobile app contract tests (validates spec + generates types first)
test-contract-app: validate-spec-app generate-app
	@echo "Running mobile app contract tests..."
	@cd $(MOBILE_DIR) && npm run test:contract

## test-contract-all: Run all contract tests
test-contract-all: test-contract-service test-contract-app
	@echo "All contract tests complete"

# ============================================================================
# Testing - Aggregates
# ============================================================================

## test-service: Run all service tests (unit + integration + e2e + contract)
test-service: test-unit-service test-integration-service test-e2e-service test-contract-service
	@echo "All service tests complete"

## test-cli: Run all CLI tests (unit + integration + e2e)
test-cli: test-unit-cli test-integration-cli test-e2e-cli
	@echo "All CLI tests complete"

## test-app: Run all mobile app tests (unit + integration + e2e + contract)
test-app: test-unit-app test-integration-app test-e2e-app test-contract-app
	@echo "All mobile app tests complete"

## test-all: Run all tests for all components
test-all: test-service test-cli test-app
	@echo "All tests complete"

# ============================================================================
# Running
# ============================================================================

## run-service: Start BoardingPass service
run-service: build-service
	@echo "Starting BoardingPass service..."
	@$(BIN_DIR)/$(SERVICE_BINARY_NAME)

## run-cli: Run boarding CLI tool
run-cli: build-cli
	@echo "Running boarding CLI..."
	@$(BIN_DIR)/$(CLI_BINARY_NAME)

## run-app-ios: Run mobile app on iOS simulator (default: $(IOS_DEVICE))
run-app-ios:
	@echo "Starting Metro bundler and running on $(IOS_DEVICE)..."
	@cd $(MOBILE_DIR) && npx expo run:ios --device "$(IOS_DEVICE)"

## run-app-ios-device: Run mobile app on connected physical iOS device
run-app-ios-device:
	@echo "Starting Metro bundler and running on iOS device: $(IOS_PHYSICAL_DEVICE)..."
	@echo "To use a different device, run: make run-app-ios-device IOS_PHYSICAL_DEVICE='<device-name>'"
	@echo "Or list devices with: make list-ios-devices"
	@cd $(MOBILE_DIR) && npx expo run:ios --device "$(IOS_PHYSICAL_DEVICE)"

## list-ios-devices: List all available iOS devices (simulators and physical)
list-ios-devices:
	@echo "=== Physical iOS Devices ==="
	@xcrun devicectl list devices 2>&1 || echo "Error listing physical devices (devicectl may not be available)"
	@echo ""
	@echo "=== iOS Simulators ==="
	@xcrun simctl list devices available | grep -E "^\s+" | grep -v "unavailable" || echo "No simulators found"
	@echo ""
	@echo "To run on a specific device, use:"
	@echo "  make run-app-ios IOS_DEVICE='<simulator-name>'"
	@echo "  make run-app-ios-device IOS_PHYSICAL_DEVICE='<device-name>'"

## run-app-android: Run mobile app on Android emulator
run-app-android:
	@echo "Starting Metro bundler and running on Android..."
	@cd $(MOBILE_DIR) && npm run android

## run-app: Run mobile app on default platform ($(DEFAULT_PLATFORM))
run-app: run-app-$(DEFAULT_PLATFORM)

# ============================================================================
# Cleaning - Standard (build artifacts only)
# ============================================================================

## clean-service: Remove service binary
clean-service:
	@echo "Cleaning service build artifacts..."
	@rm -f $(BIN_DIR)/$(SERVICE_BINARY_NAME)
	@rm -rf $(DIST_DIR)
	@echo "Service cleaned"

## clean-cli: Remove CLI binary
clean-cli:
	@echo "Cleaning CLI build artifacts..."
	@rm -f $(BIN_DIR)/$(CLI_BINARY_NAME)
	@echo "CLI cleaned"

## clean-cache-app: Clear Metro cache only (Issue #2: Metro cache corruption)
clean-cache-app:
	@echo "Clearing Metro bundler cache..."
	@rm -rf $(MOBILE_EXPO_CACHE)
	@rm -rf $(MOBILE_NODE_MODULES)/.cache
	@echo "Metro cache cleared"

## clean-native-app: Remove ios/ and android/ directories only (Issue #1, #3: Native rebuild needed)
clean-native-app:
	@echo "Removing native projects..."
	@rm -rf $(MOBILE_IOS_DIR)
	@rm -rf $(MOBILE_ANDROID_DIR)
	@echo "Native projects removed"

## clean-app: Clean Metro cache + native projects (standard clean)
clean-app: clean-cache-app clean-native-app
	@echo "Mobile app cleaned"

## clean-all: Clean all components
clean-all: clean-service clean-cli clean-app
	@echo "All components cleaned"

# ============================================================================
# Cleaning - Full (build + dependencies)
# ============================================================================

## clean-service-full: Same as clean-service (Go downloads dependencies as needed)
clean-service-full: clean-service
	@echo "Service full clean complete"

## clean-cli-full: Same as clean-cli (Go downloads dependencies as needed)
clean-cli-full: clean-cli
	@echo "CLI full clean complete"

## clean-app-full: Clean app + remove node_modules (deep clean)
clean-app-full: clean-app
	@echo "Removing node_modules..."
	@rm -rf $(MOBILE_NODE_MODULES)
	@echo "Mobile app full clean complete"

## clean-all-full: Deep clean everything
clean-all-full: clean-service-full clean-cli-full clean-app-full
	@echo "Full clean complete"
	@rm -rf $(OUTPUT_DIR)
	@echo "Output directory removed"

# ============================================================================
# App-Specific Workflows (from troubleshooting experience)
# ============================================================================

## rebuild-app-ios: Full rebuild for iOS (clean + build + run)
rebuild-app-ios: clean-app build-app-ios run-app-ios

## rebuild-app-android: Full rebuild for Android (clean + build + run)
rebuild-app-android: clean-app build-app-android run-app-android

## fix-app: Fix common app issues (Xcode 26, missing deps, cache)
fix-app:
	@echo "Fixing common mobile app issues..."
	@echo "Step 1: Running 'expo install --fix' to update dependencies..."
	@cd $(MOBILE_DIR) && npx expo install --fix
	@echo "Step 2: Cleaning cache and native projects..."
	@$(MAKE) clean-app
	@echo "Step 3: Rebuilding native projects..."
	@$(MAKE) build-app
	@echo "Fix complete. Run 'make run-app-ios' or 'make run-app-android' to start the app."

# ============================================================================
# Release & Deployment (preserve existing functionality)
# ============================================================================

## release: Build release packages (RPM, DEB, archives)
release:
	@echo "Building release packages..."
	@goreleaser release --snapshot --clean --skip=publish
	@echo "Packages built in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/*.rpm $(DIST_DIR)/*.deb 2>/dev/null || true

## deploy: Build RPM and deploy to local RHEL bootc container
deploy: release
	@echo "Detected architecture: $(UNAME_ARCH) -> $(RPM_ARCH)"
	@echo "Removing old known certificates file (if exists)..."
	@rm -f ~/.config/boardingpass/known_certs.yaml
	@echo "Building bootc container image for $(RPM_ARCH)..."
	@podman build -f $(CONTAINERFILE) --build-arg ARCH=$(RPM_ARCH) -t $(IMAGE_NAME) .
	@echo "Running bootc container..."
	@echo "Container will run with systemd. Use 'podman ps' to see running containers."
	@echo "Use 'podman exec -it $(CONTAINER_NAME) journalctl -u boardingpass' to view logs."
	@podman run -d --name $(CONTAINER_NAME) --rm \
		-p 8443:8443 \
		--tmpfs /tmp \
		--tmpfs /run \
		-v /sys/fs/cgroup:/sys/fs/cgroup:rw \
		--cgroupns=host \
		--privileged \
		$(IMAGE_NAME)
	@echo "Bootc container started. Container name: $(CONTAINER_NAME)"
	@echo ""
	@echo "Log in with: boarding -y pass --host localhost --username boardingpass --password $$(podman exec $(CONTAINER_NAME) sh -c "ip link show | grep -A1 '^2:' | grep 'link/ether' | awk '{print \$$2}' | head -n1")"
	@echo ""
	@echo "Stop with: podman stop $(CONTAINER_NAME)"

## undeploy: Stop and remove the bootc container
undeploy:
	@echo "Stopping and removing $(CONTAINER_NAME) container..."
	@podman stop $(CONTAINER_NAME) 2>/dev/null || true
	@podman rm $(CONTAINER_NAME) 2>/dev/null || true
	@echo "Container removed"

## coverage: Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTESTSUM) --format pkgname -- -race -short -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic -coverpkg=./internal/...,./pkg/... ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out | tee $(COVERAGE_DIR)/coverage.txt
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

# ============================================================================
# Development Helpers (preserve existing)
# ============================================================================

## deps: Download and verify Go dependencies
deps: install-deps-service

## deps-update: Update Go dependencies
deps-update:
	@echo "Updating Go dependencies..."
	@$(GOMOD) tidy
	@$(GOMOD) verify
