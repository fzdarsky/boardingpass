# Tasks: Review & Apply Step

**Input**: Design documents from `/specs/005-review-apply-step/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Test tasks included for new modules (action-list.ts, clock.go) as these are core logic with testable pure functions.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Update contracts and shared protocol types that all stories depend on

- [x] T001 Update OpenAPI spec: add `system_time` (string, format: date-time) and `clock_synchronized` (boolean) to OSInfo schema required fields and properties in specs/001-boardingpass-api/contracts/openapi.yaml
- [x] T002 [P] Add `SystemTime string` and `ClockSynchronized bool` fields with JSON tags to `OSInfo` struct in pkg/protocol/types.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared types, state extensions, and step indicator changes that MUST be complete before any user story

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Add `PlannedAction` interface (id, description, category, status, detail, step, infoOnly fields), change `TOTAL_STEPS` from 5 to 6, add `STEP_LABELS[6] = 'Review'`, and export updated constants in mobile/src/types/wizard.ts
- [x] T004 Add `actionList: PlannedAction[]` and `applyInProgress: boolean` to `WizardState`, add `SET_ACTION_LIST`, `SET_APPLY_IN_PROGRESS`, and `UPDATE_ACTION_STATUS` reducer actions in mobile/src/contexts/WizardContext.tsx
- [x] T005 [P] Update StepIndicator to display 6 steps with the 6th step labeled "Review" in mobile/src/components/ConfigWizard/StepIndicator.tsx
- [x] T006 Regenerate TypeScript API types from updated OpenAPI spec by running `make generate-app` to update mobile/src/types/api.ts

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 3 - Remove Per-Step Apply (Priority: P1) 🎯 MVP

**Goal**: Replace "Apply & Next" with "Next" in all wizard steps. No configuration is applied during step navigation except hostname (safe, no connectivity impact).

**Independent Test**: Navigate through the wizard in immediate mode — all steps should show "Next" button, no API calls to /configure or /command during navigation (except hostname at Step 1).

**Note**: This story alone will temporarily break the wizard's ability to apply configuration. It is a prerequisite for US1 which restores apply functionality in the Review & Apply step.

### Implementation for User Story 3

- [x] T007 [US3] Simplify WizardContainer handleNext: remove `willApplyOnNext` logic and per-step `applyStepImmediate` calls for steps 2–5 in immediate mode, change button text from "Apply & Next" to "Next" for all non-review steps, keep hostname apply at Step 1, update last-step handling to navigate to Step 6 (Review) instead of calling /complete directly in mobile/src/components/ConfigWizard/WizardContainer.tsx

**Checkpoint**: Wizard navigates through all 6 steps with "Next" button; no per-step apply occurs (except hostname)

---

## Phase 4: User Story 4 - System Time and Clock Sync (Priority: P2)

**Goal**: Extend the service's /info endpoint with system time and clock synchronization status. Display these in the Device Details screen's Operating System section.

**Independent Test**: Connect to a device, view Device Details — the OS section shows current system time and whether the clock is synchronized via NTP.

### Implementation for User Story 4

- [x] T008 [P] [US4] Create `GetClockStatus()` function that parses `timedatectl show` output for `NTPSynchronized` and `TimeUSec` properties, returns `(time.Time, bool, error)` in internal/inventory/clock.go
- [x] T009 [P] [US4] Create table-driven unit tests for `GetClockStatus()` covering: synchronized, not synchronized, timedatectl unavailable (fallback to current time + false), and malformed output in internal/inventory/clock_test.go
- [x] T010 [US4] Call `GetClockStatus()` in `gatherSystemInfo()` and populate `info.OS.SystemTime` (formatted as RFC 3339 UTC) and `info.OS.ClockSynchronized` in internal/api/handlers/info.go
- [x] T011 [US4] Add system time display (formatted to locale) and clock sync status badge (SYNCED/NOT SYNCED with green/orange coloring) to the Operating System section, below the FIPS badge in mobile/src/components/DeviceInfo/SystemInformationCard.tsx

**Checkpoint**: Device Details page shows system time and clock sync status; /info endpoint returns new fields

---

## Phase 5: User Story 5 - Action List Generation (Priority: P2)

**Goal**: Create a pure function that generates a human-readable, ordered action list from wizard state. This list is displayed on the Review & Apply screen and drives the execution flow.

**Independent Test**: Call `buildActionList()` with various wizard state configurations and verify the output matches expected action descriptions and ordering.

### Implementation for User Story 5

- [x] T012 [US5] Create `buildActionList(state: WizardState, applyMode: 'immediate' | 'deferred'): PlannedAction[]` pure function implementing all 12 action generation rules: (1) hostname keep/set, (2) interface selection, (3) WiFi connection (if WiFi), (4) IPv4 DHCP/static, (5) DNS servers (if manual), (6) IPv6 config (if not disabled), (7) connectivity check (immediate only), (8) DNS resolution check (immediate only), (9) NTP auto/manual, (10) clock sync wait, (11) Insights enrollment (if configured), (12) Flight Control enrollment (if configured). Each action has id, description, category, step number, and infoOnly flag in mobile/src/utils/action-list.ts
- [x] T013 [US5] Create table-driven unit tests for `buildActionList()` with test cases: all-defaults (DHCP, no enrollment), static IPv4 with manual DNS, WiFi with WPA2, hostname changed vs unchanged, Insights only, Flight Control only, both enrollments, IPv6 disabled vs DHCP vs static, manual NTP servers, immediate vs deferred mode (check/wait actions differ) in mobile/tests/unit/utils/action-list.test.ts

**Checkpoint**: `buildActionList()` generates correct action lists for all wizard state permutations; all unit tests pass

---

## Phase 6: User Story 1 - Immediate Mode Review & Apply (Priority: P1)

**Goal**: Create the ReviewApplyPage component that displays the action list and executes actions sequentially in immediate mode with per-action visual feedback (spinner → checkmark/error).

**Independent Test**: Complete wizard in immediate mode (different enrollment and service interfaces), reach Review & Apply screen, tap "Apply" — actions execute one by one with status icons, /complete called on success, "Provisioning Complete" terminal state shown.

### Implementation for User Story 1

- [x] T014 [US1] Add `applyAllImmediate(client: APIClient): Promise<void>` to `useConfigWizard` hook: generates action list via `buildActionList()`, iterates through actions grouped by step, for each step group sends config files via /configure and executes commands via /command, runs check/wait actions individually (connectivity-test command, clock sync polling via /info), updates per-action status (running/success/failed) in context, halts on first failure marking remaining as skipped, calls /complete with reboot=false on success in mobile/src/hooks/useConfigWizard.ts
- [x] T015 [US1] Create ReviewApplyPage component: scrollable numbered list of PlannedAction items with category icons (gear/terminal/magnifying-glass/clock), per-action status indicators (pending=gray dot, running=spinner, success=green checkmark, failed=red X with error detail, skipped=gray dash), "Apply" button (disabled during execution, loading state), "Back" button to return to Step 5, error detail expandable below failed actions in mobile/src/components/ConfigWizard/ReviewApplyPage.tsx
- [x] T016 [US1] Wire ReviewApplyPage into WizardContainer: add `WIZARD_STEPS.REVIEW` case to `renderStep()`, when Step 5 "Next" is tapped navigate to Step 6, pass `applyAllImmediate` and `onComplete` callbacks to ReviewApplyPage, show "Provisioning Complete" terminal state after successful apply, remove old `showReview` state and `ReviewPage` usage in mobile/src/components/ConfigWizard/WizardContainer.tsx

**Checkpoint**: Full immediate mode flow works end-to-end: wizard → review → apply with per-action feedback → provisioning complete

---

## Phase 7: User Story 2 - Deferred Mode Review & Apply (Priority: P1)

**Goal**: Extend ReviewApplyPage to handle deferred mode: show "Apply & Reboot" button, send atomic config bundle, trigger device reboot.

**Independent Test**: Complete wizard in deferred mode (same enrollment and service interfaces), reach Review & Apply screen, tap "Apply & Reboot" — config bundle sent, device reboots, "Device is Rebooting" terminal state shown.

### Implementation for User Story 2

- [x] T017 [US2] Add deferred mode handling to ReviewApplyPage: when `applyMode === 'deferred'` render "Apply & Reboot" button with reboot icon, mark check/wait actions as informational (gray text, no execution), on tap call existing `applyDeferred()` to send atomic bundle + `completeProvisioning(client, true)`, pass reboot terminal state back to WizardContainer in mobile/src/components/ConfigWizard/ReviewApplyPage.tsx

**Checkpoint**: Both immediate and deferred modes work end-to-end through the same ReviewApplyPage component

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Cleanup, validation, and quality assurance

- [x] T018 Delete old ReviewPage component file (fully replaced by ReviewApplyPage) and remove any remaining imports in mobile/src/components/ConfigWizard/ReviewPage.tsx
- [x] T019 Run `make lint-all` and fix all lint errors across service and mobile app
- [x] T020 Run `make test-all` and fix all test failures across service and mobile app

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **US3 (Phase 3)**: Depends on Foundational — BLOCKS US1 (removes per-step apply)
- **US4 (Phase 4)**: Depends on Foundational — independent of US3/US5/US1/US2
- **US5 (Phase 5)**: Depends on Foundational — BLOCKS US1 (generates action list)
- **US1 (Phase 6)**: Depends on US3 + US5 — first complete end-to-end flow
- **US2 (Phase 7)**: Depends on US1 — extends ReviewApplyPage with deferred mode
- **Polish (Phase 8)**: Depends on all stories being complete

### User Story Dependencies

```text
Setup → Foundational ──┬── US3 (remove per-step apply) ──┐
                        │                                  ├── US1 (immediate) → US2 (deferred) → Polish
                        ├── US5 (action list generation) ──┘
                        │
                        └── US4 (clock sync) ─────────────────────────────────────────────────→ Polish
```

- **US3 (P1)**: Prerequisite for US1. Cannot be tested independently (breaks wizard apply).
- **US4 (P2)**: Fully independent. Can be implemented in parallel with US3/US5/US1.
- **US5 (P2)**: Prerequisite for US1. Can be tested independently via unit tests.
- **US1 (P1)**: Requires US3 + US5. First complete testable flow.
- **US2 (P1)**: Requires US1. Extends the same ReviewApplyPage component.

### Within Each User Story

- Types/models before services/hooks
- Services/hooks before components
- Components before integration/wiring
- Tests can be written alongside implementation (same phase)

### Parallel Opportunities

- T001 + T002 can run in parallel (different files: OpenAPI spec vs Go types)
- T005 can run in parallel with T003/T004 (StepIndicator vs wizard types/context)
- T008 + T009 can run in parallel (clock.go vs clock_test.go)
- US4 (Phase 4) can run entirely in parallel with US3 (Phase 3) and US5 (Phase 5)

---

## Parallel Example: Phase 4 (US4)

```text
# These can run in parallel (different files):
Task T008: "Create GetClockStatus() in internal/inventory/clock.go"
Task T009: "Create unit tests in internal/inventory/clock_test.go"

# Then sequentially:
Task T010: "Wire into info handler in internal/api/handlers/info.go"
Task T011: "Display in SystemInformationCard in mobile/src/components/DeviceInfo/SystemInformationCard.tsx"
```

---

## Implementation Strategy

### MVP First (US3 + US5 + US1)

1. Complete Phase 1: Setup (contracts + types)
2. Complete Phase 2: Foundational (wizard state + step indicator)
3. Complete Phase 3: US3 — remove per-step apply
4. Complete Phase 5: US5 — action list generation
5. Complete Phase 6: US1 — ReviewApplyPage with immediate mode
6. **STOP and VALIDATE**: Test immediate mode end-to-end
7. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. US3 + US5 + US1 → Immediate mode works (MVP!)
3. US2 → Deferred mode works (full feature)
4. US4 → Clock sync in device details (enhancement)
5. Polish → Lint + test clean

### Independent Work Streams

Two developers can work in parallel:

- **Developer A**: US3 → US5 → US1 → US2 (main wizard flow)
- **Developer B**: US4 (clock sync — fully independent)

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- US4 is the only truly independent story — all others have chain dependencies
- T007 (US3) temporarily breaks wizard apply; functionality restored by T016 (US1)
- The old ReviewPage.tsx is deleted in T018 after the new ReviewApplyPage is fully wired
- Commit after each completed phase or logical task group
