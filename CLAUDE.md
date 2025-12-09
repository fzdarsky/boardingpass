# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BoardingPass is a lightweight, ephemeral bootstrap service for headless Linux devices, exposing a RESTful API over HTTPS with SRP-6a authentication. It enables secure device inventory querying, atomic configuration provisioning, and execution of allow-listed system commands.

## Build & Development Commands

```bash
make build      # Build binary to _output/bin/boardingpass
make test       # Run all tests with race detection
make lint       # Run golangci-lint (v2 config, includes gosec)
make coverage   # Generate coverage report to _output/coverage/
make clean      # Remove _output/ directory
make deps       # Download and verify dependencies
```

The project uses Go 1.25+ with CGO disabled for static linking. GoReleaser handles cross-compilation and packaging (outputs to `_output/dist/`).

## Architecture & Structure

### General Design Principles
- **Pattern:** Modular Monolith.
- **KISS (Keep It Simple, Stupid)**: Strive for simplicity in design and implementation.
- **YAGNI (You Ain't Gonna Need It)**: Avoid adding features or complexity until absolutely necessary.
- **Clear is better than clever**: Readability is paramount. Avoid "magic" code.
- **No premature abstractions**: Introduce abstractions only when there are at least three different abstracted cases.
- **Reactive Interfaces:** Define interfaces only when mocking is needed. Define them at the consumer. "Accept interfaces, return structs."
- **Minimize dependencies:** Prefer Go stdlib. Justify any third-party libraries; review them for security and maintenance. It is better to copy a small snippet of code than to import a massive library for one function.
- **Dependency Injection:** Use explicit injection via constructors (e.g., `NewService(repo Repository)`). Avoid global state.
- **Curate configuration parameters:** Exposing too many parameters increases user error and maintenance burden. Only expose what is necessary. Prefer high-level parameters.
- **Convention over Configuration**: Decrease the number of decisions users need to make by choosing for them (e.g. naming/location of configuration files), providing sensible defaults, and supporting auto-detection.
- **Configuration files are APIs**: Changes to config file structure and naming are breaking API changes. Maintain backward compatibility where possible.
- **Naming Conventions:** Use clear, concise, and consistent names. Names should intuitively describe purpose/behavior (e.g. not everything is a "...Type", but maybe a "...Method", "...Policy", or "...Strategy").

### Project Design Principles (from Constitution)

1. **Ephemeral Operation**: Service terminates after provisioning via sentinel file
2. **Transport Agnostic**: RESTful API over HTTPS, protocol-first design
3. **Minimal Dependencies**: Go stdlib preferred; `gopkg.in/yaml.v3` is the only allowed external runtime dependency.
4. **Static Linking**: CGO_ENABLED=0, single binary < 10MB
5. **FIPS 140-3 Compliance**: Use only Go stdlib `crypto/*`— absolutely no third-party crypto libraries.

### Source Layout

- `cmd/boardingpass/` - Main service binary entry point
- `internal/` - Private packages (not importable externally):
  - `api/` - HTTP handlers, middleware (auth, logging, errors), server lifecycle
  - `auth/` - SRP-6a implementation, session tokens (30-min TTL), rate limiting
  - `tls/` - Self-signed cert generation, TLS 1.3+ config
  - `inventory/` - System info extraction (TPM, board, CPU, OS, FIPS)
  - `network/` - Interface enumeration, link state, IP addresses
  - `provisioning/` - Config bundle parsing, path allow-list validation, atomic file ops
  - `command/` - Allow-listed command execution via sudo
  - `lifecycle/` - Sentinel file, inactivity timeout, graceful shutdown
  - `config/` - YAML config loading and validation
  - `logging/` - JSON logging to stdout/stderr with secret redaction
- `pkg/protocol/` - Shared types for future mobile app integration
- `tests/` - Unit, integration, contract, and e2e tests
- `build/` - systemd unit file, sudoers config, Containerfile

### Key Technical Details

- **Authentication**: SRP-6a with device-unique passwords (generated from hardware IDs)
- **Brute Force Protection**: Progressive delays (1s → 2s → 5s → 60s lockout)
- **Session Tokens**: 30-minute TTL, in-memory storage
- **Config Provisioning**: Atomic ops (temp → validate → rename), path allow-list enforcement
- **Logging**: JSON to stdout/stderr (systemd captures), secrets always redacted

## Go Coding Standards

- **Layout:** Standard Go Layout.
- **Errors:**
  - Use guard clauses to handle errors and edge cases early; avoid else blocks where possible.
  - Always wrap errors with context: `fmt.Errorf("failed to process item %s: %w", id, err)`.
- **Testing:**
  - Use Table-Driven Tests for logic.
  - Use `testify/assert` for readability (test-only dependency).
  - Mock external dependencies using interfaces generated by `uber-go/mock`.
- **Naming:**
  - Short, descriptive names (`c` for client, not `myClient`).
  - Exported functions must have comments.
- **Context:** Always propagate `context.Context` as the first argument in async/I-O functions.
- **Modern:** Use Go 1.25+ features (e.g., `errors.Join`, `slices`, `maps` packages, use `any` instead of `interface{}`).

## Specification Workflow

This project uses the SpecKit workflow. Specifications live in `specs/001-boardingpass-api/`:

- `spec.md` - Feature specification with user stories and requirements
- `plan.md` - Implementation plan with architecture decisions
- `tasks.md` - Ordered task list organized by user story
- `contracts/openapi.yaml` - OpenAPI 3.1 specification

Available slash commands: `/speckit.specify`, `/speckit.plan`, `/speckit.tasks`, `/speckit.implement`

## Linting

Always run linters and fix all issues before committing:

```bash
make lint
```

## Testing

Tests follow standard Go conventions:
- Unit tests are co-located with source files (e.g., `internal/auth/session_test.go` tests `internal/auth/session.go`)
- `tests/integration/` - API endpoint tests using httptest
- `tests/contract/` - OpenAPI spec validation
- `tests/e2e/` - Containerized systemd environment tests

Run tests:
- All tests: `make test`
- Single package: `go test -v ./internal/auth`
- Single test: `go test -v -run TestName ./internal/auth`

## Configuration Files

- `.golangci.yaml` - Linter config (v2 format) with gosec, staticcheck, revive
- `.goreleaser.yaml` - Build orchestration for Linux amd64/arm64, RPM/DEB packaging
- `build/boardingpass.service` - systemd unit with sentinel file check
- `build/boardingpass.sudoers` - Restricted sudo permissions for commands
