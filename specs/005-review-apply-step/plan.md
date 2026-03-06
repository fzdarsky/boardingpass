# Implementation Plan: Review & Apply Step

**Branch**: `005-review-apply-step` | **Date**: 2026-03-05 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/005-review-apply-step/spec.md`

## Summary

Add a "Review & Apply" screen (Step 6) to the configuration wizard that replaces the current per-step apply behavior. Users now navigate through Steps 1–5 with only a "Next" button, then review all planned actions on a single screen before applying. In immediate mode, actions execute sequentially with real-time feedback; in deferred mode, configuration is sent atomically with a reboot. Additionally, extend the service's `/info` endpoint with system time and clock synchronization status, displayed on the Device Details screen.

## Technical Context

**Language/Version**: Go 1.25+ (service), TypeScript 5.x with React Native 0.74+ (mobile app)
**Primary Dependencies**: Go stdlib, `gopkg.in/yaml.v3` (service); React Native, React Native Paper, Expo Router, Axios (app)
**Storage**: N/A — wizard state is ephemeral (in-memory), config files written to `/etc/` via existing provisioning
**Testing**: Go `testing` + `testify/assert` (service); Jest + React Native Testing Library (app)
**Target Platform**: RHEL 9+ / Linux (service); iOS 15+ / Android (app)
**Project Type**: Service + mobile app (multi-component)
**Performance Goals**: Apply sequence completes within 60 seconds for 5–12 actions; UI updates at 60 FPS during apply
**Constraints**: No new runtime dependencies (service); FIPS 140-3 compliance maintained; single binary < 10MB
**Scale/Scope**: ~8 files modified, ~3 new files, ~500 lines added/modified across service and app

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --------- | ------ | ----- |
| I. Frictionless Bootstrapping | PASS | Review step improves user confidence without adding friction — single "Apply" replaces multiple "Apply & Next" clicks |
| II. Ephemeral & Fail-Safe | PASS | Wizard state remains ephemeral. Failed actions halt cleanly without partial application in immediate mode. Deferred mode uses existing atomic bundle. |
| III. Minimal Footprint | PASS | No new binaries, no new runtime processes. Clock detection uses existing `timedatectl`. |
| IV. Minimal Dependencies | PASS | No new dependencies on either service or app side. Clock info uses Go stdlib `os/exec` + `time`. |
| V. Transport Agnostic | PASS | Uses existing REST API endpoints (/configure, /command, /complete, /info). No transport-specific changes. |
| VI. Open Source & Permissive | PASS | No new dependency licensing concerns. |
| Security: Input Validation | PASS | No new external inputs. Action list is generated from already-validated wizard state. |
| Security: Secrets Management | PASS | Enrollment credentials are already handled securely. No changes to secret handling. |

**Post-Phase 1 re-check**: All gates still pass. The OSInfo schema extension adds two read-only fields (system_time, clock_synchronized) with no security implications.

## Project Structure

### Documentation (this feature)

```text
specs/005-review-apply-step/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0: Technical research
├── data-model.md        # Phase 1: Entity definitions
├── quickstart.md        # Phase 1: Developer quick start
├── contracts/
│   └── openapi-diff.yaml # Phase 1: OSInfo schema changes
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
# Service (Go) — modifications
pkg/protocol/types.go                    # Add SystemTime, ClockSynchronized to OSInfo
internal/inventory/clock.go              # NEW: timedatectl-based clock/sync detection
internal/inventory/clock_test.go         # NEW: Unit tests for clock detection
internal/api/handlers/info.go            # Call GetClockStatus() in gatherSystemInfo

# Service (Go) — contract update
specs/001-boardingpass-api/contracts/openapi.yaml  # Add fields to OSInfo schema

# Mobile App (TypeScript) — new files
mobile/src/utils/action-list.ts          # NEW: buildActionList() pure function
mobile/src/components/ConfigWizard/ReviewApplyPage.tsx  # NEW: Review & Apply screen

# Mobile App (TypeScript) — modifications
mobile/src/types/wizard.ts               # PlannedAction type, TOTAL_STEPS=6, actionList in state
mobile/src/types/api.ts                  # Regenerated from OpenAPI (make generate-app)
mobile/src/contexts/WizardContext.tsx     # Add actionList, applyInProgress to state/reducer
mobile/src/hooks/useConfigWizard.ts      # Add applyAllImmediate(), remove per-step apply from nav
mobile/src/components/ConfigWizard/WizardContainer.tsx  # Wire Step 6, simplify handleNext
mobile/src/components/ConfigWizard/StepIndicator.tsx    # Support 6 steps
mobile/src/components/DeviceInfo/SystemInformationCard.tsx  # Show time + sync status

# Tests
mobile/tests/unit/utils/action-list.test.ts    # NEW: Table-driven tests for action generation
mobile/tests/unit/components/ReviewApplyPage.test.tsx  # NEW: Review screen rendering tests
```

**Structure Decision**: Follows the existing multi-component layout. Service changes are minimal (one new inventory file + two fields added to existing types). Mobile changes center on a new utils module and component, with modifications to existing wizard infrastructure.

## Complexity Tracking

No constitution violations. No complexity justifications needed.
