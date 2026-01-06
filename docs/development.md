# BoardingPass Development Guide

**Version**: 0.1.0
**Last Updated**: 2025-12-09

## Overview

This guide covers local development setup, testing, building, and contributing to the BoardingPass project. For deployment information, see [deployment.md](deployment.md). For API documentation, see [api.md](api.md).

---

## Prerequisites

### Required Tools

- **Go 1.25+**: Primary development language
- **Podman or Docker**: Container runtime for reproducible builds
- **Git**: Source control
- **Make**: Build automation (optional but recommended)
- **golangci-lint 2.7.1**: Code linting and security scanning
- **GoReleaser**: Build orchestration and packaging

### Platform Support

- **Linux**: Full native development support (RHEL 9+, Rocky Linux 9+, AlmaLinux 9+, Debian 12+, Ubuntu 22.04+ LTS)
- **macOS**: Container-based development only
- **Windows**: WSL2 + Podman/Docker

---

## Development Environment Setup

### Option 1: Native Go Development (Linux Only)

#### Install Go 1.25+

**RHEL 9+ / Rocky Linux / AlmaLinux**:
```bash
sudo dnf install -y go-toolset
```

**Debian 12+ / Ubuntu 22.04+**:
```bash
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
go version                  # Should show go1.25 or higher
golangci-lint version      # Should show v2.7.1
goreleaser --version       # Should show v1.23.0 or higher
```

---

### Option 2: Container-Based Development (All Platforms)

This approach uses a containerized build environment with UBI9 Go Toolset, ensuring consistent builds across all platforms.

#### Install Podman

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

## Repository Setup

### Clone and Configure

```bash
# Clone the repository
git clone https://github.com/fzdarsky/boardingpass.git
cd boardingpass

# Install Git hooks (optional)
git config core.hooksPath .githooks

# Download dependencies
go mod download
go mod verify
```

### Repository Structure

```
boardingpass/
├── cmd/boardingpass/          # Main service binary entry point
├── internal/                  # Private packages (not importable externally)
│   ├── api/                   # HTTP handlers, middleware, server lifecycle
│   ├── auth/                  # SRP-6a, session tokens, rate limiting
│   ├── command/               # Allow-listed command execution
│   ├── config/                # YAML config loading and validation
│   ├── inventory/             # System info extraction (TPM, board, CPU, OS, FIPS)
│   ├── lifecycle/             # Sentinel file, inactivity timeout, graceful shutdown
│   ├── logging/               # JSON logging with secret redaction
│   ├── network/               # Interface enumeration, link state, IP addresses
│   ├── provisioning/          # Config bundle parsing, atomic file ops
│   └── tls/                   # Self-signed cert generation, TLS 1.3+ config
├── pkg/protocol/              # Shared types for API and mobile app
├── tests/                     # Unit, integration, contract, and e2e tests
├── build/                     # systemd unit, sudoers config, Containerfile
├── _output/                   # Build artifacts (gitignored)
├── docs/                      # Documentation
├── specs/                     # Feature specifications (SpecKit workflow)
├── .golangci.yaml            # Linter config (v2 format, includes gosec)
├── .goreleaser.yaml          # Build orchestration
├── go.mod                     # Go module definition
├── Makefile                   # Build automation
└── README.md                  # Project overview
```

---

## Building the Service

### Using Makefile

```bash
make build      # Build the BoardingPass service binary
make build-cli  # Build the boarding CLI tool
make build-all  # Build both service and CLI binaries
make release    # Build release packages (RPM, DEB, archives)
make test       # Run unit tests
make lint       # Run linters
make coverage   # Generate test coverage report
make clean      # Clean build artifacts
make generate   # Generate mocks (uses go tool mockgen)
```

### Native Build with GoReleaser

```bash
# Build for current architecture only (fastest for local development)
goreleaser build --snapshot --clean --single-target

# Build for all targets (linux/amd64, linux/arm64)
goreleaser build --snapshot --clean

# Build release packages (RPM, DEB, archives) without publishing
goreleaser release --snapshot --clean --skip=publish

# Or use the Makefile target
make release
```

**Output location**: `_output/dist/`

**What gets built:**
- Binaries for Linux (amd64/arm64), Darwin (amd64/arm64), Windows (amd64/arm64)
- RPM packages: `boardingpass_*_linux_{amd64,arm64}.rpm`
- DEB packages: `boardingpass_*_linux_{amd64,arm64}.deb`
- Archives: `*.tar.gz` (Linux/Darwin), `*.zip` (Windows)

### Container Build

```bash
# Build for current architecture only
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder \
  goreleaser build --snapshot --clean --single-target

# Build for all targets (linux/amd64, linux/arm64)
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder \
  goreleaser build --snapshot --clean

# Build release packages (RPM, DEB, archives)
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder \
  goreleaser release --snapshot --clean --skip=publish
```

---

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test -v ./internal/auth

# Run specific test
go test -v -run TestSessionManager_CreateSession ./internal/auth

# Generate coverage report
make coverage
# Opens _output/coverage/coverage.html in browser
```

### Test Organization

- **Unit Tests**: Co-located with source files (e.g., `internal/auth/session_test.go`)
  - Test individual packages in isolation
  - Use table-driven tests for logic
  - Use `testify/assert` for readability

- **Integration Tests**: `tests/integration/`
  - Test API endpoints with httptest
  - Mock external dependencies using interfaces

- **Contract Tests**: `tests/contract/`
  - Validate API responses against OpenAPI spec

- **E2E Tests**: `tests/e2e/`
  - Full workflow tests in containerized systemd environment
  - Run with: `go test -v ./tests/e2e -short=false`

### Generating Mocks

```bash
# Generate mocks for all interfaces
make generate
```

---

## Linting and Code Quality

### Running Linters

```bash
# Run all linters (including gosec, govulncheck)
make lint

# Or directly:
golangci-lint run

# Run with verbose output
golangci-lint run --verbose

# Fix auto-fixable issues
golangci-lint run --fix
```

### Linter Configuration

Linters are configured in [.golangci.yaml](../.golangci.yaml):

- **gosec**: Security analysis (G101-G404 checks)
- **govulncheck**: Vulnerability scanning
- **errcheck**: Unchecked error detection
- **govet**: Standard Go vet checks
- **staticcheck**: Static analysis
- **gofmt, goimports**: Code formatting
- **revive**: Linting for Go best practices
- **gocritic**: Opinionated linter

### Fixing Common Lint Errors

**Unchecked errors**:
```go
// Bad
file.Close()

// Good
if err := file.Close(); err != nil {
    return fmt.Errorf("failed to close file: %w", err)
}
```

**Exported functions without comments**:
```go
// Bad
func DoSomething() error {

// Good
// DoSomething performs a specific operation and returns an error if it fails.
func DoSomething() error {
```

---

## Local Service Execution

### Setup Test Environment

#### 1. Create Configuration Directory

```bash
sudo mkdir -p /etc/boardingpass/tls
sudo mkdir -p /var/lib/boardingpass/staging
sudo chown -R $USER:$USER /var/lib/boardingpass
```

#### 2. Generate Self-Signed TLS Certificate

```bash
sudo openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout /etc/boardingpass/tls/server.key \
  -out /etc/boardingpass/tls/server.crt \
  -days 365 \
  -subj "/CN=localhost"

sudo chmod 600 /etc/boardingpass/tls/server.key
sudo chmod 644 /etc/boardingpass/tls/server.crt
```

#### 3. Create Password Generator Script

```bash
sudo mkdir -p /usr/local/bin
cat <<'EOF' | sudo tee /usr/local/bin/boardingpass-password-generator
#!/bin/bash
# Test password generator - outputs static password for development
echo "test-password-12345"
EOF
sudo chmod 500 /usr/local/bin/boardingpass-password-generator
```

#### 4. Create SRP Verifier Configuration

```bash
cat <<'EOF' | sudo tee /etc/boardingpass/verifier
{
  "username": "boardingpass",
  "salt": "dGVzdHNhbHQxMjM0NTY3ODkw",
  "password_generator": "/usr/local/bin/boardingpass-password-generator"
}
EOF
sudo chmod 400 /etc/boardingpass/verifier
```

#### 5. Create Service Configuration

```bash
cat <<'EOF' | sudo tee /etc/boardingpass/config.yaml
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

provisioning:
  allowed_paths:
    - /tmp/boardingpass-test/

commands:
  - id: "echo-test"
    path: "/usr/bin/echo"
    args: ["test successful"]
  - id: "hostname"
    path: "/usr/bin/hostname"
    args: []

logging:
  level: "debug"
  format: "human"
EOF
```

### Run Service

```bash
# Run service in foreground (for development/debugging)
sudo ./_output/bin/boardingpass --config=/etc/boardingpass/config.yaml

# Service will bind to https://127.0.0.1:8443

# In another terminal, test the service:
curl -k https://127.0.0.1:8443/info
# Should return: {"error":"unauthorized","message":"Invalid or expired session token"}
```

### Debugging

#### Enable Debug Logging

Set logging level to `debug` in `/etc/boardingpass/config.yaml`:

```yaml
logging:
  level: "debug"
  format: "human"
```

#### Use Delve Debugger

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Run service with debugger
sudo dlv exec ./_output/bin/boardingpass -- --config=/etc/boardingpass/config.yaml

# In Delve console:
(dlv) break main.main
(dlv) continue
```

---

## CI/CD Workflow

### GitHub Actions

The repository includes two CI/CD workflows:

1. **[service-ci.yaml](../.github/workflows/service-ci.yaml)**: Lint, test, build on every push/PR
2. **[release.yaml](../.github/workflows/release.yaml)**: Build and publish packages on git tags

### Local CI Simulation

Run the same checks that GitHub Actions runs:

```bash
# Step 1: Lint
make lint

# Step 2: Test with coverage threshold
make test

# Step 3: Build
make build

# Step 4: Check binary size
SIZE=$(stat -c%s _output/dist/boardingpass_linux_amd64_v1/boardingpass)
SIZE_MB=$(echo "scale=2; $SIZE / 1048576" | bc)
echo "Binary size: ${SIZE_MB} MB (must be < 10 MB)"
```

---

## Contributing Workflow

### 1. Create Feature Branch

```bash
git checkout -b feature/my-new-feature
```

### 2. Make Changes and Test

```bash
# Write code
vim internal/mypackage/myfile.go

# Write tests
vim internal/mypackage/myfile_test.go

# Run tests
go test -v ./internal/mypackage

# Run linters
make lint
```

### 3. Commit Changes

```bash
# Stage changes
git add .

# Commit with descriptive message
git commit -m "feat: add new feature X

- Implements functionality Y
- Adds test coverage for Z
- Updates documentation

Closes #123"
```

### 4. Push and Create Pull Request

```bash
# Push branch
git push origin feature/my-new-feature

# Create pull request on GitHub
# CI will automatically run lint, test, and build checks
```

### Commit Message Format

Follow conventional commits format:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `chore:` Maintenance tasks
- `perf:` Performance improvements

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

### Issue: Tests fail with "permission denied" for /var/lib/boardingpass

**Solution**: Run integration tests in containerized environment or with sudo:
```bash
# Run unit tests only (no permission issues)
go test -short ./...

# Run all tests with sudo (for integration tests)
sudo -E go test ./...
```

---

## Performance Profiling

### CPU Profiling

```bash
# Build with profiling enabled
go test -cpuprofile=cpu.prof -bench=. ./internal/auth

# Analyze profile
go tool pprof cpu.prof
```

### Memory Profiling

```bash
# Build with memory profiling enabled
go test -memprofile=mem.prof -bench=. ./internal/auth

# Analyze profile
go tool pprof mem.prof
```

### Runtime Profiling

```bash
# Enable pprof in development mode
# Add to main.go:
import _ "net/http/pprof"

# Run service
sudo ./_output/bin/boardingpass --config=/etc/boardingpass/config.yaml

# In another terminal, access pprof:
go tool pprof http://localhost:8443/debug/pprof/heap
```

---

## Code Style Guidelines

### General Principles

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting (enforced by linters)
- Keep functions small and focused
- Use guard clauses to handle errors early
- Always wrap errors with context: `fmt.Errorf("context: %w", err)`
- Use table-driven tests for logic

### Naming Conventions

- **Short, descriptive names**: `c` for client, not `myClient`
- **Exported functions must have comments**
- **Use consistent receiver names**: Use short abbreviations (e.g., `sm` for SessionManager)

### Error Handling

```go
// Good
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad
if err := doSomething(); err != nil {
    return err  // Missing context
}
```

### Context Propagation

Always propagate `context.Context` as the first argument in async/I-O functions:

```go
func ProcessRequest(ctx context.Context, req *Request) error {
    // ...
}
```

---

## References

- **Project README**: [README.md](../README.md)
- **Deployment Guide**: [deployment.md](deployment.md)
- **API Documentation**: [api.md](api.md)
- **Security Guide**: [security.md](security.md)
- **OpenAPI Specification**: [specs/001-boardingpass-api/contracts/openapi.yaml](../specs/001-boardingpass-api/contracts/openapi.yaml)
- **Implementation Plan**: [specs/001-boardingpass-api/plan.md](../specs/001-boardingpass-api/plan.md)
- **Feature Specification**: [specs/001-boardingpass-api/spec.md](../specs/001-boardingpass-api/spec.md)

---

**Document Status**: Complete
**Last Updated**: 2025-12-09
