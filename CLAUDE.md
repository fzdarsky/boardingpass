# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BoardingPass is a lightweight, ephemeral bootstrap service for headless Linux devices, exposing a RESTful API over HTTPS with SRP-6a authentication. It enables secure device inventory querying, atomic configuration provisioning, and execution of allow-listed system commands.

## Build & Development Commands

```bash
make build      # Build binary to _output/bin/boardingpass
make test       # Run all tests with race detection
make lint       # Run golangci-lint (v2 config, includes gosec)
make generate   # Regenerate all generated code (mocks)
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

## Generating

After changing mocked interfaces and before committing, run `make generate` to update mocks.

## Linting

After completing work on a task or list of tasks, **always** run `make lint` and fix **all** errors.

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

After completing work on a task or list of tasks, **always** run `make test` and fix **all** errors.

## Configuration Files

- `.golangci.yaml` - Linter config (v2 format) with gosec, staticcheck, revive
- `.goreleaser.yaml` - Build orchestration for Linux amd64/arm64, RPM/DEB packaging
- `build/boardingpass.service` - systemd unit with sentinel file check
- `build/boardingpass.sudoers` - Restricted sudo permissions for commands

## Active Technologies

- Go 1.25+ (BoardingPass service)
- TypeScript 5.x with React Native 0.74+, targeting ES2022 (Mobile onboarding app)

## Mobile App Development Workflow

The `mobile/` directory contains a React Native mobile application for discovering and onboarding BoardingPass devices.

### Quick Start

```bash
cd mobile
npm install                    # Install dependencies
npm run generate:types         # Generate TypeScript types from OpenAPI spec
npx expo prebuild              # Generate ios/ and android/ directories
npm start                      # Start Metro bundler
npm run ios                    # Run on iOS simulator
npm run android                # Run on Android emulator
```

### Project Structure

```
mobile/
├── app/                      # Expo Router screens (file-based routing)
│   ├── index.tsx            # Device discovery screen
│   ├── device/[id].tsx      # Device detail screen
│   └── device/authenticate.tsx  # Authentication screen
├── src/
│   ├── components/          # Reusable UI components
│   ├── services/            # Business logic (discovery, auth, API, certificates)
│   ├── hooks/               # Custom React hooks
│   ├── contexts/            # React Context providers
│   ├── types/               # TypeScript type definitions
│   └── utils/               # Utility functions
└── tests/                   # Unit, integration, contract, and E2E tests
```

### Development Commands

```bash
# Development
npm start                    # Start Metro bundler
npm run ios                  # Run on iOS
npm run android              # Run on Android
npm run web                  # Run in browser (limited functionality)

# Type Generation
npm run generate:types       # Generate types from OpenAPI spec
npm run validate:spec        # Validate OpenAPI specification

# Testing
npm test                     # Run unit tests
npm run test:watch          # Run tests in watch mode
npm run test:coverage       # Run tests with coverage report
npm run test:integration    # Run integration tests
npm run e2e:test:ios        # Run E2E tests on iOS
npm run e2e:test:android    # Run E2E tests on Android

# Code Quality
npm run lint                 # Lint code
npm run lint:fix             # Lint and auto-fix issues
npm run format               # Format code with Prettier
npm run typecheck            # TypeScript type checking

# Building
npx expo prebuild            # Generate native projects
eas build --platform ios     # Build iOS via EAS
eas build --platform android # Build Android via EAS
```

### Mobile Coding Standards

- **Framework**: React Native with Expo managed workflow
- **UI Library**: React Native Paper (Material Design 3)
- **Navigation**: Expo Router (file-based routing)
- **State Management**: React Context API + custom hooks
- **Styling**: StyleSheet API (no styled-components or CSS-in-JS libraries)
- **Testing**: Jest + React Native Testing Library + Detox (E2E)
- **Type Safety**: Strict TypeScript mode enabled

**Component Structure:**

- Keep components small and focused (single responsibility)
- Extract reusable components to `src/components/`
- Colocate component-specific files (styles, tests, utils)
- Use custom hooks for complex stateful logic

**State Management:**

- Use React Context for global state (DeviceContext, AuthContext, CertificateContext)
- Use local state (useState) for component-specific state
- Use custom hooks to encapsulate business logic
- Avoid prop drilling - use Context when passing state through 3+ levels

**Security Requirements:**

- NEVER log sensitive data (connection codes, session tokens, SRP values)
- Clear sensitive data from memory after use (FR-036)
- Use expo-secure-store for persistent sensitive data (tokens, certificate pins)
- Implement certificate pinning for self-signed TLS certificates
- Validate all user inputs before submission

**Performance Guidelines:**

- Use React.memo() for expensive components
- Optimize FlatList with proper keyExtractor and getItemLayout
- Lazy load heavy components
- Keep UI rendering at 60 FPS
- Monitor bundle size (target < 50MB)

**Accessibility:**

- Add accessibilityLabel to all interactive elements
- Support screen readers (VoiceOver/TalkBack)
- Ensure sufficient color contrast (WCAG AA)
- Test with accessibility tools enabled

### FIPS Compatibility (Critical)

The mobile app MUST use FIPS 140-3 compatible parameters for SRP-6a authentication to interoperate with the BoardingPass service:

- **Hash Algorithm**: SHA-256 (FIPS 180-4 approved)
- **SRP Group**: RFC 5054 2048-bit safe prime
- **Generator**: g = 2

Verify SRP configuration in `mobile/src/services/auth/srp.ts` before committing authentication changes.

### API Integration

The mobile app consumes the BoardingPass RESTful API defined in `specs/001-boardingpass-api/contracts/openapi.yaml`.

**Type Generation Workflow:**

1. Update OpenAPI spec in `specs/001-boardingpass-api/contracts/openapi.yaml`
2. Run `npm run generate:types` in `mobile/` directory
3. Types are generated to `mobile/src/types/api.ts`
4. Never manually edit generated types

**API Client Pattern:**

- Use `mobile/src/services/api/client.ts` for HTTP requests
- Implement service-specific modules (e.g., `info.ts`, `network.ts`)
- Always include authentication token in requests
- Handle certificate pinning for self-signed certificates
- Implement proper error handling and retry logic

### Testing Strategy

**Unit Tests** (`mobile/tests/unit/`):

- Test utilities, hooks, and business logic
- Mock external dependencies
- Use Jest + React Native Testing Library
- Target 80%+ code coverage

**Integration Tests** (`mobile/tests/integration/`):

- Test authentication flows
- Test device discovery
- Test API integration
- Use real API responses (mock server)

**Contract Tests** (`mobile/tests/contract/`):

- Validate API responses against OpenAPI spec
- Ensure mobile app and service are compatible
- Run before releases

**E2E Tests** (`mobile/tests/e2e/`):

- Test complete user flows
- Use Detox for iOS and Android
- Test on physical devices before release

### Common Tasks

**Adding a New Screen:**

1. Create screen file in `app/` directory (e.g., `app/settings.tsx`)
2. Expo Router automatically generates route
3. Add navigation in existing screens using `router.push('/settings')`

**Adding a New Component:**

1. Create component directory in `src/components/` (e.g., `src/components/Button/`)
2. Add `index.tsx` with component implementation
3. Export from component directory
4. Write unit tests in `mobile/tests/unit/components/`

**Adding a New Hook:**

1. Create hook file in `src/hooks/` (e.g., `src/hooks/useDeviceInfo.ts`)
2. Implement hook following React hooks rules
3. Write unit tests in `mobile/tests/unit/hooks/`

**Adding a New API Endpoint:**

1. Update OpenAPI spec in `specs/001-boardingpass-api/contracts/openapi.yaml`
2. Run `npm run generate:types` to update types
3. Create service module in `src/services/api/` if needed
4. Write integration tests

### Mobile App Specifications

Detailed specifications for the mobile app are in `specs/003-mobile-onboarding-app/`:

- `spec.md` - Feature specification and user stories
- `plan.md` - Implementation plan and architecture
- `data-model.md` - Entity definitions and relationships
- `research.md` - Technical research and decisions
- `quickstart.md` - Developer quick start guide
- `tasks.md` - Implementation task breakdown

### Certificate Pinning Workflow

The mobile app implements Trust-On-First-Use (TOFU) certificate pinning:

1. **First Connection**: Fetch server certificate, compute SHA-256 fingerprint
2. **User Confirmation**: Display certificate details, request user trust
3. **Pin Storage**: Store fingerprint in secure storage (OS Keychain/Keystore)
4. **Subsequent Connections**: Verify certificate fingerprint matches pinned value
5. **Certificate Change**: Alert user, require re-confirmation

Implementation: `mobile/src/services/certificates/`

### Troubleshooting

**Type errors after API changes:**

```bash
npm run generate:types        # Regenerate types from OpenAPI spec
npm run typecheck             # Verify no type errors
```

**Native module not found:**

```bash
npx expo prebuild --clean     # Regenerate native projects
npm run ios                   # Rebuild app
```

**Tests failing:**

```bash
npm test -- --clearCache      # Clear Jest cache
npm test                      # Re-run tests
```

**Bundle size too large:**

```bash
npx react-native-bundle-visualizer  # Analyze bundle
# Consider code splitting or removing unused dependencies
```

## Recent Changes

- 003-mobile-onboarding-app: Added TypeScript 5.x with React Native 0.74+, targeting ES2022
- 003-mobile-onboarding-app: Implemented device discovery (mDNS + fallback), SRP-6a authentication, certificate pinning, device information display
- 003-mobile-onboarding-app: Added skeleton loading screens, haptic feedback, comprehensive error handling
