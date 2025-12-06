# Research: BoardingPass API Implementation

**Feature**: BoardingPass API
**Branch**: `001-boardingpass-api`
**Date**: 2025-12-05
**Purpose**: Technical research for implementing the BoardingPass bootstrap service

## Overview

This document captures research findings and technical decisions for implementing the BoardingPass API service. All decisions prioritize FIPS 140-3 compliance, minimal dependencies, and alignment with the BoardingPass constitution principles.

---

## 1. SRP-6a Implementation with Go stdlib crypto

### Decision

Implement SRP-6a (Secure Remote Password) protocol using **only** Go standard library `crypto/*` packages to ensure FIPS 140-3 compliance, with **device-unique password generation** via configurable script.

### Rationale

- **FIPS 140-3 Requirement**: Target platform (RHEL 9+) requires FIPS 140-3 compliance. Using third-party crypto libraries would complicate certification and require additional validation.
- **Go stdlib Cryptography**: Go's `crypto/*` packages are built against certified cryptographic modules when compiled with `GOEXPERIMENT=boringcrypto` on FIPS-enabled systems.
- **Device-Unique Passwords**: Using a generator script (outputting serial #, TPM EK, MAC address) solves the problem of identical passwords across devices provisioned from the same bootc image.
- **Minimal Dependencies**: Using only stdlib eliminates external dependencies and reduces attack surface.

### Password Generation Strategy

**Problem**: Static verifier embedded in bootc image = all devices have same password.

**Solution**: Store username + salt in image, generate password dynamically via user-configured script that outputs device-unique values.

**Generator Script Examples**:
```bash
#!/bin/bash
# /usr/lib/boardingpass/password-generator

# Example 1: Use TPM endorsement key
tpm2_getcap handles-endorsement | grep -A1 "persistent-handle" | tail -1

# Example 2: Use board serial number
dmidecode -s system-serial-number

# Example 3: Use primary network MAC address
cat /sys/class/net/eth0/address | tr -d ':'

# Example 4: Combine multiple sources (most secure)
echo "$(dmidecode -s system-serial-number)-$(cat /sys/class/net/eth0/address | tr -d ':')"
```

**Verifier Storage** (`/etc/boardingpass/verifier` - embedded in bootc image):
```json
{
  "username": "boardingpass",
  "salt": "<base64-encoded-salt>",
  "password_generator": "/usr/lib/boardingpass/password-generator"
}
```

**Runtime Verifier Computation**:
1. Read `username`, `salt`, `password_generator` from `/etc/boardingpass/verifier`
2. Execute `password_generator` script → `password` (device-unique)
3. Compute `x = H(salt | H(username | ":" | password))`
4. Compute `v = g^x % N`
5. Use `v` for SRP-6a handshake verification

**Benefits**:
- ✅ Each device has unique password (derived from hardware identifiers)
- ✅ Password can be printed on device label (scanned by BoardingPass app)
- ✅ No plaintext password storage (computed on-demand and discarded)
- ✅ Salt is unique per bootc image build (prevents rainbow table attacks)
- ✅ User can customize generator script (flexibility for different deployment scenarios)

### SRP-6a Protocol Flow

1. **Client → Server**: Username + ephemeral public value `A = g^a % N`
2. **Server computes password** via generator script
3. **Server computes verifier** `v = g^x % N` where `x = H(salt | H(username | ":" | password))`
4. **Server → Client**: Salt `s` + ephemeral public value `B = (k*v + g^b) % N`
5. **Both compute**: Shared secret `S` and session key `K = H(S)`
6. **Client → Server**: Proof `M1 = H(H(N) XOR H(g) | H(username) | s | A | B | K)`
7. **Server → Client**: Proof `M2 = H(A | M1 | K)` + Session Token

### Go stdlib Components

- `crypto/sha256`: Hash function H (SHA-256)
- `crypto/rand`: Cryptographically secure random number generation (for `a`, `b`, salt)
- `math/big`: Arbitrary precision arithmetic for modular exponentiation
- `crypto/subtle`: Constant-time comparison for proofs (prevent timing attacks)
- `crypto/hmac`: HMAC for session token generation
- `os/exec`: Execute password generator script

### SRP-6a Parameters

- **Group**: RFC 5054 2048-bit group (N, g)
  - `N`: 2048-bit safe prime (hex-encoded constant)
  - `g`: Generator = 2
- **Hash**: SHA-256 (FIPS-approved)
- **k**: Multiplier = H(N | g) (per RFC 5054)

### Security Considerations

- Use `crypto/subtle.ConstantTimeCompare` for all proof comparisons
- Clear sensitive data (ephemeral secrets `a`, `b`, computed password) from memory after use
- Implement rate limiting on authentication attempts (defense against brute force)
- Session tokens include HMAC signature to prevent tampering
- Password generator script permissions: `0500` (executable by boardingpass user only)
- Verifier file permissions: `0400` (read-only by boardingpass user)

### Alternatives Considered

| Alternative | Rejected Because |
|-------------|------------------|
| Static verifier in bootc image | All devices would have identical password; major security risk |
| Third-party SRP libraries (go-srp, srp-go) | FIPS 140-3 compliance unclear; external dependencies violate minimal dependency principle |
| mTLS certificate-based auth | Requires PKI infrastructure; violates "frictionless bootstrapping" principle; complex certificate distribution |
| Password + HTTPS | Vulnerable to server impersonation; no mutual authentication; SRP provides perfect forward secrecy |

---

## 2. GoReleaser Build Orchestration

### Decision

Use **GoReleaser** for build orchestration, cross-compilation, and packaging (RPM/DEB) with outputs directed to `_output/dist/`.

### Rationale

- **Single Tool**: GoReleaser handles cross-compilation, stripping, compression, checksums, and packaging in one declarative configuration.
- **Native RPM/DEB Support**: Built-in support for generating RPM and DEB packages without external tools.
- **Cross-Compilation**: Seamless GOOS/GOARCH cross-compilation for `linux/amd64` and `linux/arm64`.
- **CI/CD Integration**: First-class GitHub Actions integration.
- **Reproducible Builds**: Consistent builds across local development and CI environments.

### Configuration Strategy

**`.goreleaser.yaml` Structure**:
```yaml
version: 2

project_name: boardingpass

before:
  hooks:
    - go mod tidy
    - go mod verify

builds:
  - id: boardingpass
    main: ./cmd/boardingpass
    binary: boardingpass
    env:
      - CGO_ENABLED=0  # Static linking
    goos:
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w  # Strip debug info
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
    flags:
      - -trimpath  # Reproducible builds

archives:
  - id: boardingpass-archive
    format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - build/boardingpass.service
      - build/boardingpass.sudoers
      - build/password-generator.example
      - README.md
      - LICENSE

nfpms:
  - id: boardingpass-packages
    package_name: boardingpass
    vendor: BoardingPass Project
    homepage: https://github.com/fzdarsky/boardingpass
    maintainer: BoardingPass Team <noreply@boardingpass.dev>
    description: Ephemeral bootstrap service for headless Linux devices
    license: Apache-2.0
    formats:
      - rpm
      - deb
    dependencies:
      - systemd
    contents:
      - src: build/boardingpass.service
        dst: /usr/lib/systemd/system/boardingpass.service
      - src: build/boardingpass.sudoers
        dst: /etc/sudoers.d/boardingpass
        file_info:
          mode: 0440
      - src: build/password-generator.example
        dst: /usr/lib/boardingpass/password-generator.example
        file_info:
          mode: 0500
      - src: /dev/null
        dst: /etc/boardingpass/config.yaml
        type: config|noreplace
      - src: /dev/null
        dst: /var/lib/boardingpass/staging
        type: dir
    scripts:
      preinstall: build/scripts/install-hooks.sh
      postinstall: build/scripts/install-hooks.sh

dist: _output/dist  # Output directory

checksum:
  name_template: "checksums.txt"

snapshot:
  name_template: "{{ .Tag }}-SNAPSHOT"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"
```

**Local Development Workflow**:
```bash
# Build only (no packaging)
goreleaser build --snapshot --clean --single-target

# Full release (local testing)
goreleaser release --snapshot --clean

# Output: _output/dist/
```

### Alternatives Considered

| Alternative | Rejected Because |
|-------------|------------------|
| Manual Makefiles | Error-prone; doesn't handle cross-compilation well; lacks checksum/signing automation |
| Separate build + packaging scripts | Fragmented workflow; GoReleaser provides integrated solution |

---

## 3. Logging to stdout/stderr

### Decision

Write all logs to **stdout/stderr** and let systemd's journal capture handle integration with journald.

### Rationale

- **12-Factor App Principle**: Treat logs as event streams to stdout
- **Simpler Implementation**: No need for socket protocol or dependency management
- **Systemd Integration**: Systemd automatically captures stdout/stderr to journald with structured metadata
- **Portable**: Works in container environments, systemd, and manual execution
- **Minimal Dependencies**: No external logging libraries required

### Implementation Strategy

**Structured Logging to stdout**:
```go
// internal/logging/logger.go

package logging

import (
    "encoding/json"
    "fmt"
    "os"
    "time"
)

type Logger struct {
    level string
}

type LogEntry struct {
    Timestamp string                 `json:"timestamp"`
    Level     string                 `json:"level"`
    Message   string                 `json:"message"`
    Fields    map[string]interface{} `json:"fields,omitempty"`
}

func NewLogger(level string) *Logger {
    return &Logger{level: level}
}

func (l *Logger) Info(message string, fields map[string]interface{}) {
    l.log("INFO", message, fields, os.Stdout)
}

func (l *Logger) Error(message string, fields map[string]interface{}) {
    l.log("ERROR", message, fields, os.Stderr)
}

func (l *Logger) log(level, message string, fields map[string]interface{}, output *os.File) {
    entry := LogEntry{
        Timestamp: time.Now().UTC().Format(time.RFC3339),
        Level:     level,
        Message:   message,
        Fields:    redactSecrets(fields),
    }

    jsonBytes, err := json.Marshal(entry)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
        return
    }

    fmt.Fprintln(output, string(jsonBytes))
}

func redactSecrets(fields map[string]interface{}) map[string]interface{} {
    if fields == nil {
        return nil
    }

    redacted := make(map[string]interface{})
    for k, v := range fields {
        if isSecretField(k) {
            redacted[k] = "[REDACTED]"
        } else {
            redacted[k] = v
        }
    }
    return redacted
}

func isSecretField(key string) bool {
    secretKeys := map[string]bool{
        "password": true, "token": true, "secret": true, "key": true,
        "proof": true, "verifier": true, "salt": true, "session": true,
        "content": true, "payload": true, "authorization": true,
    }
    return secretKeys[key]
}
```

**Usage Example**:
```go
logger.Info("API request received", map[string]interface{}{
    "method":     "POST",
    "path":       "/auth/srp/init",
    "client_ip":  "192.168.1.100",
    "request_id": "req-12345",
})
```

**Systemd Integration**:
- Systemd automatically captures stdout → journald (priority 6 = INFO)
- Systemd automatically captures stderr → journald (priority 3 = ERROR)
- Logs viewable via `journalctl -u boardingpass.service`
- JSON logs are preserved as-is in journald

**Development/Debug Mode**:
```go
// For local development, add human-readable logging option
if os.Getenv("BP_LOG_FORMAT") == "human" {
    fmt.Fprintf(output, "[%s] %s: %s %+v\n",
        time.Now().Format(time.RFC3339), level, message, fields)
} else {
    // JSON format (default)
}
```

### Alternatives Considered

| Alternative | Rejected Because |
|-------------|------------------|
| Direct journald socket | More complex; adds dependency-like code; violates simplicity principle |
| go-systemd library | External dependency; violates minimal dependency principle |
| File-based logging | Requires log rotation setup; less integrated with systemd |

---

## 4. Red Hat UBI9 Go Toolset Build Environment

### Decision

Use **Red Hat Universal Base Image 9 (UBI9) Go Toolset** (`registry.access.redhat.com/ubi9/go-toolset`) as the base for the build container.

### Rationale

- **RHEL Compatibility**: UBI9 is designed for RHEL 9+ target environments (primary deployment platform)
- **Enterprise Support**: Red Hat provides security updates and support for UBI images
- **FIPS Compliance**: UBI9 Go toolset is built with FIPS-enabled Go toolchain
- **Free to Use**: UBI images are freely redistributable and usable even in production
- **Consistent Toolchain**: Same environment as RHEL 9+ systems where service will run

### Containerfile Strategy

**`build/Containerfile`**:
```dockerfile
# Use Red Hat UBI9 Go Toolset
FROM registry.access.redhat.com/ubi9/go-toolset:1.23 AS builder

# Switch to root for tool installation
USER root

# Install golangci-lint 2.7.1
RUN curl -sSfL https://github.com/golangci/golangci-lint/releases/download/v2.7.1/golangci-lint-2.7.1-linux-amd64.tar.gz | \
    tar -xz -C /usr/local/bin --strip-components=1 golangci-lint-2.7.1-linux-amd64/golangci-lint

# Install GoReleaser
RUN curl -sSfL https://github.com/goreleaser/goreleaser/releases/download/v1.23.0/goreleaser_Linux_x86_64.tar.gz | \
    tar -xz -C /usr/local/bin goreleaser

# Install additional build tools
RUN dnf install -y make rpm-build && dnf clean all

# Switch back to default user
USER default

WORKDIR /workspace

# Copy dependency files
COPY --chown=default:root go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY --chown=default:root . .

# Default command: build
CMD ["goreleaser", "build", "--snapshot", "--clean", "--single-target"]
```

**Version Consistency**:
- **Go Version**: 1.23 (latest stable at time of writing; update as needed)
- **golangci-lint**: v2.7.1 (explicitly specified)
- **Both build container and GitHub Actions use identical versions**

**Local Development Workflow**:
```bash
# Build the builder image
podman build -t boardingpass-builder -f build/Containerfile .

# Run build inside container
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder goreleaser build --snapshot --clean

# Run tests inside container
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder go test ./...

# Run linters
podman run --rm -v $(pwd):/workspace:Z boardingpass-builder golangci-lint run
```

**Makefile Wrapper** (for convenience):
```makefile
.PHONY: build test lint coverage

PODMAN := podman
BUILDER_IMAGE := boardingpass-builder

build-image:
	$(PODMAN) build -t $(BUILDER_IMAGE) -f build/Containerfile .

build: build-image
	$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) goreleaser build --snapshot --clean

test: build-image
	$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) go test -v -race -cover ./...

lint: build-image
	$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) golangci-lint run --verbose

coverage: build-image
	$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) \
		go test -v -race -coverprofile=_output/coverage/coverage.out -covermode=atomic ./...
	$(PODMAN) run --rm -v $(PWD):/workspace:Z $(BUILDER_IMAGE) \
		go tool cover -html=_output/coverage/coverage.out -o _output/coverage/coverage.html
```

### Alternatives Considered

| Alternative | Rejected Because |
|-------------|------------------|
| Alpine + golang image | Not FIPS-compliant; different from target RHEL environment |
| Debian + golang image | Not FIPS-compliant; different from target RHEL environment |
| Custom RHEL base | UBI9 provides same capabilities without requiring RHEL subscription |

---

## 5. GitHub Actions CI/CD Workflows

### Decision

Implement three GitHub Actions workflows with **identical Go 1.23 and golangci-lint v2.7.1** versions as the build container.

### Service CI Workflow Design

**`.github/workflows/service-ci.yaml`**:
```yaml
name: Service CI

on:
  pull_request:
    branches: [main]
    paths:
      - 'cmd/**'
      - 'internal/**'
      - 'pkg/**'
      - 'go.mod'
      - 'go.sum'
      - '.github/workflows/service-ci.yaml'
      - '.golangci.yml'
  push:
    branches: [main]

permissions:
  contents: read
  pull-requests: write

env:
  GO_VERSION: "1.23"
  GOLANGCI_LINT_VERSION: "v2.7.1"

jobs:
  lint:
    name: Lint & Security Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: ${{ env.GOLANGCI_LINT_VERSION }}
          args: --verbose --timeout 5m

  test:
    name: Test & Coverage
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true

      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total: | awk '{print $3}' | sed 's/%//')
          echo "Coverage: ${COVERAGE}%"
          if (( $(echo "$COVERAGE < 80" | bc -l) )); then
            echo "ERROR: Coverage ${COVERAGE}% is below 80% threshold"
            exit 1
          fi

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
          flags: unittests

  build:
    name: Build (Cross-Compile)
    runs-on: ubuntu-latest
    needs: [lint, test]
    container:
      image: registry.access.redhat.com/ubi9/go-toolset:1.23
    steps:
      - uses: actions/checkout@v4

      - name: Install GoReleaser
        run: |
          curl -sSfL https://github.com/goreleaser/goreleaser/releases/download/v1.23.0/goreleaser_Linux_x86_64.tar.gz | \
          tar -xz -C /usr/local/bin goreleaser

      - name: Run GoReleaser (build only)
        run: goreleaser build --snapshot --clean

      - name: Check binary size
        run: |
          SIZE=$(stat -c%s _output/dist/boardingpass_linux_amd64/boardingpass)
          SIZE_MB=$(echo "scale=2; $SIZE / 1048576" | bc)
          echo "Binary size: ${SIZE_MB} MB"
          if (( $(echo "$SIZE_MB > 10" | bc -l) )); then
            echo "ERROR: Binary size ${SIZE_MB} MB exceeds 10 MB limit"
            exit 1
          fi

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: boardingpass-binaries
          path: _output/dist/**/boardingpass*
```

**`.golangci.yml` Configuration**:
```yaml
run:
  timeout: 5m
  modules-download-mode: readonly
  go: "1.23"

linters:
  enable:
    - gosec          # Security analysis
    - govulncheck    # Vulnerability scanning
    - errcheck       # Unchecked errors
    - govet          # Standard Go vet
    - staticcheck    # Static analysis
    - unused         # Unused code
    - ineffassign    # Ineffective assignments
    - misspell       # Spelling
    - gofmt          # Formatting
    - goimports      # Import organization
    - revive         # Linting
    - gocritic       # Opinionated linter

linters-settings:
  gosec:
    severity: medium
    confidence: medium
    excludes:
      - G104  # Audit errors not checked (covered by errcheck)
    includes:
      - G101  # Look for hard coded credentials
      - G102  # Bind to all interfaces
      - G201  # SQL injection
      - G401  # Weak crypto (MD5, SHA1, DES)
      - G402  # TLS InsecureSkipVerify
      - G403  # RSA key size < 2048
      - G404  # Weak random number generator

  govulncheck:
    go: "1.23"
    check: symbol  # Check reachable vulnerable symbols

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
  exclude-use-default: false
```

### Release Workflow Design

**`.github/workflows/release.yaml`**:
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write
  packages: write

env:
  GO_VERSION: "1.23"

jobs:
  release:
    runs-on: ubuntu-latest
    container:
      image: registry.access.redhat.com/ubi9/go-toolset:1.23
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install GoReleaser
        run: |
          curl -sSfL https://github.com/goreleaser/goreleaser/releases/download/v1.23.0/goreleaser_Linux_x86_64.tar.gz | \
          tar -xz -C /usr/local/bin goreleaser

      - name: Run GoReleaser
        run: goreleaser release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Upload RPM to release
        uses: softprops/action-gh-release@v1
        with:
          files: _output/dist/*.rpm

      - name: Upload DEB to release
        uses: softprops/action-gh-release@v1
        with:
          files: _output/dist/*.deb
```

### Version Consistency Matrix

| Environment | Go Version | golangci-lint Version | Source |
|-------------|------------|----------------------|--------|
| Build Container (Containerfile) | 1.23 | v2.7.1 | UBI9 Go Toolset base image |
| GitHub Actions (service-ci.yaml) | 1.23 | v2.7.1 | env.GO_VERSION, env.GOLANGCI_LINT_VERSION |
| GitHub Actions (release.yaml) | 1.23 | N/A (GoReleaser only) | Container image |
| golangci.yml config | 1.23 | N/A | run.go, govulncheck.go |

---

## 6. TLS 1.3 Enforcement

### Decision

Enforce **TLS 1.3 minimum** with FIPS-approved cipher suites using Go's `crypto/tls` configuration.

### Implementation

```go
// internal/api/server.go

func NewTLSConfig(certFile, keyFile string) (*tls.Config, error) {
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
    }

    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS13,  // TLS 1.3 minimum
        MaxVersion:   tls.VersionTLS13,  // TLS 1.3 only
        CipherSuites: []uint16{
            tls.TLS_AES_128_GCM_SHA256,  // FIPS-approved
            tls.TLS_AES_256_GCM_SHA384,  // FIPS-approved
        },
        CurvePreferences: []tls.CurveID{
            tls.CurveP256,  // FIPS-approved
            tls.CurveP384,  // FIPS-approved
        },
        PreferServerCipherSuites: true,
    }, nil
}
```

**Certificate Management** (clarified 2025-12-06):
- **Auto-generation**: Self-signed certificates generated at first boot if not provided in OS image
- **Storage**: `/var/lib/boardingpass/tls/server.crt` and `/var/lib/boardingpass/tls/server.key`
- **Permissions**: `0600` (owner read/write only)
- **Optional**: Certificates can be pre-provisioned in bootc image to skip generation

---

## 7. Session Token Management

### Decision

Use **HMAC-signed tokens** stored in-memory with **30-minute TTL** (clarified 2025-12-06), no persistence.

### Implementation

```go
// internal/auth/session.go

type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string]*Session
    secret   []byte
    ttl      time.Duration
}

type Session struct {
    Token     string
    Username  string
    CreatedAt time.Time
    ExpiresAt time.Time
}

func NewSessionManager(secret []byte, ttl time.Duration) *SessionManager {
    sm := &SessionManager{
        sessions: make(map[string]*Session),
        secret:   secret,
        ttl:      ttl,
    }
    go sm.cleanupExpiredSessions()
    return sm
}

func (sm *SessionManager) CreateSession(username string) (string, error) {
    // Generate high-entropy token
    tokenBytes := make([]byte, 32)
    if _, err := rand.Read(tokenBytes); err != nil {
        return "", err
    }
    tokenID := base64.URLEncoding.EncodeToString(tokenBytes)

    // Sign token with HMAC
    h := hmac.New(sha256.New, sm.secret)
    h.Write([]byte(tokenID + username))
    signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

    token := tokenID + "." + signature

    now := time.Now()
    session := &Session{
        Token:     token,
        Username:  username,
        CreatedAt: now,
        ExpiresAt: now.Add(sm.ttl),
    }

    sm.mu.Lock()
    defer sm.mu.Unlock()

    // Limit concurrent sessions (defense against exhaustion)
    if len(sm.sessions) >= 10 {
        return "", errors.New("session limit exceeded")
    }

    sm.sessions[token] = session
    return token, nil
}

func (sm *SessionManager) ValidateSession(token string) (*Session, error) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    session, exists := sm.sessions[token]
    if !exists {
        return nil, errors.New("invalid session token")
    }

    if time.Now().After(session.ExpiresAt) {
        return nil, errors.New("session expired")
    }

    return session, nil
}

func (sm *SessionManager) cleanupExpiredSessions() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        sm.mu.Lock()
        now := time.Now()
        for token, session := range sm.sessions {
            if now.After(session.ExpiresAt) {
                delete(sm.sessions, token)
            }
        }
        sm.mu.Unlock()
    }
}
```

---

## 7a. Brute Force Protection (Clarified 2025-12-06)

### Decision

Implement **progressive delay** rate limiting: 1s after 1st failure, 2s after 2nd, 5s after 3rd, then 60s lockout.

### Rationale

- Balances security (thwarts brute force attacks) with usability (allows legitimate users to retry)
- Progressive delays discourage automated attacks without completely locking out devices
- IP-based tracking prevents single attacker from exhausting attempts

### Implementation

```go
// internal/auth/ratelimit.go

type RateLimiter struct {
    mu       sync.RWMutex
    attempts map[string]*AttemptTracker  // key: client IP
}

type AttemptTracker struct {
    Count      int
    LastFailed time.Time
    LockedUntil time.Time
}

func (rl *RateLimiter) RecordFailure(clientIP string) time.Duration {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    tracker, exists := rl.attempts[clientIP]
    if !exists {
        tracker = &AttemptTracker{}
        rl.attempts[clientIP] = tracker
    }

    tracker.Count++
    tracker.LastFailed = time.Now()

    // Progressive delays
    var delay time.Duration
    switch tracker.Count {
    case 1:
        delay = 1 * time.Second
    case 2:
        delay = 2 * time.Second
    case 3:
        delay = 5 * time.Second
    default:
        // 4+ failures: 60-second lockout
        tracker.LockedUntil = time.Now().Add(60 * time.Second)
        delay = 60 * time.Second
    }

    return delay
}

func (rl *RateLimiter) IsLocked(clientIP string) (bool, time.Duration) {
    rl.mu.RLock()
    defer rl.mu.RUnlock()

    tracker, exists := rl.attempts[clientIP]
    if !exists {
        return false, 0
    }

    if time.Now().Before(tracker.LockedUntil) {
        return true, time.Until(tracker.LockedUntil)
    }

    return false, 0
}

func (rl *RateLimiter) RecordSuccess(clientIP string) {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    delete(rl.attempts, clientIP)  // Clear on successful auth
}
```

**Response codes**:
- 1st-3rd failures: HTTP 401 Unauthorized + `Retry-After` header
- 4+ failures: HTTP 429 Too Many Requests + `Retry-After: 60`

---

## 7b. Configuration Path Validation (Clarified 2025-12-06)

### Decision

Validate configuration bundle file paths against **configurable allow-list** of permitted subdirectories to prevent writes to critical system files.

### Rationale

- Prevents accidental/malicious overwrites of `/etc/passwd`, `/etc/shadow`, `/etc/sudoers`, etc.
- Allow-list approach (explicit permission) is safer than block-list (explicit denial)
- Configurability allows deployment-specific customization

### Implementation

```go
// internal/provisioning/pathvalidator.go

type PathValidator struct {
    allowedPaths []string  // e.g., ["/etc/systemd/", "/etc/myapp/"]
}

func NewPathValidator(allowedPaths []string) *PathValidator {
    return &PathValidator{allowedPaths: allowedPaths}
}

func (pv *PathValidator) ValidatePath(path string) error {
    // Normalize path
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("invalid path: %w", err)
    }

    // Check against allow-list
    for _, allowed := range pv.allowedPaths {
        if strings.HasPrefix(absPath, allowed) {
            return nil  // Path is allowed
        }
    }

    return fmt.Errorf("path %s not in allow-list", absPath)
}
```

**Service Configuration** (`/etc/boardingpass/config.yaml`):
```yaml
provisioning:
  allowed_paths:
    - /etc/systemd/system/
    - /etc/NetworkManager/system-connections/
    - /etc/myapp/
```

**Error response**:
- HTTP 400 Bad Request
- Body: `{"error": "path /etc/passwd not in allow-list"}`

---

## 8. Atomic File Operations

### Decision

Use **temp-write → validate → atomic rename** pattern for configuration file application.

### Implementation

```go
// internal/provisioning/applier.go

func ApplyConfigBundle(bundle ConfigBundle) error {
    // Create temporary directory
    tempDir, err := os.MkdirTemp("/var/lib/boardingpass/staging", "apply-*")
    if err != nil {
        return fmt.Errorf("failed to create temp dir: %w", err)
    }
    defer os.RemoveAll(tempDir)

    // Write all files to temp directory first
    for _, file := range bundle.Files {
        tempPath := filepath.Join(tempDir, filepath.Base(file.Path))
        if err := os.WriteFile(tempPath, file.Content, file.Mode); err != nil {
            return fmt.Errorf("failed to write temp file %s: %w", file.Path, err)
        }
    }

    // Validate all files
    for _, file := range bundle.Files {
        tempPath := filepath.Join(tempDir, filepath.Base(file.Path))
        if err := validateFile(tempPath); err != nil {
            return fmt.Errorf("validation failed for %s: %w", file.Path, err)
        }
    }

    // Atomic rename all files to target paths
    for _, file := range bundle.Files {
        tempPath := filepath.Join(tempDir, filepath.Base(file.Path))
        targetPath := filepath.Join("/etc", file.Path)

        // Ensure target directory exists
        if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
            return fmt.Errorf("failed to create target dir: %w", err)
        }

        // Atomic rename (os.Rename is atomic on same filesystem)
        if err := os.Rename(tempPath, targetPath); err != nil {
            return fmt.Errorf("failed to move file %s: %w", file.Path, err)
        }
    }

    return nil
}
```

---

## 9. Systemd Service Configuration

### Decision

Implement **systemd unit file** with `ConditionPathExists` for sentinel file checking.

### Implementation

**`build/boardingpass.service`**:
```ini
[Unit]
Description=BoardingPass Bootstrap Service
Documentation=https://github.com/fzdarsky/boardingpass
After=network-online.target
Wants=network-online.target
ConditionPathExists=!/etc/boardingpass/issued

[Service]
Type=notify
User=boardingpass
Group=boardingpass
ExecStart=/usr/bin/boardingpass --config=/etc/boardingpass/config.yaml
StandardOutput=journal
StandardError=journal
SyslogIdentifier=boardingpass
Restart=on-failure
RestartSec=5s
TimeoutStopSec=10s

# Security hardening
PrivateTmp=yes
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/lib/boardingpass
ReadOnlyPaths=/etc/boardingpass

# Resource limits
LimitNOFILE=1024
MemoryLimit=150M

[Install]
WantedBy=multi-user.target
```

---

## Summary of Key Decisions

| Area | Decision | Rationale |
|------|----------|-----------|
| **SRP Password** | Dynamic via generator script | Device-unique passwords from serial #/TPM/MAC |
| **Logging** | stdout/stderr → systemd journal | 12-factor app principle, simpler implementation |
| **Build Base** | UBI9 Go Toolset 1.23 | RHEL 9+ compatibility, FIPS compliance |
| **Tool Versions** | Go 1.23, golangci-lint v2.7.1 | Consistent across build container and CI |
| **Build** | GoReleaser → `_output/dist/` | Single tool, reproducible builds |
| **TLS** | TLS 1.3 only, FIPS ciphers | Security baseline, FIPS 140-3 compliance |
| **Sessions** | HMAC-signed in-memory tokens | No persistence, TTL expiration |
| **Files** | Atomic rename pattern | Fail-safe configuration application |
| **Systemd** | ConditionPathExists sentinel | Ephemeral lifecycle enforcement |

---

## Next Steps

Phase 1 design will produce:
1. **data-model.md**: Entity definitions and JSON schemas
2. **contracts/openapi.yaml**: Complete OpenAPI 3.1 specification
3. **quickstart.md**: Local development setup guide

All implementation details documented here will inform Phase 1 deliverables.
