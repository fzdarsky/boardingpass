# Tasks: BoardingPass API

**Input**: Design documents from `/specs/001-boardingpass-api/`
**Prerequisites**: plan.md (complete), spec.md (complete), research.md (complete), data-model.md (complete), contracts/ (complete)

**Feature**: BoardingPass is a lightweight, ephemeral bootstrap service for headless Linux devices, exposing a RESTful API over HTTPS with SRP-6a authentication.

**Testing Strategy**: Unit tests and linting are configured from the start. Each component includes corresponding unit tests to ensure code quality and coverage from day one.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure + CI/CD)

**Purpose**: Project initialization, basic structure, and automated testing infrastructure

- [X] T001 Create project directory structure per plan.md (cmd/, internal/, pkg/, tests/, build/, _output/)
- [X] T002 Initialize Go module with go.mod and go.sum
- [X] T003 [P] Create .gitignore file with _output/ and build artifacts
- [X] T004 [P] Create .golangci.yml configuration file including gosec and govulncheck linters
- [X] T005 [P] Create .github/workflows/service-ci.yaml with lint, test, build jobs (Go 1.25, golangci-lint v2.7.1)
- [X] T006 [P] Create .github/workflows/release.yaml with tag-triggered release workflow
- [X] T007 [P] Create .goreleaser.yaml configuration file for build orchestration to _output/dist/
- [X] T008 [P] Create build/Containerfile for build environment, using registry.access.redhat.com/ubi9/go-toolset:1.25 as base
- [X] T009 [P] Create Makefile with build, test, lint, coverage targets
- [X] T010 [P] Create LICENSE file (Apache 2.0)
- [X] T011 [P] Create README.md with project overview and badges

**Checkpoint**: CI/CD and testing infrastructure ready - all commits will now be automatically linted and tested

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T012 Create pkg/protocol/types.go with shared data structures (SystemInfo, NetworkConfig, ConfigBundle)
- [X] T013 [P] Create pkg/protocol/errors.go with standardized error codes and messages
- [X] T014 [P] Create tests/unit/protocol/types_test.go with unit tests for protocol types
- [X] T015 [P] Create tests/unit/protocol/errors_test.go with unit tests for error handling
- [X] T016 Create internal/config/config.go with ServiceConfig struct and YAML loading
- [X] T017 Create internal/config/validate.go with configuration validation logic
- [X] T018 [P] Create tests/unit/config/config_test.go with unit tests for config loading and validation
- [X] T019 Create internal/logging/logger.go with JSON logging to stdout/stderr and systemd integration
- [X] T020 Create internal/logging/redactor.go with secret redaction logic for auth tokens and config payloads
- [X] T021 [P] Create tests/unit/logging/logger_test.go with unit tests for logging and redaction
- [X] T022 Create internal/tls/certgen.go with self-signed certificate generation logic
- [X] T023 Create internal/tls/config.go with TLS 1.3+ configuration and FIPS cipher suites
- [X] T024 [P] Create tests/unit/tls/certgen_test.go with unit tests for certificate generation
- [X] T025 [P] Create tests/unit/tls/config_test.go with unit tests for TLS configuration
- [X] T026 Create internal/api/server.go with HTTP server setup, TLS configuration, and lifecycle management
- [X] T027 Create internal/api/middleware/logging.go with request/response logging middleware
- [X] T028 [P] Create internal/api/middleware/errors.go with error handling middleware
- [X] T029 [P] Create tests/unit/api/middleware_test.go with unit tests for middleware
- [X] T030 [P] Create build/boardingpass.service systemd unit file with sentinel file check and security hardening
- [X] T031 [P] Create build/boardingpass.sudoers configuration file for restricted command execution
- [X] T032 [P] Create build/password-generator.example script demonstrating device-unique password generation
- [X] T033 [P] Create build/scripts/install-hooks.sh for RPM/DEB pre/post-install hooks

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel, with automated testing ensuring quality

---

## Phase 3: User Story 1 - Secure Session Establishment (SRP) (Priority: P1) 🎯 MVP

**Goal**: Authenticate bootstrap operator using SRP-6a protocol with device-unique password to establish secure session without PKI

**Independent Test**: Can be fully tested by performing the SRP handshake against POST /auth/srp/init and POST /auth/srp/verify and verifying a valid Session Token is issued

### Implementation for User Story 1

- [X] T034 [P] [US1] Create internal/auth/verifier.go with SRP verifier file loading, password generator execution, and verifier computation
- [X] T035 [P] [US1] Create internal/auth/srp.go with SRP-6a protocol implementation using Go stdlib crypto
- [X] T036 [US1] Create internal/auth/session.go with session token generation, HMAC signing, and 30-minute TTL management
- [X] T037 [P] [US1] Create internal/auth/ratelimit.go with progressive delay brute force protection (1s, 2s, 5s, 60s lockout)
- [X] T038 [P] [US1] Create tests/unit/auth/verifier_test.go with unit tests for verifier loading and password generation
- [X] T039 [P] [US1] Create tests/unit/auth/srp_test.go with unit tests for SRP-6a protocol implementation
- [X] T040 [P] [US1] Create tests/unit/auth/session_test.go with unit tests for session token management and expiry
- [X] T041 [P] [US1] Create tests/unit/auth/ratelimit_test.go with unit tests for rate limiting and progressive delays
- [X] T042 [US1] Create internal/api/middleware/auth.go with session token validation middleware
- [X] T043 [US1] Create internal/api/handlers/auth.go with POST /auth/srp/init handler
- [X] T044 [US1] Create POST /auth/srp/verify handler in internal/api/handlers/auth.go
- [X] T045 [US1] Integrate rate limiting into auth handlers with Retry-After headers
- [X] T046 [US1] Add authentication logging with secret redaction in auth handlers
- [X] T047 [P] [US1] Create tests/integration/auth_test.go with integration tests for full SRP handshake flow
- [X] T048 [P] [US1] Create tests/contract/auth_test.go with OpenAPI contract validation for auth endpoints

**Checkpoint**: User Story 1 complete with full test coverage - SRP handshake works and returns valid session token

---

## Phase 4: User Story 2 - Device Inventory & Network Query (Priority: P2)

**Goal**: Query device hardware identity and network state to verify compatibility and troubleshoot connectivity

**Independent Test**: Can be fully tested by authenticating and querying GET /info and GET /network endpoints

### Implementation for User Story 2

- [X] T049 [P] [US2] Create internal/inventory/tpm.go with TPM information extraction from /sys/class/tpm
- [X] T050 [P] [US2] Create internal/inventory/board.go with DMI board information extraction via dmidecode or /sys/class/dmi
- [X] T051 [P] [US2] Create internal/inventory/cpu.go with CPU architecture detection from runtime.GOARCH
- [X] T052 [P] [US2] Create internal/inventory/os.go with OS distribution and version detection from /etc/os-release
- [X] T053 [P] [US2] Create internal/inventory/fips.go with FIPS mode status check from /proc/sys/crypto/fips_enabled
- [X] T054 [P] [US2] Create tests/unit/inventory/tpm_test.go with unit tests for TPM extraction
- [X] T055 [P] [US2] Create tests/unit/inventory/board_test.go with unit tests for board info extraction
- [X] T056 [P] [US2] Create tests/unit/inventory/cpu_test.go with unit tests for CPU detection
- [X] T057 [P] [US2] Create tests/unit/inventory/os_test.go with unit tests for OS detection
- [X] T058 [P] [US2] Create tests/unit/inventory/fips_test.go with unit tests for FIPS mode check
- [X] T059 [P] [US2] Create internal/network/interfaces.go with network interface enumeration via D-Bus or netlink
- [X] T060 [P] [US2] Create internal/network/linkstate.go with link state detection (up/down)
- [X] T061 [P] [US2] Create internal/network/addresses.go with IP address extraction (IPv4 and IPv6)
- [X] T062 [P] [US2] Create tests/unit/network/interfaces_test.go with unit tests for interface enumeration
- [X] T063 [P] [US2] Create tests/unit/network/linkstate_test.go with unit tests for link state detection
- [X] T064 [P] [US2] Create tests/unit/network/addresses_test.go with unit tests for IP address extraction
- [X] T065 [US2] Create internal/api/handlers/info.go with GET /info endpoint assembling SystemInfo response
- [X] T066 [US2] Create internal/api/handlers/network.go with GET /network endpoint assembling NetworkConfig response
- [X] T067 [US2] Add 1-second response caching to GET /info handler to reduce syscall overhead
- [X] T068 [US2] Add authentication requirement to info and network handlers via middleware
- [X] T069 [P] [US2] Create tests/integration/info_test.go with integration tests for /info endpoint
- [X] T070 [P] [US2] Create tests/integration/network_test.go with integration tests for /network endpoint
- [X] T071 [P] [US2] Create tests/contract/info_test.go with OpenAPI contract validation for /info endpoint
- [X] T072 [P] [US2] Create tests/contract/network_test.go with OpenAPI contract validation for /network endpoint

**Checkpoint**: User Stories 1 AND 2 complete with full test coverage - can authenticate and query device information

---

## Phase 5: User Story 3 - Atomic Configuration Provisioning (Priority: P3)

**Goal**: Upload complete configuration bundle and apply to /etc in single atomic transaction

**Independent Test**: Can be fully tested by posting a JSON configuration bundle and verifying files land in /etc with correct permissions and allow-list validation

### Implementation for User Story 3

- [ ] T073 [P] [US3] Create internal/provisioning/bundle.go with ConfigBundle parsing and Base64 decoding
- [ ] T074 [P] [US3] Create internal/provisioning/pathvalidator.go with path allow-list validation against config.yaml
- [ ] T075 [US3] Create internal/provisioning/applier.go with atomic file application logic (temp → validate → rename)
- [ ] T076 [P] [US3] Create internal/provisioning/rollback.go with rollback mechanism on failure
- [ ] T077 [P] [US3] Create tests/unit/provisioning/bundle_test.go with unit tests for bundle parsing
- [ ] T078 [P] [US3] Create tests/unit/provisioning/pathvalidator_test.go with unit tests for path validation
- [ ] T079 [P] [US3] Create tests/unit/provisioning/applier_test.go with unit tests for atomic operations
- [ ] T080 [P] [US3] Create tests/unit/provisioning/rollback_test.go with unit tests for rollback logic
- [ ] T081 [US3] Create internal/api/handlers/configure.go with POST /configure endpoint
- [ ] T082 [US3] Add configuration bundle size validation (10MB max, 100 files max) to configure handler
- [ ] T083 [US3] Add path allow-list validation before provisioning in configure handler
- [ ] T084 [US3] Add authentication requirement to configure handler
- [ ] T085 [US3] Add strict content redaction for configuration payloads in logs
- [ ] T086 [P] [US3] Create tests/integration/configure_test.go with integration tests for configuration provisioning
- [ ] T087 [P] [US3] Create tests/contract/configure_test.go with OpenAPI contract validation for /configure endpoint

**Checkpoint**: User Stories 1, 2, AND 3 complete with full test coverage - can provision configuration bundles

---

## Phase 6: User Story 4 - System Command Execution (Priority: P4)

**Goal**: Execute allow-listed commands via sudo to activate applied configurations

**Independent Test**: Can be tested by triggering allowed commands and observing system behavior or mock script execution

### Implementation for User Story 4

- [ ] T088 [P] [US4] Create internal/command/allowlist.go with command allow-list loading from config.yaml
- [ ] T089 [P] [US4] Create internal/command/executor.go with sudo command execution and output capture
- [ ] T090 [P] [US4] Create internal/command/sudoers.go with sudoers file validation logic
- [ ] T091 [P] [US4] Create tests/unit/command/allowlist_test.go with unit tests for allow-list validation
- [ ] T092 [P] [US4] Create tests/unit/command/executor_test.go with unit tests for command execution
- [ ] T093 [P] [US4] Create tests/unit/command/sudoers_test.go with unit tests for sudoers validation
- [ ] T094 [US4] Create internal/api/handlers/command.go with POST /command endpoint
- [ ] T095 [US4] Add command ID validation against allow-list before execution
- [ ] T096 [US4] Add stdout/stderr capture and return in JSON response
- [ ] T097 [US4] Add authentication requirement to command handler
- [ ] T098 [US4] Add command execution logging with exit codes
- [ ] T099 [P] [US4] Create tests/integration/command_test.go with integration tests for command execution
- [ ] T100 [P] [US4] Create tests/contract/command_test.go with OpenAPI contract validation for /command endpoint

**Checkpoint**: User Stories 1-4 complete with full test coverage - can execute system commands

---

## Phase 7: User Story 5 - Automatic Lifecycle Management (Priority: P5)

**Goal**: Manage service lifecycle to ensure ephemeral operation and explicit completion

**Independent Test**: Can be tested by creating sentinel file and attempting to start service, calling POST /complete, or waiting for timeout

### Implementation for User Story 5

- [ ] T101 [P] [US5] Create internal/lifecycle/sentinel.go with sentinel file checking and creation logic
- [ ] T102 [P] [US5] Create internal/lifecycle/timeout.go with inactivity timeout tracking (10-minute default)
- [ ] T103 [P] [US5] Create internal/lifecycle/shutdown.go with graceful shutdown handler and signal handling
- [ ] T104 [P] [US5] Create tests/unit/lifecycle/sentinel_test.go with unit tests for sentinel file operations
- [ ] T105 [P] [US5] Create tests/unit/lifecycle/timeout_test.go with unit tests for timeout tracking
- [ ] T106 [P] [US5] Create tests/unit/lifecycle/shutdown_test.go with unit tests for shutdown logic
- [ ] T107 [US5] Create internal/api/handlers/complete.go with POST /complete endpoint
- [ ] T108 [US5] Integrate sentinel file check into cmd/boardingpass/main.go startup sequence
- [ ] T109 [US5] Integrate inactivity timeout into HTTP server in internal/api/server.go
- [ ] T110 [US5] Add graceful shutdown trigger in POST /complete handler
- [ ] T111 [US5] Add authentication requirement to complete handler
- [ ] T112 [US5] Update systemd unit file with ConditionPathExists=!/etc/boardingpass/issued
- [ ] T113 [P] [US5] Create tests/integration/lifecycle_test.go with integration tests for lifecycle management
- [ ] T114 [P] [US5] Create tests/contract/complete_test.go with OpenAPI contract validation for /complete endpoint

**Checkpoint**: All user stories complete with full test coverage - service manages its own lifecycle

---

## Phase 8: Integration & Main Entry Point

**Purpose**: Wire all components together into functional service binary

- [ ] T115 Create cmd/boardingpass/main.go with configuration loading, server initialization, and lifecycle management
- [ ] T116 Add signal handling (SIGTERM, SIGINT) for graceful shutdown in main.go
- [ ] T117 Add API route registration for all endpoints in internal/api/server.go
- [ ] T118 Add startup logging with version, commit, and configuration summary
- [ ] T119 Add systemd notify integration for Type=notify in main.go
- [ ] T120 [P] Create tests/e2e/service_test.go with end-to-end tests in containerized systemd environment
- [ ] T121 Verify test coverage meets 80% threshold across all packages

**Checkpoint**: Full service integration complete with E2E test coverage

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories and final validation

- [ ] T122 [P] Create docs/development.md with local setup, build, test instructions
- [ ] T123 [P] Create docs/deployment.md with package installation and configuration guide
- [ ] T124 [P] Create docs/api.md with API documentation generated from OpenAPI spec
- [ ] T125 [P] Create docs/security.md with security considerations and best practices
- [ ] T126 Code cleanup and refactoring across all packages
- [ ] T127 Add comprehensive error messages with context to all error returns
- [ ] T128 Validate binary size < 10MB after build
- [ ] T129 Validate memory consumption < 50MB idle via profiling
- [ ] T130 Validate SRP handshake < 500ms on Raspberry Pi 4 equivalent
- [ ] T131 Run quickstart.md validation end-to-end
- [ ] T132 Verify OpenAPI contract compliance for all endpoints
- [ ] T133 Run golangci-lint with zero warnings
- [ ] T134 Generate and review test coverage report

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately - **CI/CD configured from the start**
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3 → P4 → P5)
  - **Each component includes unit tests for immediate validation**
- **Integration (Phase 8)**: Depends on all user stories being complete
- **Polish (Phase 9)**: Depends on Integration being complete

### User Story Dependencies

- **User Story 1 (P1) - SRP Authentication**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2) - Device Info & Network**: Can start after Foundational (Phase 2) - Needs US1 for auth middleware
- **User Story 3 (P3) - Configuration Provisioning**: Can start after Foundational (Phase 2) - Needs US1 for auth middleware
- **User Story 4 (P4) - Command Execution**: Can start after Foundational (Phase 2) - Needs US1 for auth middleware
- **User Story 5 (P5) - Lifecycle Management**: Can start after Foundational (Phase 2) - Integrates with all stories but can be developed independently

### Within Each User Story

- Unit tests written alongside implementation (not strict TDD, but tests for each component)
- Models/utilities before services
- Services before handlers
- Handlers before middleware integration
- Integration tests after handlers complete
- Contract tests to validate OpenAPI compliance
- Core implementation before logging/monitoring
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user story implementations can start in parallel (if team capacity allows)
- Within each user story, tasks marked [P] can run in parallel
- Unit tests for different packages can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1 (SRP Authentication)

```bash
# After Foundational phase is complete, launch all [P] tasks for US1:
Task T034: "Create internal/auth/verifier.go"
Task T035: "Create internal/auth/srp.go"
Task T037: "Create internal/auth/ratelimit.go"
Task T038: "Create tests/unit/auth/verifier_test.go"
Task T039: "Create tests/unit/auth/srp_test.go"
Task T041: "Create tests/unit/auth/ratelimit_test.go"

# Then sequentially:
Task T036: "Create internal/auth/session.go" (depends on T035)
Task T040: "Create tests/unit/auth/session_test.go" (depends on T036)
Task T042: "Create internal/api/middleware/auth.go" (depends on T036)
Task T043-T046: Auth handlers and integration (depends on T034-T042)
Task T047-T048: Integration and contract tests (depends on handlers)
```

---

## Parallel Example: User Story 2 (Device Info & Network)

```bash
# After Foundational phase is complete, launch all inventory tasks in parallel:
Task T049: "Create internal/inventory/tpm.go"
Task T050: "Create internal/inventory/board.go"
Task T051: "Create internal/inventory/cpu.go"
Task T052: "Create internal/inventory/os.go"
Task T053: "Create internal/inventory/fips.go"
Task T054-T058: Unit tests for inventory components

# And all network tasks in parallel:
Task T059: "Create internal/network/interfaces.go"
Task T060: "Create internal/network/linkstate.go"
Task T061: "Create internal/network/addresses.go"
Task T062-T064: Unit tests for network components

# Then handlers:
Task T065: "Create internal/api/handlers/info.go" (depends on inventory tasks)
Task T066: "Create internal/api/handlers/network.go" (depends on network tasks)
Task T069-T072: Integration and contract tests (depends on handlers)
```

---

## Implementation Strategy

### MVP First (User Story 1 + User Story 2 Only)

1. Complete Phase 1: Setup (including CI/CD - **tests run automatically on every commit**)
2. Complete Phase 2: Foundational (with unit tests for all foundation components)
3. Complete Phase 3: User Story 1 (SRP Authentication with full test coverage)
4. Complete Phase 4: User Story 2 (Device Info & Network with full test coverage)
5. Complete Phase 8: Integration (minimal - just US1 + US2)
6. **STOP and VALIDATE**: Test SRP auth + device query flow end-to-end with CI passing
7. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready, **CI running on every commit**
2. Add User Story 1 + tests → CI validates → Basic auth works
3. Add User Story 2 + tests → CI validates → Can query device info (MVP!)
4. Add User Story 3 + tests → CI validates → Can provision configs
5. Add User Story 4 + tests → CI validates → Can execute commands
6. Add User Story 5 + tests → CI validates → Lifecycle managed
7. Each story adds value without breaking previous stories (validated by CI)

### Parallel Team Strategy

With multiple developers (recommended after Foundational phase):

1. Team completes Setup + Foundational together (CI/CD and tests operational)
2. Once Foundational is done:
   - Developer A: User Story 1 (SRP Authentication) - CRITICAL PATH, includes unit tests
   - Developer B: User Story 2 (Device Info) - parallel to A, includes unit tests
   - Developer C: User Story 3 (Config Provisioning) - parallel to A, includes unit tests
   - Developer D: User Story 5 (Lifecycle) - parallel to A, includes unit tests
3. After US1 complete, all stories can integrate auth middleware
4. Developer E: User Story 4 (Commands) - after US3 complete, includes unit tests
5. Stories complete and integrate independently, **CI validates all changes**

---

## Notes

- **CI/CD configured from day one**: All commits are automatically linted and tested
- **Unit tests alongside implementation**: Each component has corresponding tests
- **Coverage threshold**: 80% minimum, validated in CI
- **License**: Apache 2.0
- [P] tasks = different files, no dependencies, can run in parallel
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- All file paths follow the project structure defined in plan.md
- GoReleaser outputs to _output/dist/, local builds to _output/bin/
- Use Go stdlib crypto/* for FIPS 140-3 compliance
- Session tokens expire after 30 minutes
- Brute force protection uses progressive delays (1s, 2s, 5s, 60s)
- Configuration paths must be validated against allow-list
- TLS certificates auto-generated at first boot if not in OS image
