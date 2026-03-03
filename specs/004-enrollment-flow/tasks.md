# Tasks: Enrollment Configuration Wizard

**Input**: Design documents from `/specs/004-enrollment-flow/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/api-extensions.yaml

**Tests**: TDD approach — write tests FIRST, ensure they FAIL, then implement.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Update API contract, regenerate types, prepare project scaffolding

- [X] T001 Update OpenAPI spec: fix `/configure` description (remove sentinel mention), extend `SystemInfo` with `hostname`, extend `NetworkInterface` with `type`/`speed`/`carrier`/`driver`/`vendor`/`model`, extend `CommandRequest` with `params`, add `CompleteRequest` schema, extend `CompleteResponse` status enum in `specs/001-boardingpass-api/contracts/openapi.yaml`
- [X] T002 Regenerate TypeScript types from updated OpenAPI spec via `make generate-app`
- [X] T003 Create mobile app scaffold directories: `mobile/src/components/ConfigWizard/`, `mobile/src/services/api/` (if not existing), `mobile/src/utils/`, `mobile/src/types/`

---

## Phase 2: Foundational (Service-Side API Extensions + CLI)

**Purpose**: Extend the BoardingPass service API and CLI to support all wizard operations. MUST complete before any mobile app user story work begins.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete. API changes are accepted when service and CLI end-to-end tests pass.

### Protocol types

- [X] T004 Extend `CommandRequest` with `Params []string` field, add `CompleteRequest` struct with `Reboot bool`, extend `CompleteResponse.Status` enum to include `"rebooting"`, add `Hostname string` to `SystemInfo`, add `Type`/`Speed`/`Carrier`/`Driver`/`Vendor`/`Model` fields to `NetworkInterface` in `pkg/protocol/types.go`

### Tests: Hostname in /info

- [X] T005 [P] Write unit test for `GetHostname()` in `internal/inventory/hostname_test.go`
- [X] T006 [P] Write integration test for `/info` endpoint verifying `hostname` field is present in response in `tests/integration/info_test.go`

### Implementation: Hostname in /info

- [X] T007 [P] Create `internal/inventory/hostname.go` with `GetHostname() (string, error)` using `os.Hostname()`
- [X] T008 Modify `/info` handler to include hostname from `GetHostname()` in `SystemInfo` response in `internal/api/handlers/info.go`

### Tests: Network interface metadata

- [X] T009 [P] Write unit tests for sysfs-based interface type detection, speed, carrier, driver, vendor, and model reading (including hwdata pci.ids lookup when available and hex ID fallback) in `internal/network/interfaces_test.go`

### Implementation: Network interface metadata

- [X] T010 Extend `ListInterfaces()` to read sysfs for each interface: detect type (`/sys/class/net/<name>/wireless/`, `/sys/class/net/<name>/bridge/`, etc.), read speed (`/sys/class/net/<name>/speed`), read carrier (`/sys/class/net/<name>/carrier`), read driver (symlink basename of `/sys/class/net/<name>/device/driver`), read vendor/device PCI IDs from `/sys/class/net/<name>/device/vendor` and `/sys/class/net/<name>/device/device`, look up human-readable names in `/usr/share/hwdata/pci.ids` if present (fall back to hex IDs) in `internal/network/interfaces.go`

### Tests: Command parameter support

- [X] T011 [P] Write unit tests for `max_params` validation: reject if `len(params) > max_params`, reject empty param strings, enforce 1024-char max per param in `internal/command/allowlist_test.go`
- [X] T012 [P] Write unit tests for executor appending `--` separator followed by params in `internal/command/executor_test.go`
- [X] T013 [P] Write integration test for `/command` endpoint with params in `tests/integration/command_test.go`

### Implementation: Command parameter support

- [X] T014 [P] Add `MaxParams int` field to `CommandDefinition` in `internal/config/config.go`
- [X] T015 Add `max_params` validation to allowlist lookup in `internal/command/allowlist.go`
- [X] T016 Modify executor to append `--` separator followed by params to the command args in `internal/command/executor.go`
- [X] T017 Modify `/command` handler to parse `params` from `CommandRequest` and pass to executor in `internal/api/handlers/command.go`

### Tests: Complete with reboot

- [X] T018 [P] Write unit test for `ScheduleReboot(delay)` in `internal/lifecycle/reboot_test.go`
- [X] T019 [P] Write integration test for `/complete` with `reboot: true` returning `"rebooting"` status in `tests/integration/complete_test.go`

### Implementation: Complete with reboot

- [X] T020 [P] Create `internal/lifecycle/reboot.go` with `ScheduleReboot(delay time.Duration)` that execs `systemctl reboot` after delay
- [X] T021 Modify `/complete` handler to accept optional `CompleteRequest` body, return `"rebooting"` status when `reboot: true`, call `ScheduleReboot(3s)` after sending response in `internal/api/handlers/complete.go`

### Regenerate mocks

- [X] T022 Regenerate Go mocks via `make generate-service` (if any interfaces changed)

### CLI updates

- [X] T023 Update CLI `info` command to display `hostname` field in output in `internal/cli/commands/info.go`
- [X] T024 [P] Update CLI `connections` command to display `type`, `speed`, `carrier`, `driver`, `vendor`, `model` columns in output in `internal/cli/commands/connections.go`
- [X] T025 [P] Update CLI `command` subcommand to accept and send `params` (as positional args after command ID) in `internal/cli/commands/command.go`
- [X] T026 [P] Update CLI `complete` subcommand to accept `--reboot` flag and send `CompleteRequest` body in `internal/cli/commands/complete.go`

### Helper scripts

- [X] T027 [P] Create `build/scripts/wifi-scan.sh` — nmcli wrapper that outputs JSON array of WiFi networks (device, ssid, bssid, signal, security, channel, frequency, rate)
- [X] T028 [P] Create `build/scripts/reload-connection.sh` — takes connection name as param, runs `nmcli connection reload` then `nmcli connection up <name>`
- [X] T029 [P] Create `build/scripts/connectivity-test.sh` — takes interface name and gateway IP as params, tests IP assignment + gateway ping + DNS resolution + internet reachability, outputs JSON result
- [X] T030 [P] Create `build/scripts/enroll-insights.sh` — reads `/etc/boardingpass/staging/insights.json`, runs `rhc connect --organization <ORG_ID> --activation-key <KEY_NAME>`, deletes staging file
- [X] T031 [P] Create `build/scripts/enroll-flightctl.sh` — reads `/etc/boardingpass/staging/flightctl.json`, runs `flightctl login <URL> --username <USERNAME> --password <PASSWORD>`, deletes staging file

### Service configuration

- [X] T032 Add new command definitions (wifi-scan, set-hostname, reload-connection, restart-chronyd, connectivity-test, enroll-insights, enroll-flightctl) with `max_params` to `build/config.yaml`
- [X] T033 [P] Add sudoers entries for all new scripts and commands (including `rhc connect` and `flightctl login`) to `build/boardingpass.sudoers`
- [X] T034 Add path allow-list entries (`/etc/hostname`, `/etc/NetworkManager/system-connections/`, `/etc/chrony.d/`, `/etc/profile.d/`, `/etc/boardingpass/staging/`, `/etc/systemd/system/`) to `build/config.yaml`

### Contract tests

- [X] T035 Update contract tests to validate new OpenAPI schema fields (hostname in SystemInfo, type/speed/carrier/driver/vendor/model in NetworkInterface, params in CommandRequest, CompleteRequest schema) in `tests/contract/`

### End-to-end validation

- [X] T036 Run `make lint-service && make test-service` — fix all failures
- [X] T037 Run CLI integration/e2e tests (`tests/cli-integration/`, `tests/cli-e2e/`) — fix all failures

**Checkpoint**: Service API and CLI fully extended. All service and CLI tests pass.

---

## Phase 3: User Story 1 — Device Configuration Wizard (Priority: P1) 🎯 MVP

**Goal**: Deliver the 5-step wizard UI with navigation, validation, and pre-population — without apply logic (apply is US2)

**Independent Test**: Navigate through all 5 steps with valid input, see validation errors for invalid input, navigate backward with data preserved

### Mobile types and utilities

- [X] T038 [P] [US1] Create wizard TypeScript types (WizardState, HostnameConfig, InterfaceConfig, WiFiConfig, AddressingConfig, IPv4Config, IPv6Config, ServicesConfig, NTPConfig, ProxyConfig, EnrollmentConfig, InsightsConfig, FlightControlConfig, ApplyStatus, ConnectivityResult, WiFiNetwork) in `mobile/src/types/wizard.ts`

### Tests: Validation utilities

- [X] T039 [P] [US1] Write unit tests for network validation utilities (validateHostname, validateIPv4, validateIPv6, validateSubnetMask, validateGatewayInSubnet, validatePort, validateHttpsUrl, validateNtpServer) in `mobile/tests/unit/utils/network-validation.test.ts`

### Tests: NM connection builder

- [X] T040 [P] [US1] Write unit tests for NM connection file builder (Ethernet, WiFi, VLAN, all IPv4/IPv6 combinations) in `mobile/tests/unit/utils/nm-connection.test.ts`

### Tests: Wizard hook

- [X] T041 [P] [US1] Write unit tests for useConfigWizard hook (step validation, navigation guards, step completion checks) in `mobile/tests/unit/hooks/useConfigWizard.test.ts`

### Tests: Wizard integration

- [X] T042 [P] [US1] Write integration test for full wizard navigation flow (forward/backward, validation blocking, data preservation) in `mobile/tests/integration/config-wizard.test.ts`

### Implementation: Utilities

- [X] T043 [P] [US1] Create network validation utilities (validateHostname RFC 1123, validateIPv4, validateIPv6, validateSubnetMask, validateGatewayInSubnet, validatePort, validateHttpsUrl, validateNtpServer) in `mobile/src/utils/network-validation.ts`
- [X] T044 [P] [US1] Create NetworkManager connection file builder (generateNmConnection for Ethernet/WiFi/VLAN, with IPv4/IPv6 config sections) in `mobile/src/utils/nm-connection.ts`

### Implementation: API service wrappers

- [X] T045 [P] [US1] Create `POST /configure` wrapper (sends file bundle, returns success/error) in `mobile/src/services/api/configure.ts`
- [X] T046 [P] [US1] Create `POST /command` wrapper (sends command ID + optional params, returns stdout/stderr/exit_code) in `mobile/src/services/api/command.ts`
- [X] T047 [P] [US1] Create `POST /complete` wrapper (sends optional reboot flag, returns status) in `mobile/src/services/api/complete.ts`

### Implementation: Wizard state management

- [X] T048 [US1] Create WizardContext with useReducer: actions for setStep, updateHostname, updateInterface, updateAddressing, updateServices, updateEnrollment, setApplyMode, setApplyStatus, reset; provider wraps wizard screen in `mobile/src/contexts/WizardContext.tsx`

### Implementation: Step components

- [X] T049 [P] [US1] Create StepIndicator component showing current step (1–5), step labels, and completion progress in `mobile/src/components/ConfigWizard/StepIndicator.tsx`
- [X] T050 [P] [US1] Create HostnameStep component: TextInput pre-populated from device info, RFC 1123 validation, inline error display in `mobile/src/components/ConfigWizard/HostnameStep.tsx`
- [X] T051 [P] [US1] Create InterfaceStep component: DataTable of interfaces (name, type, MAC, vendor, model, speed, state/carrier), radio selection, optional VLAN ID input (1–4094), highlight service interface in `mobile/src/components/ConfigWizard/InterfaceStep.tsx`
- [X] T052 [P] [US1] Create AddressingStep component: IPv4 radio (DHCP/Static) with conditional fields (address, subnet, gateway), DNS auto checkbox with conditional DNS fields; IPv6 radio (DHCP/Static/Disabled) with same pattern in `mobile/src/components/ConfigWizard/AddressingStep.tsx`
- [X] T053 [P] [US1] Create ServicesStep component: NTP radio (Automatic/Manual) with manual server list input, optional proxy section (hostname, port, optional username/password) in `mobile/src/components/ConfigWizard/ServicesStep.tsx`
- [X] T054 [P] [US1] Create EnrollmentStep component: Insights toggle with endpoint (default URL), Org ID, Activation Key fields; Flight Control toggle with endpoint, username, password fields in `mobile/src/components/ConfigWizard/EnrollmentStep.tsx`

### Implementation: Wizard container and screen

- [X] T055 [US1] Create WizardContainer: renders current step component, StepIndicator, Next/Back buttons, validates step before forward navigation, tracks maxReachedStep in `mobile/src/components/ConfigWizard/WizardContainer.tsx`
- [X] T056 [US1] Create useConfigWizard hook: encapsulates step validation logic, per-step data access, navigation guards, step completion checks in `mobile/src/hooks/useConfigWizard.ts`
- [X] T057 [US1] Create wizard screen at `mobile/app/device/configure.tsx`: wraps WizardContainer in WizardContext provider, fetches device info + network data on mount, passes to context as initial state
- [X] T058 [US1] Add "Configure" button to device detail screen that navigates to `device/configure` in `mobile/app/device/[id].tsx`

### Checkpoint validation

- [X] T059 [US1] Run `make lint-app && make test-unit-app` — fix all failures

**Checkpoint**: Full 5-step wizard navigable with validation. No apply logic yet.

---

## Phase 4: User Story 2 — Smart Configuration Application (Priority: P1)

**Goal**: Implement immediate and deferred apply modes with per-step feedback and review page

**Independent Test**: With different-interface: verify per-step apply with success/failure feedback. With same-interface: verify review page and atomic apply + reboot.

### Tests: Apply mode detection and immediate apply

- [ ] T060 [P] [US2] Write unit tests for apply mode detection (compare connection target IP against device interface IPs) and per-step immediate apply logic in `mobile/tests/unit/hooks/useConfigWizard.test.ts` (extend existing)

### Tests: Deferred apply and review

- [ ] T061 [P] [US2] Write integration test for deferred apply flow: all config files bundled into single `/configure` call, then `/complete` with `reboot: true` in `mobile/tests/integration/config-wizard.test.ts` (extend existing)

### Implementation: Apply mode detection

- [ ] T062 [US2] Add service interface detection: compare app's connection target IP against device interface IPs to determine serviceInterfaceName; set applyMode in WizardContext when interface is selected in Step 2 in `mobile/src/hooks/useConfigWizard.ts`

### Implementation: Immediate apply (different interface)

- [ ] T063 [US2] Implement per-step immediate apply in useConfigWizard: after step completion, call `/configure` with step's config files then `/command` with step's apply command, update stepApplyStatus in context in `mobile/src/hooks/useConfigWizard.ts`
- [ ] T064 [P] [US2] Create ApplyFeedback component: shows per-step apply status (spinner during apply, success checkmark, error with retry button) in `mobile/src/components/ConfigWizard/ApplyFeedback.tsx`

### Implementation: Connectivity verification

- [ ] T065 [US2] After network addressing apply (Step 3), invoke `connectivity-test` command with interface name and gateway IP, display ConnectivityResult (IP assigned, gateway reachable, DNS resolves, internet reachable) in ApplyFeedback in `mobile/src/hooks/useConfigWizard.ts`

### Implementation: Deferred apply (same interface)

- [ ] T066 [US2] Implement deferred apply: collect all config files across steps, generate NM connection file, NTP config, proxy config, enrollment staging files, systemd oneshot (if enrollment enabled), send entire bundle via single `/configure` call, then call `/complete` with `reboot: true` in `mobile/src/hooks/useConfigWizard.ts`
- [ ] T067 [P] [US2] Create ReviewPage component: human-readable summary of all queued changes (hostname, interface, addressing, services, enrollment), "Confirm & Reboot" and "Back" buttons in `mobile/src/components/ConfigWizard/ReviewPage.tsx`

### Implementation: Post-apply UX

- [ ] T068 [US2] Handle post-apply states: in immediate mode call `/complete` (graceful shutdown) after final step; in deferred mode show "Device is rebooting" message after confirm, return to device list after timeout in `mobile/src/components/ConfigWizard/WizardContainer.tsx`

### Checkpoint validation

- [ ] T069 [US2] Run `make lint-app && make test-app` — fix all failures

**Checkpoint**: Both apply modes functional.

---

## Phase 5: User Story 3 — WiFi Network Selection (Priority: P2)

**Goal**: Enable WiFi interface selection with SSID scanning, selection, and password entry

**Independent Test**: Select a WiFi interface, see scanned SSIDs, select a secured network, enter password

### Tests

- [ ] T070 [P] [US3] Write unit tests for WiFiStep component (scan trigger, SSID table rendering, password field visibility, rescan, empty state) in `mobile/tests/unit/components/WiFiStep.test.ts`
- [ ] T071 [P] [US3] Write unit tests for WiFi NM connection file generation (wifi + wifi-security sections with WPA-PSK/SAE) in `mobile/tests/unit/utils/nm-connection.test.ts` (extend existing)

### Implementation

- [ ] T072 [P] [US3] Create WiFiStep component: triggers wifi-scan command on mount, displays SSID table (SSID, signal, security, channel, band, rate), radio selection, conditional password field for secured networks, Rescan button, empty-state message in `mobile/src/components/ConfigWizard/WiFiStep.tsx`
- [ ] T073 [US3] Integrate WiFiStep into WizardContainer: show WiFiStep as sub-step after InterfaceStep when WiFi interface is selected, include WiFi config in NM connection file generation in `mobile/src/components/ConfigWizard/WizardContainer.tsx`
- [ ] T074 [US3] Extend nm-connection.ts to generate WiFi-specific NM connection file sections (wifi, wifi-security with WPA-PSK/SAE) in `mobile/src/utils/nm-connection.ts`

### Checkpoint validation

- [ ] T075 [US3] Run `make lint-app && make test-unit-app` — fix all failures

**Checkpoint**: WiFi flow works end-to-end.

---

## Phase 6: User Story 4 — Enrollment Server Registration (Priority: P2)

**Goal**: Enable optional enrollment with Red Hat Insights (`rhc connect`) and Flight Control (`flightctl login`) via staging files and enrollment commands

**Independent Test**: Enable Insights, enter credentials, verify staging file is included in apply bundle and enrollment command is executed

### Tests

- [ ] T076 [P] [US4] Write unit tests for enrollment staging file generation (insights.json with org_id/activation_key, flightctl.json with endpoint/username/password, mode 0600) in `mobile/tests/unit/hooks/useConfigWizard.test.ts` (extend existing)
- [ ] T077 [P] [US4] Write unit tests for systemd oneshot service unit generation (boardingpass-enroll.service with both ExecStart lines, ConditionPathExists, enablement symlink) in `mobile/tests/unit/hooks/useConfigWizard.test.ts` (extend existing)

### Implementation

- [ ] T078 [US4] Implement enrollment staging file generation in useConfigWizard: build `insights.json` and/or `flightctl.json` staging file content from EnrollmentConfig, include in `/configure` bundle with mode 0600 in `mobile/src/hooks/useConfigWizard.ts`
- [ ] T079 [US4] Implement enrollment command execution in immediate apply: after staging files are written via `/configure`, invoke `enroll-insights` and/or `enroll-flightctl` commands, display success/failure in ApplyFeedback in `mobile/src/hooks/useConfigWizard.ts`
- [ ] T080 [US4] Implement deferred enrollment: generate systemd oneshot service unit (`boardingpass-enroll.service` with ExecStart for both `enroll-insights.sh` and `enroll-flightctl.sh`) and enablement symlink, include in deferred `/configure` bundle in `mobile/src/hooks/useConfigWizard.ts`

### Checkpoint validation

- [ ] T081 [US4] Run `make lint-app && make test-unit-app` — fix all failures

**Checkpoint**: Enrollment registration works in both apply modes.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final quality, end-to-end validation, and security verification

- [ ] T082 Run `make lint-all` and fix all errors
- [ ] T083 Run `make test-all` and fix all failures (service + CLI + app, all test types)
- [ ] T084 Validate quickstart.md scenarios manually (both immediate and deferred API call flows)
- [ ] T085 Verify credential security: confirm WiFi passwords, proxy passwords, activation keys, and enrollment passwords use secure text input, are not logged, and are cleared on wizard close
- [ ] T086 Run contract tests to verify mobile app and service types align with updated OpenAPI spec

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (updated OpenAPI spec) — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 (service API and CLI must be extended and tested)
- **US2 (Phase 4)**: Depends on Phase 3 (wizard UI must exist before apply logic)
- **US3 (Phase 5)**: Depends on Phase 3 (wizard container must exist); can run in parallel with US2
- **US4 (Phase 6)**: Depends on Phase 3 (EnrollmentStep exists); can run in parallel with US2 and US3
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Blocked by Foundational (Phase 2) — no dependencies on other stories
- **US2 (P1)**: Depends on US1 (wizard UI) — adds apply logic to existing components
- **US3 (P2)**: Depends on US1 (WizardContainer) — can run in parallel with US2
- **US4 (P2)**: Depends on US1 (EnrollmentStep shell) — can run in parallel with US2 and US3

### Within Each Phase (TDD Order)

- Tests FIRST → ensure they FAIL
- Types/utilities before components
- Components before container/hook integration
- API wrappers before apply logic
- Run checkpoint validation before moving to next phase

### Parallel Opportunities

- **Phase 2**: T005+T006, T009, T011+T012+T013, T018+T019 (test groups) can run in parallel; T027–T031 (scripts) can all run in parallel; T023–T026 (CLI updates) can run in parallel
- **Phase 3**: T039–T042 (all test tasks) can run in parallel; T043–T047 (utils/wrappers) can run in parallel; T049–T054 (step components) can run in parallel
- **Phase 5 + Phase 6**: Can run in parallel with each other and with Phase 4

---

## Parallel Example: Phase 2 (Foundational Tests)

```bash
# Launch all test tasks in parallel (different test files):
Task: "Write unit test for GetHostname() in internal/inventory/hostname_test.go"
Task: "Write unit tests for sysfs interface detection in internal/network/interfaces_test.go"
Task: "Write unit tests for max_params validation in internal/command/allowlist_test.go"
Task: "Write unit tests for executor param appending in internal/command/executor_test.go"
Task: "Write unit test for ScheduleReboot in internal/lifecycle/reboot_test.go"
```

## Parallel Example: Phase 3 (US1 Tests + Types)

```bash
# Launch all US1 tests in parallel:
Task: "Write validation utils tests in mobile/tests/unit/utils/network-validation.test.ts"
Task: "Write NM connection builder tests in mobile/tests/unit/utils/nm-connection.test.ts"
Task: "Write useConfigWizard hook tests in mobile/tests/unit/hooks/useConfigWizard.test.ts"
Task: "Write wizard integration test in mobile/tests/integration/config-wizard.test.ts"

# Then launch all implementation utilities in parallel:
Task: "Create validation utils in mobile/src/utils/network-validation.ts"
Task: "Create NM connection builder in mobile/src/utils/nm-connection.ts"
Task: "Create /configure wrapper in mobile/src/services/api/configure.ts"
Task: "Create /command wrapper in mobile/src/services/api/command.ts"
Task: "Create /complete wrapper in mobile/src/services/api/complete.ts"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup (OpenAPI + type gen)
2. Complete Phase 2: Foundational (service API + CLI extensions, tests pass)
3. Complete Phase 3: US1 — Core Wizard (tests first, then implementation)
4. **STOP and VALIDATE**: Navigate all 5 steps, test validation, test data preservation on back nav
5. Deploy/demo — wizard works but doesn't apply changes yet

### Incremental Delivery

1. Setup + Foundational → Service API + CLI ready, all e2e tests pass
2. Add US1 → Wizard UI navigable → Demo (MVP!)
3. Add US2 → Apply logic works for both modes → Demo
4. Add US3 → WiFi networks selectable → Demo
5. Add US4 → Enrollment registration functional → Demo
6. Polish → Lint clean, all tests pass, security verified

### Parallel Team Strategy

With two developers:

1. Both complete Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US1 → US2 (sequential — US2 depends on US1)
   - Developer B: US3 + US4 (after US1 components are available)
3. Both do Polish phase together

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- **TDD**: Write tests FIRST in each phase, verify they fail, then implement
- **CLI updates required**: Service API changes must be reflected in the CLI; API changes are accepted when service AND CLI e2e tests pass
- Run `make lint-all && make test-all` at each checkpoint
- Service-side changes (Phase 2) can be committed and tested independently of mobile app changes
- Enrollment credentials use staging files, NOT command params (per RD-10 — keeps them off process table)
- **Enrollment commands**: `rhc connect --organization <ORG_ID> --activation-key <KEY_NAME>` for Insights; `flightctl login <URL> --username <USERNAME> --password <PASSWORD>` for Flight Control
