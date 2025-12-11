# Implementation Plan: Boarding CLI Tool

**Branch**: `002-boarding-cli` | **Date**: 2025-12-10 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-boarding-cli/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement a CLI tool (`boarding`) that enables developers and CI systems to interact with the BoardingPass service for device provisioning. The CLI provides six commands (pass, info, connections, load, command, complete) that handle SRP-6a authentication, session management, system information querying, configuration upload, and command execution. The tool must follow the project's minimal dependencies philosophy, using only Go stdlib + gopkg.in/yaml.v3, and maintain FIPS 140-3 compliance by avoiding third-party cryptographic libraries.

## Technical Context

**Language/Version**: Go 1.25+ (CGO_ENABLED=0 for static linking)
**Primary Dependencies**: gopkg.in/yaml.v3 (config file parsing) - NO other external runtime dependencies
**Storage**: Filesystem-based (session tokens in OS temp dir with 0600 permissions, config in OS config dir, cert fingerprints in OS config dir)
**Testing**: `go test` with table-driven tests, `testify/assert` (test-only dependency), `httptest` for mock servers
**Target Platform**: Cross-platform (Linux,  macOS, Windows) - uses `os.UserConfigDir()` and `os.UserCacheDir()` for OS-specific paths
**Project Type**: Single project (modular monolith) - adding new CLI binary to existing BoardingPass codebase
**Performance Goals**: Authenticate in <5s (SC-001), query info in <2s (SC-002), upload 100 files/10MB in <30s (SC-004)
**Constraints**: Minimal footprint (<10MB binary), no new runtime dependencies, FIPS 140-3 compliance (stdlib crypto/* only), must NOT use external CLI frameworks (no cobra, no urfave/cli)
**Scale/Scope**: Small CLI tool (~2000-2500 lines new code), 6 commands, supports multiple BoardingPass service instances via per-server session tokens

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Frictionless Bootstrapping
✅ **PASS** - CLI simplifies device provisioning by providing a command-line interface that can be automated in CI/CD pipelines. Supports both interactive (developer) and non-interactive (CI) modes via flags and environment variables.

### II. Ephemeral & Fail-Safe Operation
✅ **PASS** - CLI is stateless (does not run as daemon). Session tokens are ephemeral (stored in OS temp dirs, cleared on system reboot). All operations either succeed or fail with clear error messages and non-zero exit codes.

### III. Minimal Footprint
✅ **PASS** - Static binary built with CGO_ENABLED=0. Estimated binary size: 8-10MB (comparable to existing boardingpass binary). No runtime dependencies beyond Go stdlib + gopkg.in/yaml.v3 (already present).

### IV. Minimal Dependencies
✅ **PASS** - Uses ONLY Go stdlib + gopkg.in/yaml.v3 (already a project dependency). Explicitly REJECTS external CLI frameworks (cobra, urfave/cli) in favor of stdlib `flag` package. No new runtime dependencies added.

### V. Transport Agnostic & Protocol First
✅ **PASS** - CLI is a pure HTTP/S client that consumes the existing BoardingPass REST API. Reuses `pkg/protocol/` types for API contracts. Does not introduce transport-specific logic.

### VI. Open Source & Permissive Licensing
✅ **PASS** - CLI will be part of the BoardingPass open source project (same license). No dependencies with restrictive licenses.

### Security Requirements
✅ **PASS** - Implements SRP-6a client-side authentication (stdlib crypto), enforces TLS 1.3+, stores session tokens with 0600 permissions, validates all inputs, redacts secrets from logs, uses Trust-on-First-Use (TOFU) for TLS certificates with fingerprint verification.

**Overall Assessment**: ✅ ALL GATES PASS - No constitution violations. Implementation aligns with all core principles.

## Project Structure

### Documentation (this feature)

```text
specs/002-boarding-cli/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (SRP client design, config precedence, TLS TOFU)
├── data-model.md        # Phase 1 output (Config, Session Token, CertFingerprint entities)
├── quickstart.md        # Phase 1 output (CLI installation and usage guide)
├── contracts/           # Phase 1 output (CLI does NOT define new contracts - consumes existing API)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/boarding/              # NEW: CLI binary entry point
  main.go                  # Command routing, flag parsing, error handling

internal/cli/              # NEW: CLI-specific internal packages
  client/                  # HTTP client for BoardingPass API
    client.go              # Main HTTP client with TLS config, session injection
    srp.go                 # SRP-6a client-side protocol implementation
    transport.go           # Custom RoundTripper for cert fingerprinting
  commands/                # CLI command implementations
    root.go                # Common command infrastructure
    pass.go                # US1: Authentication command
    info.go                # US2: System info query command
    connections.go         # US3: Network query command
    load.go                # US4: Configuration upload command
    command.go             # US5: Command execution
    complete.go            # US6: Session termination
  config/                  # Configuration management
    config.go              # Config loading with precedence (flags > env > file)
    dirs.go                # OS-specific directory helpers (UserConfigDir, UserCacheDir)
  session/                 # Session token persistence
    store.go               # Token file I/O with 0600 permissions
  tls/                     # TLS certificate handling
    fingerprint.go         # Cert fingerprint computation (SHA-256)
    store.go               # Known certificates storage (YAML)
    prompt.go              # Interactive cert acceptance prompts
  output/                  # Output formatting
    formatter.go           # YAML and JSON formatters

pkg/protocol/              # EXISTING: Reused for API request/response types
  types.go                 # Request/response structs, error codes

internal/auth/             # EXISTING: Reference for SRP parameters
  srp.go                   # Server-side SRP (client will mirror this logic)

tests/cli-integration/     # NEW: CLI integration tests
  client_test.go           # HTTP client tests with mock server
  auth_test.go             # SRP authentication flow tests
  commands_test.go         # Command tests with mock BoardingPass API

tests/cli-e2e/             # NEW: CLI end-to-end tests
  workflow_test.go         # Full provisioning workflow (pass → info → load → complete)
```

**Structure Decision**: Single project (Option 1) with new `cmd/boarding/` binary and `internal/cli/` packages. This follows the existing BoardingPass project structure where `cmd/` contains service binaries and `internal/` contains private implementation packages. The CLI reuses `pkg/protocol/` for API contracts and references `internal/auth/srp.go` for SRP-6a group parameters.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

_No violations detected. All gates pass._

