# BoardingPass API - Quickstart Guide

**Feature**: BoardingPass API
**Branch**: `001-boardingpass-api`
**Version**: 0.1.0
**Date**: 2025-12-06

## Overview

This guide provides step-by-step instructions for setting up a local development environment, building the BoardingPass service, running tests, and creating distributable packages.

---

## Prerequisites

### Required Tools

- **Podman** (or Docker): Container runtime for reproducible builds
- **Git**: Source control
- **Text Editor**: VSCode, Vim, Emacs, etc.
- **curl**: For API testing (optional)

### Platform Support

- **Linux**: Native development (RHEL 9+, Rocky Linux 9+, AlmaLinux 9+, Debian 12+, Ubuntu 22.04+ LTS)
- **macOS**: Container-based development only
- **Windows**: WSL2 + Podman/Docker

---

## Repository Setup

### Clone the Repository

```bash
git clone https://github.com/fzdarsky/boardingpass.git
cd boardingpass
```

### Checkout Feature Branch

```bash
git checkout 001-boardingpass-api
```

### Repository Structure

```
boardingpass/
├── cmd/boardingpass/          # Main service binary
├── internal/                  # Private packages (API, auth, provisioning, etc.)
├── pkg/protocol/              # Public protocol definitions
├── tests/                     # Unit, integration, contract, E2E tests
├── build/                     # Build artifacts, systemd unit, sudoers config
├── _output/                   # Build output directory (gitignored)
├── .goreleaser.yaml           # GoReleaser configuration
├── .golangci.yml              # golangci-lint configuration
├── go.mod                     # Go module definition
└── README.md                  # Project overview
```

---

## Development Environment Setup

### Option 1: Native Go Development (Linux Only)

#### Install Go 1.25+

**RHEL 9+ / Rocky Linux / AlmaLinux**:
```bash
# Enable UBI9 Go Toolset
sudo dnf install -y go-toolset-1.25
```

**Debian 12+ / Ubuntu 22.04+**:
```bash
# Install Go from official binaries
wget https://go.dev/dl/go1.25.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.25.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

#### Install golangci-lint 2.7.1

```bash
curl -sSfL https://github.com/golangci/golangci-lint/releases/download/v2.7.1/golangci-lint-2.7.1-linux-amd64.tar.gz | \
  sudo tar -xz -C /usr/local/bin --strip-components=1 golangci-lint-2.7.1-linux-amd64/golangci-lint
```

#### Install GoReleaser

```bash
curl -sSfL https://github.com/goreleaser/goreleaser/releases/download/v1.23.0/goreleaser_Linux_x86_64.tar.gz | \
  sudo tar -xz -C /usr/local/bin goreleaser
```

#### Verify Installations

```bash
go version          # Should show go1.25 or higher
golangci-lint version   # Should show v2.7.1
goreleaser --version    # Should show v1.23.0 or higher
```

---

### Option 2: Container-Based Development (All Platforms)

This approach uses a containerized build environment with UBI9 Go Toolset, ensuring consistent builds across all platforms.

#### Install Podman (or Docker)

**RHEL 9+ / Rocky Linux / AlmaLinux**:
```bash
sudo dnf install -y podman
```

**Debian 12+ / Ubuntu 22.04+**:
```bash
sudo apt update
sudo apt install -y podman
```

**macOS**:
```bash
brew install podman
podman machine init
podman machine start
```

#### Build the Builder Image

```bash
podman build -t boardingpass-builder -f build/Containerfile .
```

This creates a reproducible build environment based on `registry.access.redhat.com/ubi9/go-toolset:1.25` with Go 1.25, golangci-lint v2.7.1, and GoReleaser pre-installed.

---

## Building the Service

### Native Build (Go Installed Locally)

#### Download Dependencies

```bash
go mod download
go mod verify
```

#### Build Binary

```bash
# Development build (includes debug symbols)
go build -o _output/bin/boardingpass ./cmd/boardingpass

# Production build (stripped, optimized)
go build -ldflags="-s -w" -o _output/bin/boardingpass ./cmd/boardingpass
```

#### Cross-Compile for ARM64

```bash
GOOS=linux GOARCH=arm64 go build -o _output/bin/boardingpass-arm64 ./cmd/boardingpass
```

---

### Container Build (Podman/Docker)

#### Quick Build (Single-Target)

```bash
# Build for current architecture only
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder \
  goreleaser build --snapshot --clean --single-target

# Output: _output/dist/<os>_<arch>/boardingpass
```

#### Full Build (All Targets)

```bash
# Build for all targets (linux/amd64, linux/arm64)
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder \
  goreleaser build --snapshot --clean

# Output: _output/dist/<os>_<arch>/boardingpass
```

#### Build Packages (RPM/DEB)

```bash
# Generate RPM and DEB packages
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder \
  goreleaser release --snapshot --clean

# Output:
#   _output/dist/*.rpm
#   _output/dist/*.deb
#   _output/dist/*.tar.gz
#   _output/dist/checksums.txt
```

---

### Makefile Shortcuts (Container Build)

Create a `Makefile` in the repository root for convenience:

```makefile
.PHONY: build test lint coverage clean

PODMAN := podman
BUILDER_IMAGE := boardingpass-builder

build-image:
\t$(PODMAN) build -t $(BUILDER_IMAGE) -f build/Containerfile .

build: build-image
\t$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) \
\t\tgoreleaser build --snapshot --clean

test: build-image
\t$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) \
\t\tgo test -v -race -cover ./...

lint: build-image
\t$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) \
\t\tgolangci-lint run --verbose

coverage: build-image
\t@mkdir -p _output/coverage
\t$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) \
\t\tgo test -v -race -coverprofile=_output/coverage/coverage.out -covermode=atomic ./...
\t$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) \
\t\tgo tool cover -html=_output/coverage/coverage.out -o _output/coverage/coverage.html
\t@echo "Coverage report: _output/coverage/coverage.html"

clean:
\t rm -rf _output/
```

**Usage**:

```bash
make build      # Build binaries
make test       # Run tests
make lint       # Run linters
make coverage   # Generate coverage report
make clean      # Clean build artifacts
```

---

## Running Tests

### Native Test Execution

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=_output/coverage/coverage.out ./...
go tool cover -html=_output/coverage/coverage.out -o _output/coverage/coverage.html
```

### Container Test Execution

```bash
# Run all tests in container
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder \
  go test -v -race -cover ./...

# Or use Makefile
make test
```

### Test Organization

- **Unit Tests**: `tests/unit/` - Test individual packages in isolation
- **Integration Tests**: `tests/integration/` - Test API endpoints with httptest
- **Contract Tests**: `tests/contract/` - Validate API responses against OpenAPI spec
- **E2E Tests**: `tests/e2e/` - Full workflow tests in containerized systemd environment

---

## Running Linters

### Native Lint Execution

```bash
# Run all linters (including gosec, govulncheck)
golangci-lint run

# Run with verbose output
golangci-lint run --verbose

# Fix auto-fixable issues
golangci-lint run --fix
```

### Container Lint Execution

```bash
# Run linters in container
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder \
  golangci-lint run --verbose

# Or use Makefile
make lint
```

### Linter Configuration

Linters are configured in [.golangci.yml](.golangci.yml):
- **gosec**: Security analysis (includes G101-G404 checks)
- **govulncheck**: Vulnerability scanning
- **errcheck**: Unchecked error detection
- **govet**: Standard Go vet checks
- **staticcheck**: Static analysis
- **gofmt, goimports**: Code formatting

---

## Local Service Execution

### Generate Test Configuration

Create `/etc/boardingpass/config.yaml`:

```yaml
service:
  inactivity_timeout: "10m"
  session_ttl: "30m"
  sentinel_file: "/tmp/boardingpass-issued"  # Use /tmp for testing

transports:
  ethernet:
    enabled: true
    interfaces: []
    address: "127.0.0.1"
    port: 8443
    tls_cert: "/etc/boardingpass/tls/server.crt"
    tls_key: "/etc/boardingpass/tls/server.key"

commands:
  - id: "echo-test"
    path: "/usr/bin/echo"
    args: ["test successful"]

logging:
  level: "debug"
  format: "human"
```

### Generate Self-Signed TLS Certificate

```bash
sudo mkdir -p /etc/boardingpass/tls

# Generate self-signed certificate (valid for 365 days)
sudo openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout /etc/boardingpass/tls/server.key \
  -out /etc/boardingpass/tls/server.crt \
  -days 365 \
  -subj "/CN=localhost"

# Set permissions
sudo chmod 600 /etc/boardingpass/tls/server.key
sudo chmod 644 /etc/boardingpass/tls/server.crt
```

### Generate SRP Verifier

Create `/etc/boardingpass/verifier`:

```json
{
  "username": "boardingpass",
  "salt": "dGVzdHNhbHQxMjM0NTY3ODkw",
  "password_generator": "/usr/local/bin/boardingpass-password-generator"
}
```

Create password generator script `/usr/local/bin/boardingpass-password-generator`:

```bash
#!/bin/bash
# Test password generator - outputs static password for development
echo "test-password-12345"
```

```bash
sudo chmod 500 /usr/local/bin/boardingpass-password-generator
```

### Run Service

```bash
# Run service in foreground
sudo ./_output/bin/boardingpass --config=/etc/boardingpass/config.yaml

# Service will bind to https://127.0.0.1:8443
```

### Test Service

```bash
# Check service is running
curl -k https://127.0.0.1:8443/info
# Should return: {"error":"unauthorized","message":"Invalid or expired session token"}

# TODO: Implement SRP client for authentication testing
```

---

## CI/CD Workflow

### GitHub Actions

The repository includes three CI/CD workflows:

1. **service-ci.yaml**: Lint, test, build on every push/PR
2. **mobile-ci.yaml**: (Future) Mobile app CI/CD
3. **release.yaml**: Build and publish packages on git tags

### Local CI Simulation

Run the same checks that GitHub Actions runs:

```bash
# Step 1: Lint
make lint

# Step 2: Test with coverage threshold
make test
go test -coverprofile=_output/coverage/coverage.out -covermode=atomic ./...
COVERAGE=$(go tool cover -func=_output/coverage/coverage.out | grep total: | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 80" | bc -l) )); then
  echo "ERROR: Coverage ${COVERAGE}% is below 80% threshold"
  exit 1
fi

# Step 3: Build
make build

# Step 4: Check binary size
SIZE=$(stat -c%s _output/dist/boardingpass_linux_amd64/boardingpass)
SIZE_MB=$(echo "scale=2; $SIZE / 1048576" | bc)
if (( $(echo "$SIZE_MB > 10" | bc -l) )); then
  echo "ERROR: Binary size ${SIZE_MB} MB exceeds 10 MB limit"
  exit 1
fi
```

---

## Creating a Release

### Tag a Release

```bash
# Create annotated tag
git tag -a v0.1.0 -m "Initial release - BoardingPass API 0.1.0"

# Push tag to trigger release workflow
git push origin v0.1.0
```

### GitHub Actions Release Workflow

1. Checks out code with full history
2. Builds binaries for linux/amd64 and linux/arm64
3. Generates RPM and DEB packages
4. Creates GitHub release with artifacts
5. Uploads checksums and signatures

### Manual Release Build

```bash
# Create release build locally
goreleaser release --snapshot --clean

# Artifacts:
#   _output/dist/boardingpass_0.1.0_linux_amd64.tar.gz
#   _output/dist/boardingpass_0.1.0_linux_arm64.tar.gz
#   _output/dist/boardingpass-0.1.0-1.x86_64.rpm
#   _output/dist/boardingpass-0.1.0-1.aarch64.rpm
#   _output/dist/boardingpass_0.1.0_amd64.deb
#   _output/dist/boardingpass_0.1.0_arm64.deb
#   _output/dist/checksums.txt
```

---

## Troubleshooting

### Issue: "Permission denied" when running service

**Solution**: Run with sudo or configure capabilities:
```bash
sudo setcap 'cap_net_bind_service=+ep' ./_output/bin/boardingpass
```

### Issue: "Failed to load TLS certificate"

**Solution**: Ensure certificate files exist and have correct permissions:
```bash
ls -l /etc/boardingpass/tls/
# Should show:
# -rw-r--r-- server.crt
# -rw------- server.key
```

### Issue: "Sentinel file exists, exiting"

**Solution**: Remove sentinel file for testing:
```bash
sudo rm /tmp/boardingpass-issued
```

### Issue: "Failed to connect to D-Bus"

**Solution**: Ensure NetworkManager is running:
```bash
systemctl status NetworkManager
# If not running:
sudo systemctl start NetworkManager
```

### Issue: Container build fails with "permission denied"

**Solution**: Ensure SELinux labels are correct (`:Z` flag):
```bash
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder ...
```

---

## Next Steps

1. **Implement Core Packages**: Start with `internal/auth/srp.go` (SRP-6a implementation)
2. **Write Unit Tests**: Test each package in isolation
3. **Implement API Handlers**: Create HTTP handlers in `internal/api/handlers/`
4. **Integration Tests**: Test full API workflows
5. **E2E Tests**: Test service lifecycle with systemd
6. **Documentation**: Complete deployment guide and API documentation

---

## References

- **OpenAPI Specification**: [contracts/openapi.yaml](contracts/openapi.yaml)
- **Data Model**: [data-model.md](data-model.md)
- **Implementation Plan**: [plan.md](plan.md)
- **Research**: [research.md](research.md)
- **Feature Spec**: [spec.md](spec.md)
- **Constitution**: [.specify/memory/constitution.md](../../.specify/memory/constitution.md)

---

**Document Status**: Complete
**Last Updated**: 2025-12-06
