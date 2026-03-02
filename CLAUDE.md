# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BoardingPass is a lightweight, ephemeral bootstrap service for headless Linux devices, exposing a RESTful API over HTTPS with SRP-6a authentication. It enables secure device inventory querying, atomic configuration provisioning, and execution of allow-listed system commands.

## Build & Development Commands

BoardingPass uses a consistent Makefile naming scheme across all components: service, CLI, and mobile app.

### Common Commands

```bash
# Quick feedback loop (default target)
make all                  # lint + unit tests + build (all components)

# Component-specific builds
make build-service        # Build service binary to _output/bin/boardingpass
make build-cli            # Build CLI binary to _output/bin/boarding
make build-app-ios        # Generate iOS native project (expo prebuild)
make build-app-android    # Generate Android native project
make build-app            # Generate both iOS and Android projects
make build-all            # Build all components

# Testing
make test-unit-service    # Run service unit tests
make test-unit-app        # Run mobile app unit tests
make test-unit-all        # Run all unit tests
make test-service         # Run all service tests (unit + integration + e2e + contract)
make test-app             # Run all app tests
make test-all             # Run all tests for all components

# Code quality
make lint-service         # Run golangci-lint on service
make lint-app             # Run ESLint + react-doctor on mobile app
make lint-all             # Lint all components

# Code generation
make generate-service     # Regenerate Go mocks
make generate-app         # Regenerate TypeScript types from OpenAPI
make generate-all         # Regenerate all code

# Cleaning
make clean-app            # Clear Metro cache + native projects
make clean-all            # Clean all components
make clean-all-full       # Deep clean including node_modules

# App-specific workflows (troubleshooting)
make clean-cache-app      # Clear Metro cache only
make clean-native-app     # Remove ios/android only
make rebuild-app-ios      # Full rebuild: clean + build + run (iOS)
make fix-app              # Fix common issues (Xcode 26, deps, cache)

# Other
make coverage             # Generate coverage report to _output/coverage/
make deps                 # Download and verify Go dependencies
make release              # Build release packages (RPM, DEB, archives)
make deploy               # Deploy service to bootc container
```

**Pattern**: `{action}-{component}[-{variant}]`

- **Components**: `service`, `cli`, `app`, `all`
- **Actions**: `install-deps`, `generate`, `lint`, `build`, `test-*`, `run`, `clean`
- **Platform variants**: `-ios`, `-android` (for app targets)

Run `make help` to see all available targets.

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

After changing mocked interfaces and before committing, run `make generate-service` to update mocks.
For mobile app type generation, run `make generate-app` to regenerate TypeScript types from OpenAPI spec.

## Linting

After completing work on a task or list of tasks, **always** run `make lint-all` (or component-specific `make lint-service` / `make lint-app`) and fix **all** errors.

## Testing

Tests follow standard Go conventions:

- Unit tests are co-located with source files (e.g., `internal/auth/session_test.go` tests `internal/auth/session.go`)
- `tests/integration/` - API endpoint tests using httptest
- `tests/contract/` - OpenAPI spec validation
- `tests/e2e/` - Containerized systemd environment tests

Run tests:

- All tests: `make test-all` (all components, all test types)
- Service tests: `make test-service` (unit + integration + e2e + contract)
- App tests: `make test-app` (unit + integration + e2e + contract)
- Unit tests only: `make test-unit-all` (fast feedback)
- Single Go package: `go test -v ./internal/auth`
- Single Go test: `go test -v -run TestName ./internal/auth`

After completing work on a task or list of tasks, **always** run `make test-all` (or component-specific tests) and fix **all** errors.

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

Using Makefile (recommended):

```bash
make install-deps-app          # Install dependencies
make generate-app              # Generate TypeScript types from OpenAPI spec
make build-app-ios             # Generate iOS native project
make run-app-ios               # Run on iOS simulator
make run-app-ios-device        # Run on connected physical iOS device
```

Or using npm directly:

```bash
cd mobile
npm install                    # Install dependencies
npm run generate:types         # Generate TypeScript types from OpenAPI spec
npx expo prebuild --platform ios  # Generate ios/ directory
npm run ios                    # Run on iOS simulator
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
npm run ios                  # Run on iOS simulator (iPhone 17 Pro)
npm run ios:device           # Run on connected physical iOS device (prompts for selection)
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

**Xcode 26+ build errors (TARGET_IPHONE_SIMULATOR):**

```bash
# Use Makefile target to fix common issues
make fix-app       # Runs expo install --fix + clean + prebuild
make run-app-ios   # Run on iOS
```

Or manually:

```bash
cd mobile
npx expo install --fix
rm -rf ios && npx expo prebuild --platform ios
npm run ios
```

If errors persist, manually patch `node_modules/expo-dev-menu/ios/DevMenuViewController.swift` to use `#if targetEnvironment(simulator)` instead of `TARGET_IPHONE_SIMULATOR`. See [mobile/README.md](mobile/README.md) for details.

**Running on physical iOS device:**

The default `make run-app-ios` targets a simulator. To run on a connected physical iOS device:

```bash
# First, list available devices to find your device name
make list-ios-devices

# Then run on your physical device (default: "a phone")
make run-app-ios-device

# Or specify a different device name
make run-app-ios-device IOS_PHYSICAL_DEVICE='My iPhone'
```

Or using npm:

```bash
cd mobile
npm run ios:device        # Uses default physical device name
```

The default physical device name is set in the Makefile as `IOS_PHYSICAL_DEVICE`. To permanently change it:

1. Find your device name: `make list-ios-devices`
2. Update the Makefile variable or set an environment variable:

   ```bash
   export IOS_PHYSICAL_DEVICE='My iPhone'
   make run-app-ios-device
   ```

**Important - mDNS Limitation on Free Apple Developer Accounts:**

The Multicast Networking entitlement required for mDNS device discovery is **NOT available** with free Apple Developer accounts (Personal Teams). This means:

- ✅ The app works perfectly on physical devices with **manual IP entry**
- ❌ Automatic mDNS device discovery is **disabled** (fallback option required)
- ✅ To enable mDNS discovery: enroll in the paid **Apple Developer Program** ($99/year), then uncomment the multicast plugin in `mobile/app.json` and rebuild

The multicast plugin line has been commented out by default to support free accounts. The app gracefully degrades to manual device entry when mDNS is unavailable.

**Note**: If you see a warning about "Unexpected devicectl JSON version output", this may indicate a devicectl compatibility issue. The app should still run correctly. If deployment fails, ensure:

- Your physical device is connected via USB and unlocked
- You've trusted the computer on your iOS device
- Xcode Command Line Tools are up to date: `xcode-select --install`
- The device name matches exactly (check with `make list-ios-devices`)
- You've configured code signing in Xcode with your Apple ID

**Missing native modules (e.g., expo-haptics):**

```bash
cd mobile
npm run typecheck                # Identify missing modules
npx expo install expo-haptics    # Install missing module
make build-app-ios               # Rebuild native projects (REQUIRED)
make run-app-ios                 # Run app
```

**IMPORTANT**: Always run `make build-app-ios` (or `npx expo prebuild`) after installing native Expo modules. Unlike JavaScript-only packages, native modules require regenerating the native projects.

**Metro bundler cache issues ("Unable to resolve module ./index"):**

```bash
# Use Makefile target for quick cache clear
make clean-cache-app   # Clears .expo and node_modules/.cache
make run-app-ios       # Restart with clear cache
```

Or manually:

```bash
cd mobile
rm -rf .expo node_modules/.cache
npx expo start --clear
# In another terminal: npm run ios
```

Common after installing native modules, running prebuild, or switching branches.

**"Unable to resolve module crypto" error:**

Occurs when packages (axios, etc.) load Node.js modules instead of React Native versions.

```bash
# Ensure metro.config.js has:
# config.resolver.resolverMainFields = ['react-native', 'browser', 'main']
# config.resolver.unstable_enablePackageExports = true

# Then clear cache and rebuild
make clean-app         # Clears cache + native projects
make build-app-ios     # Regenerate iOS project
make run-app-ios       # Run
```

**Type errors after API changes:**

```bash
make generate-app      # Regenerate types from OpenAPI spec
cd mobile && npm run typecheck  # Verify no type errors
```

**Native module not found:**

```bash
make clean-native-app  # Remove ios/android
make build-app-ios     # Regenerate native projects
make run-app-ios       # Rebuild app
```

**Tests failing:**

```bash
cd mobile
npm test -- --clearCache  # Clear Jest cache
npm test                  # Re-run tests
```

Or use Makefile:

```bash
make test-unit-app     # Run app unit tests
```

**Bundle size too large:**

```bash
cd mobile
npx react-native-bundle-visualizer  # Analyze bundle
# Consider code splitting or removing unused dependencies
```

## Recent Changes

- 003-mobile-onboarding-app: Added TypeScript 5.x with React Native 0.74+, targeting ES2022
- 003-mobile-onboarding-app: Implemented device discovery (mDNS + fallback), SRP-6a authentication, certificate pinning, device information display
- 003-mobile-onboarding-app: Added skeleton loading screens, haptic feedback, comprehensive error handling
