# Quick Start: Review & Apply Step

**Feature**: 005-review-apply-step
**Date**: 2026-03-05

## What This Feature Does

Adds a "Review & Apply" screen (Step 6) to the configuration wizard. Instead of applying configuration changes at each step, the user reviews all planned actions on a single screen and applies them in one go. Also adds system time and clock sync status to the device info endpoint and Device Details screen.

## Changes Overview

### Service (Go)
- **`internal/inventory/clock.go`** (new): Reads system time and NTP sync status via `timedatectl show`
- **`pkg/protocol/types.go`**: Add `SystemTime` and `ClockSynchronized` fields to `OSInfo`
- **`internal/api/handlers/info.go`**: Call new clock inventory function in `gatherSystemInfo`
- **OpenAPI spec**: Add `system_time` and `clock_synchronized` to `OSInfo` schema

### Mobile App (TypeScript/React Native)
- **`mobile/src/utils/action-list.ts`** (new): Pure function `buildActionList()` that generates human-readable action list from wizard state
- **`mobile/src/components/ConfigWizard/ReviewApplyPage.tsx`** (new): Review screen with action list display and apply execution
- **`mobile/src/components/ConfigWizard/WizardContainer.tsx`**: Wire up Step 6, remove per-step apply logic
- **`mobile/src/hooks/useConfigWizard.ts`**: Add `applyAllImmediate()` for sequential execution, remove per-step apply from navigation
- **`mobile/src/types/wizard.ts`**: Update `TOTAL_STEPS` to 6, add `PlannedAction` type, add `actionList` to state
- **`mobile/src/components/DeviceInfo/SystemInformationCard.tsx`**: Add system time and clock sync to OS section

## Development Workflow

```bash
# 1. Service changes
# Edit Go files, then:
make lint-service && make test-unit-service && make build-service

# 2. Mobile app changes
# Edit TypeScript files, then:
make generate-app          # Regenerate types from updated OpenAPI spec
make lint-app && make test-unit-app

# 3. Run the app
make run-app-ios           # Test on iOS simulator

# 4. Full validation
make lint-all && make test-all
```

## Key Design Decisions

1. **Step 6, not a modal**: Review is a real wizard step with step indicator, back navigation, and full-screen layout — not a popup.
2. **Pure action list generation**: `buildActionList()` is a pure function (state in → actions out) for easy testing.
3. **Sequential execution**: In immediate mode, actions execute one by one with real-time UI feedback. Failure stops the sequence.
4. **Hostname exception**: Still applied immediately at Step 1 (safe, no connectivity impact).
5. **Clock via timedatectl**: Uses systemd's `timedatectl show` — always available on RHEL 9+, works with any NTP provider.
