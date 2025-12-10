# Tasks: Mobile Device Onboarding App

**Input**: Design documents from `/specs/003-mobile-onboarding-app/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Test tasks are included throughout to ensure quality and enable TDD workflow. Tests should be written first and verified to fail before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Mobile monorepo**: `mobile/` directory within boardingpass repository
- **Mobile app paths**: `mobile/app/`, `mobile/src/`, `mobile/tests/`
- **Shared paths**: `pkg/protocol/` (Go types), `specs/001-boardingpass-api/contracts/` (OpenAPI)

---

## Phase 1: Setup (Build & Test Infrastructure)

**Purpose**: Project initialization, test infrastructure, and CI pipeline configuration

**Priority**: Complete this phase FIRST to enable test-driven development from the start

- [x] T001 Create mobile/ directory structure per plan.md (app/, src/, tests/, ios/, android/)
- [x] T002 Initialize React Native project with Expo SDK 51+ in mobile/package.json
- [x] T003 [P] Configure TypeScript 5.x with strict mode in mobile/tsconfig.json
- [x] T004 [P] Configure Expo app settings in mobile/app.json (bundle IDs, permissions, plugins)
- [x] T005 [P] Configure Metro bundler in mobile/metro.config.js
- [x] T006 [P] Configure Babel for React Native in mobile/babel.config.js
- [x] T007 Setup Jest test framework in mobile/jest.config.js
- [x] T008 [P] Setup React Native Testing Library in mobile/tests/setup.ts
- [x] T009 [P] Setup Detox E2E test framework in mobile/.detoxrc.json
- [x] T010 [P] Configure ESLint and Prettier in mobile/.eslintrc.js and mobile/.prettierrc
- [x] T011 Add npm scripts in mobile/package.json (start, test, lint, typecheck, generate:types)
- [x] T012 Create .env.example template in mobile/.env.example (default port, fallback IP, mDNS service name)
- [x] T013 Setup .gitignore for React Native (node_modules/, ios/, android/, .env)
- [x] T014 Run npx expo prebuild to generate ios/ and android/ directories
- [x] T015 Create GitHub Actions workflow for mobile CI in .github/workflows/mobile-ci.yml
- [x] T016 [P] Setup OpenAPI type generation script using openapi-typescript in mobile/package.json
- [x] T017 Run type generation to create mobile/src/types/api.ts from specs/001-boardingpass-api/contracts/openapi.yaml
- [x] T018 Verify build and test infrastructure: npm test, npm run lint, npm run typecheck all pass

**Checkpoint**: Build and test infrastructure is fully operational - all npm scripts work, CI pipeline configured

---

## Phase 2: Foundational (Core Infrastructure)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T019 Create root app layout with navigation in mobile/app/_layout.tsx
- [x] T020 [P] Create app context providers wrapper in mobile/src/contexts/AppProvider.tsx
- [x] T021 [P] Create DeviceContext with state management in mobile/src/contexts/DeviceContext.tsx
- [x] T022 [P] Create AuthContext with state management in mobile/src/contexts/AuthContext.tsx
- [x] T023 [P] Create CertificateContext with state management in mobile/src/contexts/CertificateContext.tsx
- [x] T024 Create TypeScript types for Device entity in mobile/src/types/device.ts
- [x] T025 [P] Create TypeScript types for AuthenticationSession in mobile/src/types/auth.ts
- [x] T026 [P] Create TypeScript types for CertificateInfo in mobile/src/types/certificate.ts
- [x] T027 Create base API client class in mobile/src/services/api/client.ts (Axios with HTTPS, timeout config)
- [x] T028 [P] Create secure storage utilities in mobile/src/hooks/useSecureStorage.ts (expo-secure-store wrapper)
- [x] T029 [P] Create validation utilities in mobile/src/utils/validation.ts (IP addresses, connection codes, fingerprints)
- [x] T030 [P] Create error utilities in mobile/src/utils/errors.ts (error types, error handling helpers)
- [x] T031 [P] Create crypto utilities in mobile/src/utils/crypto.ts (SHA-256 fingerprint computation with expo-crypto)
- [x] T032 [P] Create ErrorBoundary component in mobile/src/components/ErrorBoundary/index.tsx
- [x] T033 Create Material Design theme configuration in mobile/src/theme/index.ts (React Native Paper)
- [x] T034 Write unit tests for validation utilities in mobile/tests/unit/utils/validation.test.ts
- [x] T035 [P] Write unit tests for error utilities in mobile/tests/unit/utils/errors.test.ts
- [x] T036 [P] Write unit tests for crypto utilities in mobile/tests/unit/utils/crypto.test.ts

**Checkpoint**: Foundation ready - contexts, base types, utilities, and error handling infrastructure complete

---

## Phase 3: User Story 1 - Device Discovery (Priority: P1) 🎯 MVP

**Goal**: Enable users to discover BoardingPass devices on local network via mDNS and fallback IP

**Independent Test**: Launch app near a BoardingPass device and verify device appears in list; verify fallback IP detection works when mDNS unavailable

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T037 [P] [US1] Contract test for mDNS service discovery in mobile/tests/contract/mdns.test.ts
- [ ] T038 [P] [US1] Integration test for device discovery flow in mobile/tests/integration/discovery.test.ts
- [ ] T039 [P] [US1] Unit test for useDeviceDiscovery hook in mobile/tests/unit/hooks/useDeviceDiscovery.test.ts

### Implementation for User Story 1

- [ ] T040 [P] [US1] Install react-native-zeroconf and configure in mobile/app.json (Expo config plugin)
- [ ] T041 [US1] Create mDNS discovery service in mobile/src/services/discovery/mdns.ts (scan for _boardingpass._tcp)
- [ ] T042 [P] [US1] Create fallback IP service in mobile/src/services/discovery/fallback.ts (check 192.168.1.100:9443)
- [ ] T043 [US1] Create useDeviceDiscovery hook in mobile/src/hooks/useDeviceDiscovery.ts (combines mDNS + fallback)
- [ ] T044 [P] [US1] Create DeviceCard component in mobile/src/components/DeviceList/DeviceCard.tsx
- [ ] T045 [US1] Create DeviceList component in mobile/src/components/DeviceList/index.tsx
- [ ] T046 [US1] Implement device discovery screen in mobile/app/(tabs)/index.tsx
- [ ] T047 [US1] Add device status indicators (online, offline, authenticating) to DeviceCard
- [ ] T048 [US1] Add refresh functionality to device list (pull-to-refresh)
- [ ] T049 [US1] Handle duplicate device names (display IP as secondary identifier per FR-006)
- [ ] T050 [US1] Add empty state UI for "no devices found"
- [ ] T051 [US1] Add scanning state UI with loading indicator
- [ ] T052 [US1] Implement auto-refresh on device appear/disappear (FR-005)
- [ ] T053 [US1] Add logging for device discovery events (no sensitive data per FR-029)

**Checkpoint**: Device discovery should work independently - devices appear in list, refresh works, fallback IP detected

---

## Phase 4: User Story 2 - Device Authentication (Priority: P2)

**Goal**: Enable users to authenticate with devices using connection codes (manual or QR) via SRP-6a protocol

**Independent Test**: Tap device in list, enter/scan connection code, verify successful authentication and session token storage

### Tests for User Story 2

- [ ] T054 [P] [US2] Contract test for /auth/srp/init endpoint in mobile/tests/contract/auth-init.test.ts
- [ ] T055 [P] [US2] Contract test for /auth/srp/verify endpoint in mobile/tests/contract/auth-verify.test.ts
- [ ] T056 [P] [US2] Integration test for SRP-6a authentication flow in mobile/tests/integration/authentication.test.ts
- [ ] T057 [P] [US2] Unit test for SRP service in mobile/tests/unit/services/auth/srp.test.ts
- [ ] T058 [P] [US2] Unit test for session management in mobile/tests/unit/services/auth/session.test.ts

### Implementation for User Story 2

- [ ] T059 [P] [US2] Install secure-remote-password library in mobile/package.json
- [ ] T060 [P] [US2] Install expo-camera and expo-barcode-scanner in mobile/package.json
- [ ] T061 [US2] **CRITICAL**: Configure SRP-6a client with FIPS parameters in mobile/src/services/auth/srp.ts (SHA-256, RFC 5054 2048-bit, g=2)
- [ ] T062 [US2] Implement SRP-6a init flow in mobile/src/services/auth/srp.ts (POST /auth/srp/init)
- [ ] T063 [US2] Implement SRP-6a verify flow in mobile/src/services/auth/srp.ts (POST /auth/srp/verify)
- [ ] T064 [US2] Create session management service in mobile/src/services/auth/session.ts (store token, check expiry)
- [ ] T065 [US2] Create useAuth hook in mobile/src/hooks/useAuth.ts (wraps SRP flow and session management)
- [ ] T066 [P] [US2] Create QRScanner component in mobile/src/components/QRScanner/index.tsx
- [ ] T067 [P] [US2] Create ConnectionCodeInput component in mobile/src/components/ConnectionCodeInput/index.tsx
- [ ] T068 [US2] Implement authentication screen in mobile/app/device/authenticate.tsx
- [ ] T069 [US2] Add connection code validation (format check per FR-027)
- [ ] T070 [US2] Add camera permission handling with rationale UI (FR-026)
- [ ] T071 [US2] Implement QR code format validation (reject invalid codes)
- [ ] T072 [US2] Add manual/QR toggle UI in authentication screen
- [ ] T073 [US2] Clear connection code from memory after auth attempt (FR-036)
- [ ] T074 [US2] Implement progressive delay on auth failures (1s → 2s → 5s → 60s per FR-038)
- [ ] T075 [US2] Add authentication error handling (invalid code, timeout, network errors per FR-024, FR-028)
- [ ] T076 [US2] Add success transition to device detail screen
- [ ] T077 [US2] Add logging for auth events (no sensitive data: connection codes, SRP values, tokens)

**Checkpoint**: Authentication should work independently - can authenticate via manual or QR code, session stored securely

---

## Phase 5: User Story 3 - Device Information Display (Priority: P3)

**Goal**: Display device system information and network configuration after successful authentication

**Independent Test**: Authenticate with device, verify system info (TPM, board, CPU, OS, FIPS status) and network config displayed correctly

### Tests for User Story 3

- [ ] T078 [P] [US3] Contract test for /info endpoint in mobile/tests/contract/info.test.ts
- [ ] T079 [P] [US3] Contract test for /network endpoint in mobile/tests/contract/network.test.ts
- [ ] T080 [P] [US3] Integration test for device info retrieval in mobile/tests/integration/device-info.test.ts

### Implementation for User Story 3

- [ ] T081 [P] [US3] Create device info service in mobile/src/services/api/info.ts (GET /info with auth token)
- [ ] T082 [P] [US3] Create network config service in mobile/src/services/api/network.ts (GET /network with auth token)
- [ ] T083 [US3] Create useDeviceInfo hook in mobile/src/hooks/useDeviceInfo.ts (fetch info + network)
- [ ] T084 [P] [US3] Create SystemInfo display component in mobile/src/components/DeviceInfo/SystemInfo.tsx
- [ ] T085 [P] [US3] Create NetworkConfig display component in mobile/src/components/DeviceInfo/NetworkConfig.tsx
- [ ] T086 [P] [US3] Create TPMInfo display component in mobile/src/components/DeviceInfo/TPMInfo.tsx
- [ ] T087 [P] [US3] Create BoardInfo display component in mobile/src/components/DeviceInfo/BoardInfo.tsx
- [ ] T088 [US3] Implement device detail screen in mobile/app/device/[id].tsx
- [ ] T089 [US3] Format device data for readability (FR-018): UUIDs, MAC addresses, IP addresses
- [ ] T090 [US3] Add loading states for info queries (FR-019)
- [ ] T091 [US3] Add error handling for info query failures (FR-024, FR-028)
- [ ] T092 [US3] Add retry mechanism for transient failures (FR-025)
- [ ] T093 [US3] Display partial data when some queries succeed and others fail
- [ ] T094 [US3] Add FIPS status indicator (visual badge for enabled/validated)
- [ ] T095 [US3] Add network interface status indicators (up/down with colors)

**Checkpoint**: Device information should display correctly - all data formatted and readable, errors handled gracefully

---

## Phase 6: User Story 4 - Error Recovery and User Guidance (Priority: P2)

**Goal**: Provide clear error messages, recovery options, and user guidance for all failure scenarios

**Independent Test**: Simulate various failures (network errors, permission denials, timeouts) and verify clear error messages and recovery paths

### Tests for User Story 4

- [ ] T096 [P] [US4] Integration test for network error recovery in mobile/tests/integration/error-recovery.test.ts
- [ ] T097 [P] [US4] Integration test for permission denial handling in mobile/tests/integration/permission-errors.test.ts
- [ ] T098 [P] [US4] Unit test for error message generation in mobile/tests/unit/utils/error-messages.test.ts

### Implementation for User Story 4

- [ ] T099 [P] [US4] Create error message utilities in mobile/src/utils/error-messages.ts (user-friendly messages per FR-024)
- [ ] T100 [P] [US4] Create PermissionDeniedView component in mobile/src/components/Errors/PermissionDeniedView.tsx
- [ ] T101 [P] [US4] Create NetworkErrorView component in mobile/src/components/Errors/NetworkErrorView.tsx
- [ ] T102 [P] [US4] Create TimeoutErrorView component in mobile/src/components/Errors/TimeoutErrorView.tsx
- [ ] T103 [US4] Enhance ErrorBoundary with recovery actions (retry, navigate back)
- [ ] T104 [US4] Add network error handling to discovery service (show error, offer retry)
- [ ] T105 [US4] Add timeout error handling to auth service (show timeout message, offer retry)
- [ ] T106 [US4] Add permission error handling to QR scanner (guide to settings, offer manual entry)
- [ ] T107 [US4] Add unreachable device error handling (clear message, suggest retry or different device)
- [ ] T108 [US4] Implement global error toast notifications (non-critical errors)
- [ ] T109 [US4] Implement error dialog for critical errors (blocking issues)
- [ ] T110 [US4] Add "Open Settings" button for permission errors (iOS/Android deep links)
- [ ] T111 [US4] Add retry buttons to all error views
- [ ] T112 [US4] Add context-specific help text for common errors (mDNS blocked, network issues)

**Checkpoint**: Error handling should be comprehensive - all error scenarios have clear messages and recovery options

---

## Phase 7: Certificate Pinning & Security (Cross-Cutting)

**Purpose**: Implement certificate pinning for self-signed certificates with user confirmation

**Note**: This affects User Stories 2 and 3 (authentication and info retrieval require HTTPS)

- [ ] T113 Create certificate utilities in mobile/src/services/certificates/utils.ts (fetch cert, compute fingerprint)
- [ ] T114 [P] Create certificate validation service in mobile/src/services/certificates/validation.ts (pin on first trust, validate on subsequent)
- [ ] T115 [P] Create CertificateInfo display component in mobile/src/components/CertificateInfo/index.tsx
- [ ] T116 [P] Create CertificateStatusIndicator component in mobile/src/components/CertificateInfo/StatusIndicator.tsx
- [ ] T117 Create certificate trust dialog in mobile/src/components/CertificateInfo/TrustDialog.tsx
- [ ] T118 Integrate certificate fetching into API client (mobile/src/services/api/client.ts)
- [ ] T119 Implement certificate trust workflow (fetch → display info → user confirms → pin)
- [ ] T120 Implement certificate validation on subsequent connections (check fingerprint, alert on change)
- [ ] T121 Add certificate status indicators to device list (trusted CA, self-signed trusted, new, changed)
- [ ] T122 Add certificate info tooltip/popup to device cards (issuer, subject, validity, fingerprint per FR-033)
- [ ] T123 Implement certificate change alert (FR-035) with option to trust new cert or abort
- [ ] T124 Store certificate pins in secure storage (expo-secure-store)
- [ ] T125 Add self-signed vs CA-signed visual distinction (FR-032)
- [ ] T126 Write unit tests for certificate utilities in mobile/tests/unit/services/certificates/utils.test.ts
- [ ] T127 [P] Write integration tests for certificate pinning flow in mobile/tests/integration/certificate-pinning.test.ts

**Checkpoint**: Certificate pinning should work - users can trust self-signed certs, pinning prevents MITM, cert changes detected

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories and final quality assurance

- [ ] T128 [P] Add loading states and skeleton screens to all data-fetching screens
- [ ] T129 [P] Add haptic feedback for button presses and important events
- [ ] T130 [P] Add accessibility labels to all interactive elements
- [ ] T131 [P] Test and fix screen orientation changes (portrait/landscape per FR-023)
- [ ] T132 [P] Optimize bundle size (tree shaking, code splitting)
- [ ] T133 [P] Add performance monitoring (measure discovery time, auth time, info load time)
- [ ] T134 [P] Verify 60 FPS UI rendering (React DevTools profiler)
- [ ] T135 Add app icon and splash screen assets
- [ ] T136 [P] Write E2E test for complete onboarding flow in mobile/tests/e2e/onboarding.test.ts
- [ ] T137 [P] Write E2E test for error scenarios in mobile/tests/e2e/error-handling.test.ts
- [ ] T138 [P] Add comprehensive logging with log levels (debug, info, warn, error)
- [ ] T139 [P] Verify no sensitive data in logs (connection codes, tokens, SRP values per FR-029)
- [ ] T140 Run full test suite: npm test (all unit, integration, contract tests pass)
- [ ] T141 Run E2E tests: npm run test:e2e (all E2E scenarios pass)
- [ ] T142 Run linter: npm run lint (zero errors, zero warnings)
- [ ] T143 Run type checker: npm run typecheck (zero errors)
- [ ] T144 Test on physical iOS device (iPhone, verify camera, mDNS, secure storage)
- [ ] T145 Test on physical Android device (verify camera, mDNS, secure storage, permissions)
- [ ] T146 Validate against success criteria in spec.md (SC-001 through SC-010)
- [ ] T147 Run quickstart.md validation (follow setup guide from scratch, verify it works)
- [ ] T148 [P] Update CLAUDE.md with mobile app development workflow
- [ ] T149 Create mobile app README in mobile/README.md (setup, development, testing)

**Checkpoint**: App is production-ready - all tests pass, performance targets met, works on both platforms

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
  - **PRIORITY**: Complete FIRST to enable test-driven development
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phases 3-6)**: All depend on Foundational phase completion
  - User Story 1 (Device Discovery) - P1 Priority 🎯 MVP
  - User Story 2 (Authentication) - P2 Priority
  - User Story 3 (Device Info) - P3 Priority
  - User Story 4 (Error Handling) - P2 Priority
  - Stories can proceed in parallel (if staffed) or sequentially in priority order
- **Certificate Pinning (Phase 7)**: Can start after Foundational, integrates with US2 and US3
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Integrates with US1 (uses discovered devices) but independently testable
- **User Story 3 (P3)**: Depends on User Story 2 (requires authentication) - Should start after US2 complete
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - Enhances all stories but independently testable

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Contexts and utilities before services
- Services before components
- Components before screens
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- **Phase 1 (Setup)**: T003, T004, T005, T006, T008, T009, T010, T016 can run in parallel
- **Phase 2 (Foundational)**: T020-T023 (contexts), T025-T026 (types), T028-T031 (utilities), T034-T036 (tests) can run in parallel
- **Phase 3 (US1)**: T037-T039 (tests), T040+T042 (services), T044 (component) can run in parallel
- **Phase 4 (US2)**: T054-T058 (tests), T059-T060 (dependencies), T066-T067 (components) can run in parallel
- **Phase 5 (US3)**: T078-T080 (tests), T081-T082 (services), T084-T087 (components) can run in parallel
- **Phase 6 (US4)**: T096-T098 (tests), T099-T102 (components) can run in parallel
- **Phase 7 (Cert Pinning)**: T114-T116 (services and components) can run in parallel
- **Phase 8 (Polish)**: T128-T134, T136-T139, T148 can run in parallel

---

## Parallel Example: User Story 2 (Authentication)

```bash
# Launch all tests for User Story 2 together:
Task: "Contract test for /auth/srp/init endpoint in mobile/tests/contract/auth-init.test.ts"
Task: "Contract test for /auth/srp/verify endpoint in mobile/tests/contract/auth-verify.test.ts"
Task: "Integration test for SRP-6a authentication flow in mobile/tests/integration/authentication.test.ts"
Task: "Unit test for SRP service in mobile/tests/unit/services/auth/srp.test.ts"
Task: "Unit test for session management in mobile/tests/unit/services/auth/session.test.ts"

# Launch all components for User Story 2 together (after services complete):
Task: "Create QRScanner component in mobile/src/components/QRScanner/index.tsx"
Task: "Create ConnectionCodeInput component in mobile/src/components/ConnectionCodeInput/index.tsx"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (build & test infrastructure) ✅ PRIORITY
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Device Discovery)
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

**Deliverable**: Users can discover devices on network - MVP validated

### Incremental Delivery (Recommended)

1. Complete Setup (Phase 1) → Build & test infrastructure ready ✅
2. Complete Foundational (Phase 2) → Foundation ready
3. Add User Story 1 (Phase 3) → Test independently → Deploy/Demo (MVP!)
4. Add User Story 2 (Phase 4) → Test independently → Deploy/Demo (Can authenticate)
5. Add Certificate Pinning (Phase 7) → Integrate with US2 → Deploy/Demo (Secure HTTPS)
6. Add User Story 3 (Phase 5) → Test independently → Deploy/Demo (Can view device info)
7. Add User Story 4 (Phase 6) → Test independently → Deploy/Demo (Error handling complete)
8. Add Polish (Phase 8) → Final QA → Production release

**Benefit**: Each phase adds value without breaking previous functionality

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup (Phase 1) together - build & test infrastructure
2. Team completes Foundational (Phase 2) together
3. Once Foundational is done:
   - Developer A: User Story 1 (Device Discovery)
   - Developer B: User Story 2 (Authentication) + Certificate Pinning integration
   - Developer C: User Story 4 (Error Handling)
4. After US1 and US2 complete:
   - Developer A or B: User Story 3 (Device Info - depends on auth)
5. Team completes Polish (Phase 8) together

---

## FIPS Compatibility Checklist

**CRITICAL for User Story 2 (Authentication)**:

- [ ] T061: SRP-6a client MUST use SHA-256 hash (FIPS 180-4 approved)
- [ ] T061: SRP-6a client MUST use RFC 5054 2048-bit group (FIPS 186-4 compliant)
- [ ] T061: SRP-6a client MUST use generator g=2
- [ ] T061: Add development logging to verify SRP configuration matches server
- [ ] T056: Integration test MUST authenticate against actual BoardingPass service (not mocks)
- [ ] T077: Verify no SRP values logged (ephemeral keys, session keys, connection codes)

**References**:
- `specs/003-mobile-onboarding-app/research.md` Section 1 "FIPS Compatibility Requirements"
- `specs/003-mobile-onboarding-app/contracts/README.md` "FIPS Compatibility Requirements"
- `specs/003-mobile-onboarding-app/quickstart.md` "FIPS Compatibility Setup"

---

## Notes

- [P] tasks = different files, no dependencies, can run in parallel
- [Story] label maps task to specific user story for traceability and independent testing
- Each user story should be independently completable and testable
- Tests should be written FIRST and verified to fail before implementation (TDD)
- Commit after each task or logical group of related tasks
- Stop at any checkpoint to validate story independently
- Build & test infrastructure (Phase 1) is highest priority - enables TDD from start
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- Mobile paths: All source in `mobile/`, tests in `mobile/tests/`, generated native in `mobile/ios/` and `mobile/android/`
