# Tasks: Boarding CLI Tool

**Input**: Design documents from `/specs/002-boarding-cli/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/README.md, quickstart.md

**Tests**: Tests are co-located with implementation files following existing project patterns (e.g., `srp_test.go` alongside `srp.go`). Table-driven tests using `testify/assert`.

**Organization**: Tasks are grouped by user story (P1-P6 from spec.md) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4, US5, US6)
- Include exact file paths in descriptions

## Path Conventions

Project uses Go standard layout:
- Binaries: `cmd/boarding/`
- Internal packages: `internal/cli/`
- Shared packages: `pkg/protocol/`
- Tests: Co-located with source files

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic CLI structure

- [x] T001 Create directory structure for CLI: `cmd/boarding/`, `internal/cli/{client,commands,config,session,tls,output}/`
- [x] T002 [P] Create `cmd/boarding/main.go` with command routing skeleton (switch statement for 6 commands)
- [x] T003 [P] Update `Makefile` with `build-cli` and `build-all` targets
- [x] T004 [P] Update `.goreleaser.yaml` with boarding binary build configuration

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Implement configuration loading in `internal/cli/config/config.go` with precedence (flags > env > file)
- [x] T006 [P] Implement OS-specific directory helpers in `internal/cli/config/dirs.go` (UserConfigDir, UserCacheDir)
- [x] T007 [P] Implement session token storage in `internal/cli/session/store.go` with 0600 permissions
- [x] T008 [P] Implement SHA-256 certificate fingerprint computation in `internal/cli/tls/fingerprint.go`
- [x] T009 [P] Implement certificate fingerprint storage (YAML) in `internal/cli/tls/store.go`
- [x] T010 [P] Implement interactive certificate prompt in `internal/cli/tls/prompt.go`
- [x] T011 [P] Implement custom HTTP RoundTripper in `internal/cli/client/transport.go` for TOFU cert verification
- [x] T012 [P] Implement YAML formatter in `internal/cli/output/formatter.go`
- [x] T013 [P] Implement JSON formatter in `internal/cli/output/formatter.go`
- [x] T014 Write unit tests for config precedence in `internal/cli/config/config_test.go`
- [x] T015 [P] Write unit tests for session store in `internal/cli/session/store_test.go`
- [x] T016 [P] Write unit tests for TLS fingerprinting in `internal/cli/tls/fingerprint_test.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Authentication (Priority: P1) 🎯 MVP

**Goal**: Enable users to authenticate with BoardingPass service using SRP-6a protocol and store session tokens

**Independent Test**: Run `boarding pass --host <ip> --username admin`, verify session token file created in `~/.cache/boardingpass/`, verify subsequent commands use the token

### Implementation for User Story 1

- [x] T017 [P] [US1] Implement SRP-6a client ephemeral keypair generation in `internal/cli/client/srp.go` (GenerateEphemeralKeypair function)
- [x] T018 [P] [US1] Implement SRP-6a shared secret computation in `internal/cli/client/srp.go` (ComputeSharedSecret function)
- [x] T019 [P] [US1] Implement SRP-6a client proof M1 computation in `internal/cli/client/srp.go` (ComputeClientProof function)
- [x] T020 [P] [US1] Implement SRP-6a server proof M2 verification in `internal/cli/client/srp.go` (VerifyServerProof function)
- [x] T021 [P] [US1] Implement SRP-6a private key derivation in `internal/cli/client/srp.go` (DerivePrivateKey function)
- [x] T022 [US1] Implement HTTP client foundation in `internal/cli/client/client.go` (NewClient, with TLS config, custom transport, session token injection)
- [x] T023 [US1] Implement SRP Init API call in `internal/cli/client/client.go` (SRPInit method: POST /auth/srp/init)
- [x] T024 [US1] Implement SRP Verify API call in `internal/cli/client/client.go` (SRPVerify method: POST /auth/srp/verify)
- [x] T025 [US1] Implement common command infrastructure in `internal/cli/commands/common.go` (config loading, client creation, error handling)
- [x] T026 [US1] Implement `pass` command in `internal/cli/commands/pass.go` (flags, interactive password prompt, full SRP flow, token storage)
- [x] T027 [US1] Write unit tests for SRP client in `pkg/srp/client_test.go` (test all crypto functions with known test vectors)
- [x] T028 [US1] Create integration test for SRP authentication flow in `tests/cli-integration/auth_test.go` (SRP protocol flow validation)

**Checkpoint**: At this point, User Story 1 should be fully functional - users can authenticate and session tokens are stored

---

## Phase 4: User Story 2 - Query System Information (Priority: P2)

**Goal**: Enable users to query device system information (CPU, board, TPM, OS, FIPS status) and display as YAML/JSON

**Independent Test**: Run `boarding info` after authentication, verify system information is displayed in YAML format; run `boarding info -o json`, verify JSON format

### Implementation for User Story 2

- [ ] T029 [P] [US2] Implement GET /info API call in `internal/cli/client/client.go` (GetInfo method with session token auth)
- [ ] T030 [US2] Implement `info` command in `internal/cli/commands/info.go` (flags for output format, call GetInfo, format output)
- [ ] T031 [US2] Write integration test for info command in `tests/cli-integration/commands_test.go` (mock /info endpoint, test YAML and JSON output)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Query Network Interfaces (Priority: P3)

**Goal**: Enable users to query network interface configuration and display as YAML/JSON

**Independent Test**: Run `boarding connections` after authentication, verify network interfaces displayed in YAML; run `boarding connections -o json`, verify JSON format

### Implementation for User Story 3

- [ ] T032 [P] [US3] Implement GET /network API call in `internal/cli/client/client.go` (GetNetwork method with session token auth)
- [ ] T033 [US3] Implement `connections` command in `internal/cli/commands/connections.go` (flags for output format, call GetNetwork, format output)
- [ ] T034 [US3] Write integration test for connections command in `tests/cli-integration/commands_test.go` (mock /network endpoint, test YAML and JSON output)

**Checkpoint**: All read-only query operations (US1-US3) should now be independently functional

---

## Phase 6: User Story 4 - Upload Configuration (Priority: P4)

**Goal**: Enable users to upload configuration directories to the device for provisioning

**Independent Test**: Create test config directory with sample files, run `boarding load /path/to/config`, verify files uploaded, verify progress feedback displayed

### Implementation for User Story 4

- [ ] T035 [P] [US4] Implement directory scanning in `internal/cli/commands/load.go` (walk directory, collect files, validate size/count limits)
- [ ] T036 [P] [US4] Implement progress tracking in `internal/cli/commands/load.go` (progress bar or status for file upload)
- [ ] T037 [US4] Implement POST /configure API call in `internal/cli/client/client.go` (PostConfigure method with multipart form data, session token auth)
- [ ] T038 [US4] Implement `load` command in `internal/cli/commands/load.go` (directory arg, scan files, call PostConfigure, display progress, handle errors)
- [ ] T039 [US4] Write integration test for load command in `tests/cli-integration/commands_test.go` (mock /configure endpoint, test file upload, test error cases)

**Checkpoint**: Configuration provisioning (US4) should work independently

---

## Phase 7: User Story 5 - Execute Commands (Priority: P5)

**Goal**: Enable users to execute allow-listed commands on the device and display output

**Independent Test**: Run `boarding command "systemctl status sshd"` after authentication, verify command executes, verify stdout/stderr displayed

### Implementation for User Story 5

- [ ] T040 [P] [US5] Implement POST /command API call in `internal/cli/client/client.go` (ExecuteCommand method with session token auth)
- [ ] T041 [US5] Implement `command` command in `internal/cli/commands/command.go` (command string arg, call ExecuteCommand, display stdout/stderr, exit code handling)
- [ ] T042 [US5] Write integration test for command execution in `tests/cli-integration/commands_test.go` (mock /command endpoint, test output display, test error handling for non-allowed commands)

**Checkpoint**: Command execution (US5) should work independently

---

## Phase 8: User Story 6 - Logout and Complete Session (Priority: P6)

**Goal**: Enable users to terminate their session, trigger BoardingPass service to finalize provisioning

**Independent Test**: Run `boarding complete` after authentication, verify session token deleted, verify /complete endpoint called

### Implementation for User Story 6

- [ ] T043 [P] [US6] Implement POST /complete API call in `internal/cli/client/client.go` (Complete method with session token auth)
- [ ] T044 [US6] Implement `complete` command in `internal/cli/commands/complete.go` (call Complete, delete session token, display success message)
- [ ] T045 [US6] Write integration test for complete command in `tests/cli-integration/commands_test.go` (mock /complete endpoint, verify token deletion)

**Checkpoint**: All user stories (US1-US6) should now be independently functional

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T046 [P] Create end-to-end test for full provisioning workflow in `tests/cli-e2e/workflow_test.go` (pass → info → load → command → complete)
- [ ] T047 [P] Add error handling for network failures with retry logic in `internal/cli/client/client.go`
- [ ] T048 [P] Add error handling for session expiry (401 responses) in `internal/cli/client/client.go`
- [ ] T049 [P] Add validation for malformed API responses in `internal/cli/client/client.go`
- [ ] T050 [P] Update README.md with CLI tool section (installation, quick start, command reference)
- [ ] T051 [P] Add usage help text for each command (--help flag support)
- [ ] T052 Run `make lint` and fix all linting errors
- [ ] T053 Run `make test` and verify all tests pass
- [ ] T054 Run quickstart.md validation (manually test all examples from quickstart.md)
- [ ] T055 [P] Generate CLI usage documentation for README

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phases 3-8)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3 → P4 → P5 → P6)
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Depends on US1 (needs authentication)
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Depends on US1 (needs authentication)
- **User Story 4 (P4)**: Can start after Foundational (Phase 2) - Depends on US1 (needs authentication)
- **User Story 5 (P5)**: Can start after Foundational (Phase 2) - Depends on US1 (needs authentication)
- **User Story 6 (P6)**: Can start after Foundational (Phase 2) - Depends on US1 (needs authentication and session management)

**Note**: While US2-US6 technically depend on US1 for authentication, they can be developed in parallel once US1's session management is complete. Each story should be independently testable with mocked authentication.

### Within Each User Story

- Models/data structures first (parallelizable if in different files)
- API client methods (depends on client foundation from US1)
- Commands (depends on API methods and formatters)
- Tests (can write in parallel with implementation)

### Parallel Opportunities

**Phase 1 (Setup)**: All 4 tasks can run in parallel
- T002 (main.go), T003 (Makefile), T004 (.goreleaser.yaml)

**Phase 2 (Foundational)**: Multiple tasks can run in parallel:
- Group 1: T006 (dirs.go), T007 (session/store.go), T008+T009+T010 (tls/ package), T012+T013 (output/formatter.go)
- Group 2: T014, T015, T016 (unit tests - can write while implementing)
- T005 (config.go) should complete first as other components depend on it

**Phase 3 (US1)**: SRP functions can be written in parallel:
- T017-T021 (all SRP functions in srp.go - different functions, same file)
- Then T022-T024 (HTTP client methods)
- Then T025-T026 (commands)
- Then T027-T028 (tests)

**Phases 4-8 (US2-US6)**: Each user story is independent:
- Can assign different developers to different stories simultaneously
- Within each story, API method and command can be developed in quick succession

**Phase 9 (Polish)**: All marked [P] tasks can run in parallel:
- T046-T051 can all be done simultaneously

---

## Parallel Example: Foundational Phase

```bash
# Launch multiple foundational components in parallel:
Task: "Implement OS-specific directory helpers in internal/cli/config/dirs.go"
Task: "Implement session token storage in internal/cli/session/store.go"
Task: "Implement SHA-256 certificate fingerprint computation in internal/cli/tls/fingerprint.go"
Task: "Implement YAML formatter in internal/cli/output/formatter.go"

# Then in parallel:
Task: "Write unit tests for session store in internal/cli/session/store_test.go"
Task: "Write unit tests for TLS fingerprinting in internal/cli/tls/fingerprint_test.go"
```

## Parallel Example: User Story 1

```bash
# Launch all SRP functions together (different functions, same file):
Task: "Implement SRP-6a client ephemeral keypair generation in internal/cli/client/srp.go"
Task: "Implement SRP-6a shared secret computation in internal/cli/client/srp.go"
Task: "Implement SRP-6a client proof M1 computation in internal/cli/client/srp.go"
Task: "Implement SRP-6a server proof M2 verification in internal/cli/client/srp.go"
Task: "Implement SRP-6a private key derivation in internal/cli/client/srp.go"

# Note: These are in the same file but represent distinct functions that can be
# developed in parallel with merge/conflict resolution at the end
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Authentication)
4. **STOP and VALIDATE**: Test authentication independently
   - `boarding pass --host <ip> --username admin`
   - Verify session token created
   - Verify subsequent commands fail with clear "not yet implemented" errors
5. Deploy/demo MVP

**MVP Deliverable**: Users can authenticate with BoardingPass service

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 (P1: Authentication) → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 (P2: System Info) → Test independently → Deploy/Demo
4. Add User Story 3 (P3: Network Info) → Test independently → Deploy/Demo
5. Add User Story 4 (P4: Config Upload) → Test independently → Deploy/Demo
6. Add User Story 5 (P5: Command Execution) → Test independently → Deploy/Demo
7. Add User Story 6 (P6: Session Completion) → Test independently → Deploy/Demo
8. Complete Phase 9 (Polish) → Final release

**Value Delivery**: Each story adds incrementally more functionality without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (critical path)
2. Once Foundational is done:
   - Developer A: User Story 1 (Authentication) - highest priority, blocks others
3. Once US1 session management is complete:
   - Developer A: User Story 2 (System Info)
   - Developer B: User Story 3 (Network Info)
   - Developer C: User Story 4 (Config Upload)
4. Continue parallel development for US5 and US6
5. Team completes Polish phase together

**Team Coordination**: While US2-US6 depend on US1, they can be mocked for parallel development once the session interface is defined

---

## Task Summary

**Total Tasks**: 55

### Tasks by Phase

- **Phase 1 (Setup)**: 4 tasks
- **Phase 2 (Foundational)**: 12 tasks (11 implementation + 3 unit tests)
- **Phase 3 (US1 - Authentication)**: 12 tasks (8 implementation + 2 unit tests + 1 integration test)
- **Phase 4 (US2 - System Info)**: 3 tasks
- **Phase 5 (US3 - Network Info)**: 3 tasks
- **Phase 6 (US4 - Config Upload)**: 5 tasks
- **Phase 7 (US5 - Command Execution)**: 3 tasks
- **Phase 8 (US6 - Session Completion)**: 3 tasks
- **Phase 9 (Polish)**: 10 tasks

### Tasks by User Story

- **US1**: 12 tasks (SRP client, HTTP foundation, pass command)
- **US2**: 3 tasks (info command + API call)
- **US3**: 3 tasks (connections command + API call)
- **US4**: 5 tasks (load command + directory handling)
- **US5**: 3 tasks (command execution)
- **US6**: 3 tasks (complete command)

### Parallel Opportunities

- **Phase 1**: 3 tasks can run in parallel (T002, T003, T004)
- **Phase 2**: 9 tasks can run in parallel (T006-T013, T015-T016)
- **Phase 3 (US1)**: 5 SRP functions (T017-T021) can be developed in parallel
- **Phase 9 (Polish)**: 6 tasks can run in parallel (T046-T051)
- **Cross-Story**: US2-US6 can be developed in parallel once US1 session management is done

### Independent Test Criteria

- **US1**: `boarding pass` creates session token, stores it with 0600 permissions
- **US2**: `boarding info` displays system information in YAML (default) or JSON (-o json)
- **US3**: `boarding connections` displays network interfaces in YAML or JSON
- **US4**: `boarding load <dir>` uploads files, shows progress, succeeds or fails with clear errors
- **US5**: `boarding command "<cmd>"` executes command, displays output, handles errors
- **US6**: `boarding complete` terminates session, deletes token

### Suggested MVP Scope

**Minimum Viable Product**: Phases 1-3 only (Setup + Foundational + US1 Authentication)

This provides:
- Working authentication flow
- Session token management
- TLS certificate handling (TOFU)
- Foundation for all subsequent commands

**Time Estimate**: ~2-3 days for MVP (Phases 1-3), ~5-7 days for full implementation (all phases)

---

## Notes

- **[P] tasks**: Different files or independent functions - can be developed in parallel
- **[Story] label**: Maps task to specific user story for traceability and independent testing
- **File paths**: All tasks include exact file paths for implementation
- **Testing strategy**: Co-locate tests with source files (Go convention), table-driven tests, use existing testify/assert
- **Constitution compliance**: All tasks follow minimal dependencies (stdlib + gopkg.in/yaml.v3 only), FIPS 140-3 (stdlib crypto only)
- **Commit strategy**: Commit after completing each task or logical group of related tasks
- **Validation**: Stop at any checkpoint to test the user story independently before proceeding
- **Error handling**: Each command should have clear error messages (per SC-005 from spec)
- **Security**: Enforce 0600 permissions on session tokens, never log secrets, validate all inputs
