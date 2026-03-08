# Data Model: Review & Apply Step

**Feature**: 005-review-apply-step
**Date**: 2026-03-05

## Entities

### PlannedAction

A single item in the review action list. Represents one discrete operation that will be performed during the apply phase.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier for the action (e.g., `hostname`, `wifi-connect`, `ipv4-config`, `connectivity-check`, `enroll-insights`) |
| description | string | Human-readable description (e.g., "Set hostname to `my-device`") |
| category | enum | One of: `config`, `command`, `check`, `wait` |
| status | enum | One of: `pending`, `running`, `success`, `failed`, `skipped` |
| detail | string or null | Additional detail shown after execution (e.g., error message, connectivity test results) |
| configFiles | RawConfigFile[] | Config files to send via /configure for this action (may be empty) |
| commands | StepCommand[] | Commands to execute via /command for this action (may be empty) |
| infoOnly | boolean | If true, the action is informational and does not execute anything (used in deferred mode for checks/waits) |

**Notes**:
- `category` determines the icon shown: `config` = gear, `command` = terminal, `check` = magnifying glass, `wait` = clock
- `skipped` status is used for actions that were not executed because a prior action failed
- `configFiles` and `commands` reuse existing types from `useConfigWizard.ts` — no new API data structures

### ClockStatus (added to OSInfo)

Clock synchronization state returned as part of the /info endpoint response, nested within the existing `OSInfo` object.

| Field | Type | Description |
|-------|------|-------------|
| system_time | string (ISO 8601) | Current system time in UTC (e.g., `2026-03-05T14:30:00Z`) |
| clock_synchronized | boolean | Whether the system clock is synchronized via NTP |

**Notes**:
- These fields are added directly to the existing `OSInfo` schema — no new top-level entity
- `system_time` is dynamic (not cached) unlike other /info fields — the info handler's 1-second cache is acceptable since time precision to the second is sufficient
- On systems where timedatectl is unavailable (unlikely on RHEL 9+), defaults to current time and `false` for sync status

## State Changes

### WizardState (extended)

The existing `WizardState` type is extended to support the review step:

| Change | Field | Type | Description |
|--------|-------|------|-------------|
| Modified | `TOTAL_STEPS` | constant | Changed from `5` to `6` |
| Added | `WIZARD_STEPS.REVIEW` | constant | Value `6` (already defined but unused) |
| Added | `STEP_LABELS[6]` | string | `"Review"` |
| Added | `actionList` | `PlannedAction[]` | Populated when entering Step 6 |
| Added | `applyInProgress` | boolean | True while the apply sequence is executing |

**Notes**:
- `WIZARD_STEPS.REVIEW = 6` already exists in the current codebase but is not wired into navigation
- The `actionList` is regenerated each time the user enters Step 6 (not cached across back-navigation)
- `stepApplyStatus` is retained for per-step tracking but is no longer updated during step navigation in immediate mode (only during the apply phase)

## Relationships

```
WizardState
  ├── hostname: HostnameConfig ──────┐
  ├── networkInterface: InterfaceConfig ──┤
  ├── addressing: AddressingConfig ──┤   generates
  ├── services: ServicesConfig ──────┤──────────→ PlannedAction[]
  ├── enrollment: EnrollmentConfig ──┘              │
  └── actionList: PlannedAction[]  ←────────────────┘

SystemInfo
  └── os: OSInfo
        ├── distribution: string
        ├── version: string
        ├── fips_enabled: boolean
        ├── system_time: string (NEW)
        └── clock_synchronized: boolean (NEW)
```

## Action Generation Rules

The `buildActionList` function generates actions based on wizard state. The order is fixed:

1. **Hostname** — always present (either "Keep..." or "Set...")
2. **Interface selection** — always present ("Use [type] interface [name]")
3. **WiFi connection** — only if WiFi interface selected
4. **IPv4 configuration** — always present (DHCP or static details)
5. **DNS configuration** — only if manual DNS configured
6. **IPv6 configuration** — only if not disabled
7. **Connectivity check** — only in immediate mode, if gateway configured
8. **DNS resolution check** — only in immediate mode, if DNS configured
9. **NTP configuration** — always present (automatic or manual)
10. **Clock sync wait** — always present (immediate = poll, deferred = informational)
11. **Enrollment: Insights** — only if Insights configured
12. **Enrollment: Flight Control** — only if Flight Control configured
