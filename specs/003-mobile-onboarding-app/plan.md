# Implementation Plan: Mobile Device Onboarding App

**Branch**: `003-mobile-onboarding-app` | **Date**: 2025-12-10 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-mobile-onboarding-app/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Build a cross-platform React Native mobile application for iOS and Android that enables system administrators to discover, authenticate with, and onboard headless Linux devices running the BoardingPass service. The app provides mDNS-based device discovery, QR code/manual connection code input, SRP-6a authentication, and device information display through a clean Material Design interface. The app implements certificate pinning for self-signed TLS certificates with transparent certificate status indicators and implements the BoardingPass RESTful API protocol without requiring knowledge of service implementation details.

## Technical Context

**Language/Version**: TypeScript 5.x with React Native 0.74+, targeting ES2022

**Primary Dependencies**:

- React Native (core framework)
- Expo SDK 51+ (managed workflow for cross-platform builds)
- React Native Paper 5.x (Material Design components)
- react-native-zeroconf (mDNS/Bonjour service discovery)
- react-native-vision-camera v4 (camera/QR scanning)
- Axios 1.x (HTTP client with TLS certificate pinning support)
- React Context API (state management)
- @react-navigation/native 6.x (navigation)

**Storage**:

- AsyncStorage for certificate pins and session tokens (encrypted)
- In-memory state for device discovery and active sessions
- No persistent storage of connection codes

**Testing**:

- Jest for unit tests
- React Native Testing Library for component tests
- Detox for E2E tests
- Contract tests against OpenAPI spec

**Target Platform**:

- iOS 15.0+ (iPhone and iPad)
- Android 10+ (API level 29+)
- Monorepo alongside BoardingPass Go service

**Project Type**: Mobile (React Native/Expo monorepo)

**Performance Goals**:

- Device discovery within 10 seconds
- Authentication completion under 30 seconds
- Device info display within 5 seconds post-auth
- 60 FPS UI rendering
- App bundle size < 50MB

**Constraints**:

- Must work on local networks with mDNS multicast
- Requires camera permissions for QR scanning
- Must gracefully handle network transitions
- Must support self-signed certificates with user confirmation
- Must not persist sensitive data (connection codes)

**Scale/Scope**:

- Single-user mobile app (no multi-user accounts)
- Support discovering and onboarding multiple devices
- 5-10 primary screens/views
- ~10-15 React components
- Integration with existing BoardingPass API (3 endpoints initially)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle Alignment

| Principle | Status | Compliance Notes |
|-----------|--------|-----------------|
| **I. Frictionless Bootstrapping** | ✅ PASS | Mobile app is the UX layer that enables frictionless bootstrapping. Minimizes user interaction through mDNS auto-discovery, QR code scanning, and automatic device info retrieval. |
| **II. Ephemeral & Fail-Safe Operation** | ✅ PASS | App manages ephemeral sessions (30min TTL). No persistent background processes. Fails safely on errors with clear recovery paths. Does not persist sensitive data. |
| **III. Minimal Footprint** | ⚠️ ACCEPTABLE | Mobile apps are inherently larger than Go binaries. Target <50MB bundle is reasonable for React Native. Expo managed workflow adds overhead but justified by cross-platform benefits and reduced maintenance burden. |
| **IV. Minimal Dependencies** | ⚠️ JUSTIFIED | React Native ecosystem requires more dependencies than Go stdlib. All dependencies are justified: Expo (cross-platform tooling), Paper (consistent UI), zeroconf (mDNS - no stdlib alternative), vision-camera (QR scanning), Axios (HTTP with cert pinning). See Complexity Tracking. |
| **V. Transport Agnostic & Protocol First** | ✅ PASS | App implements against OpenAPI contract from `specs/001-boardingpass-api/contracts/openapi.yaml`. Uses shared types from `pkg/protocol/`. No coupling to BoardingPass implementation details. HTTPS/REST is the transport; app logic is protocol-driven. |
| **VI. Open Source & Permissive Licensing** | ✅ PASS | Will be distributed under same license as BoardingPass (MIT/Apache). All dependencies use permissive licenses (React Native: MIT, Expo: MIT, Paper: MIT, others: MIT/Apache). |

### Security Requirements Compliance

| Requirement | Status | Implementation |
|------------|--------|----------------|
| **Authentication** | ✅ PASS | Implements SRP-6a authentication via `/auth/srp` endpoint with FIPS 140-3 compatible parameters (SHA-256 hash, RFC 5054 2048-bit group, g=2). Connection codes required before device access. |
| **Encryption** | ✅ PASS | All communication over HTTPS (TLS 1.3+). Certificate pinning for self-signed certs after first trust. |
| **Least Privilege** | ✅ PASS | App requests only required permissions (camera, network). No elevated privileges needed on mobile OS. |
| **Input Validation** | ✅ PASS | All user inputs validated before submission (FR-027). QR codes validated for format. API responses validated against OpenAPI schema. |
| **Secrets Management** | ✅ PASS | Connection codes never logged or persisted (FR-036). Session tokens encrypted in storage. Secrets cleared on logout/failure. |
| **Audit Logging** | ✅ PASS | Security events logged (auth attempts, cert trust decisions, failures) via FR-029. No sensitive data in logs. |

### Gate Result: ✅ PASS (with justified exceptions)

Mobile app complements the BoardingPass service and aligns with constitution principles. Dependency and footprint differences are inherent to mobile platform and justified by cross-platform requirements and UX needs.

## Project Structure

### Documentation (this feature)

```text
specs/003-mobile-onboarding-app/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── mobile-api.yaml  # Mobile-specific API extensions (if any)
├── checklists/          # Quality validation checklists
│   └── requirements.md  # Specification quality checklist (already created)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# React Native Mobile App (new monorepo structure)
mobile/
├── app/                      # Expo Router app directory
│   ├── (tabs)/              # Tab-based navigation
│   │   ├── index.tsx        # Device discovery screen
│   │   └── settings.tsx     # Settings screen
│   ├── device/              # Device-related screens
│   │   ├── [id].tsx         # Device detail screen
│   │   └── authenticate.tsx # Authentication screen
│   └── _layout.tsx          # Root layout
│
├── src/
│   ├── components/          # Reusable UI components
│   │   ├── DeviceList/      # Device list component
│   │   ├── QRScanner/       # QR code scanner component
│   │   ├── CertificateInfo/ # Certificate display component
│   │   └── ErrorBoundary/   # Error handling component
│   │
│   ├── services/            # Business logic and API integration
│   │   ├── discovery/       # mDNS device discovery
│   │   ├── auth/            # SRP-6a authentication
│   │   ├── api/             # BoardingPass API client
│   │   └── certificates/    # Certificate pinning and validation
│   │
│   ├── hooks/               # Custom React hooks
│   │   ├── useDeviceDiscovery.ts
│   │   ├── useAuth.ts
│   │   └── useSecureStorage.ts
│   │
│   ├── contexts/            # React Context providers
│   │   ├── DeviceContext.tsx
│   │   ├── AuthContext.tsx
│   │   └── CertificateContext.tsx
│   │
│   ├── types/               # TypeScript type definitions
│   │   ├── device.ts
│   │   ├── auth.ts
│   │   └── api.ts           # Generated from OpenAPI spec
│   │
│   └── utils/               # Utility functions
│       ├── validation.ts
│       ├── crypto.ts
│       └── errors.ts
│
├── tests/
│   ├── unit/               # Jest unit tests
│   ├── integration/        # Integration tests
│   ├── e2e/               # Detox E2E tests
│   └── contract/          # OpenAPI contract tests
│
├── app.json               # Expo configuration
├── package.json           # NPM dependencies
├── tsconfig.json          # TypeScript config
├── metro.config.js        # Metro bundler config
└── babel.config.js        # Babel config

# Existing Go service (unchanged location)
cmd/boardingpass/
internal/
pkg/protocol/              # Shared protocol types (consumed by mobile app)

# Root monorepo files (new)
package.json               # Root workspace config
turbo.json                # Turborepo config (optional)
```

**Structure Decision**: Monorepo with separate `mobile/` directory for React Native app alongside existing Go service in root. This enables:

- Shared protocol types from `pkg/protocol/` (TypeScript generation from Go types)
- Independent versioning and release cycles
- Separate testing and CI/CD pipelines
- Clear separation of concerns (mobile vs service)
- Uses Expo Router for file-based routing
- Expo managed workflow for simplified cross-platform builds

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| **Principle IV: Minimal Dependencies** - React Native ecosystem requires ~20+ direct dependencies | Cross-platform mobile development requires framework support not available in mobile OS base libraries. React Native, Expo, and associated libraries provide: cross-platform UI rendering (no native alternative that's cross-platform); mDNS service discovery (no stdlib equivalent on iOS/Android); QR code scanning (requires camera API abstraction); Material Design components (consistent UX). | **Native Swift/Kotlin**: Would require maintaining two separate codebases, doubling development and testing effort. Violates DRY principle and increases maintenance burden. **Flutter**: Similar dependency count, less mature ecosystem for enterprise apps, not specified in requirements. **Direct stdlib only**: Mobile OS APIs are platform-specific and low-level. Building cross-platform abstractions from scratch would recreate React Native. |
| **Principle III: Minimal Footprint** - React Native bundle size ~30-50MB vs <10MB for Go binary | Mobile apps include: JavaScript runtime (Hermes/JSC); Platform bridges (iOS/Android); UI framework and components; Platform-specific assets and resources. This is standard for modern mobile apps and acceptable to end users. | **Lighter frameworks**: No production-ready alternatives that meet cross-platform + UX requirements. Progressive Web App (PWA) lacks native features (mDNS, reliable camera access, certificate pinning) and offline capabilities. Native apps would still be ~20-30MB each. |

**Justification Summary**: Mobile app inherently requires more dependencies and larger footprint than server-side Go code. These are acceptable tradeoffs for:

1. Cross-platform development efficiency (single codebase vs two native apps)
2. Rich UX requirements (Material Design, smooth animations, camera integration)
3. Platform integration needs (mDNS, TLS certificate management, secure storage)
4. Maintainability and community support

The chosen stack (React Native + Expo + TypeScript) represents industry best practices for enterprise cross-platform mobile development with strong security characteristics.
