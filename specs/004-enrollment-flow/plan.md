# Implementation Plan: Enrollment Configuration Wizard

**Branch**: `004-enrollment-flow` | **Date**: 2026-03-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-enrollment-flow/spec.md`

## Summary

Add a 5-step configuration wizard to the BoardingPass mobile app (Hostname → Network Interface → Addressing → Services → Enrollment Server) with two apply modes: immediate apply when the enrollment interface differs from the service interface, and deferred apply with atomic bundle + reboot when they're the same. The implementation extends the existing BoardingPass API minimally — adding `hostname` to `/info`, hardware metadata to `/network` interfaces, command parameters to `/command`, and a reboot option to `/complete` — while leveraging the transactional `/configure` endpoint for all file provisioning. Device-side operations use allow-listed helper scripts for WiFi scanning, network reload, connectivity testing, and enrollment.

## Technical Context

**Language/Version**: Go 1.25+ (service), TypeScript 5.x with React Native 0.74+ (mobile app)
**Primary Dependencies**: Go stdlib, `gopkg.in/yaml.v3` (service); React Native, React Native Paper, Expo Router, Axios (app)
**Storage**: N/A — wizard state is ephemeral (in-memory), config files written to `/etc/` via existing provisioning
**Testing**: Go `testing` + `testify/assert` + `uber-go/mock` (service); Jest + React Native Testing Library (app)
**Target Platform**: Linux amd64/arm64 (service); iOS 15+ / Android API 29+ (app)
**Project Type**: Mobile app + backend service
**Constraints**: FIPS 140-3 compliance (no third-party crypto), single static binary <10MB (service), bundle <50MB (app), Go stdlib crypto only

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Frictionless Bootstrapping | ✅ Pass | Wizard guides non-technical users through enrollment step-by-step |
| II. Ephemeral & Fail-Safe | ✅ Pass | Wizard state is ephemeral; deferred mode uses atomic `/configure`; sentinel prevents re-run |
| III. Minimal Footprint | ✅ Pass | No new dependencies on service side; interface type detection uses sysfs (no hwdata); scripts use system utilities |
| IV. Minimal Dependencies | ✅ Pass | Service changes use Go stdlib only; scripts use nmcli, hostnamectl, systemctl (all present on target RHEL 9+ systems) |
| V. Transport Agnostic | ✅ Pass | All changes are to the application layer (API handlers, protocol types); transport layer untouched |
| VI. Open Source & Permissive | ✅ Pass | No new proprietary dependencies |
| Security: Secrets Management | ✅ Pass | Enrollment credentials via staging files (not process table); deleted after use; never logged |
| Security: Input Validation | ✅ Pass | Command params validated (max_params, length); config paths validated by existing allow-list |
| Security: Least Privilege | ✅ Pass | Scripts run via sudo with specific sudoers entries; `--` separator prevents option injection |

**Post-Phase 1 re-check**: All gates still pass. The staging file + command param hybrid maintains security boundaries. The systemd oneshot for deferred enrollment is a standard Linux pattern.

## Project Structure

### Documentation (this feature)

```text
specs/004-enrollment-flow/
├── spec.md
├── plan.md              # This file
├── research.md          # Technical decisions (RD-01 through RD-12)
├── data-model.md        # Entity definitions
├── quickstart.md        # Developer quick start
├── contracts/
│   └── api-extensions.yaml  # OpenAPI changes + command config
└── checklists/
    └── requirements.md  # Spec quality checklist
```

### Source Code (repository root)

```text
# Service (Go) — modifications to existing code
internal/
├── network/
│   └── interfaces.go         # MODIFY: add type, speed, carrier, driver, vendor, model from sysfs + hwdata
├── inventory/
│   └── hostname.go           # NEW: hostname retrieval via os.Hostname()
├── command/
│   ├── executor.go           # MODIFY: append -- params to command
│   └── allowlist.go          # MODIFY: validate max_params
├── config/
│   └── config.go             # MODIFY: add max_params to CommandDefinition
├── api/handlers/
│   ├── command.go            # MODIFY: parse params from request
│   ├── complete.go           # MODIFY: handle reboot flag
│   └── info.go               # MODIFY: include hostname in response
└── lifecycle/
    └── reboot.go             # NEW: scheduled reboot logic

pkg/protocol/
└── types.go                  # MODIFY: add params, hostname, interface fields, CompleteRequest

# Service — new scripts
build/
├── scripts/
│   ├── wifi-scan.sh              # NEW: nmcli wrapper → JSON
│   ├── reload-connection.sh      # NEW: nmcli connection reload + up
│   ├── connectivity-test.sh      # NEW: DNS + gateway + internet test
│   ├── enroll-insights.sh        # NEW: subscription-manager register
│   └── enroll-flightctl.sh       # NEW: flightctl enroll
├── boardingpass.sudoers          # MODIFY: add new script entries
└── config.yaml                   # MODIFY: add commands + path allow-list entries

# OpenAPI spec
specs/001-boardingpass-api/contracts/
└── openapi.yaml                  # MODIFY: fix /configure desc, extend schemas

# Mobile App (React Native)
mobile/
├── app/device/
│   └── configure.tsx             # NEW: wizard screen
├── src/
│   ├── components/ConfigWizard/
│   │   ├── WizardContainer.tsx   # NEW: step management + navigation
│   │   ├── StepIndicator.tsx     # NEW: progress indicator
│   │   ├── HostnameStep.tsx      # NEW: step 1
│   │   ├── InterfaceStep.tsx     # NEW: step 2 + VLAN
│   │   ├── WiFiStep.tsx          # NEW: step 2a (conditional)
│   │   ├── AddressingStep.tsx    # NEW: step 3
│   │   ├── ServicesStep.tsx      # NEW: step 4
│   │   ├── EnrollmentStep.tsx    # NEW: step 5
│   │   ├── ReviewPage.tsx        # NEW: deferred mode summary
│   │   └── ApplyFeedback.tsx     # NEW: per-step apply status
│   ├── contexts/
│   │   └── WizardContext.tsx     # NEW: wizard state via useReducer
│   ├── hooks/
│   │   └── useConfigWizard.ts    # NEW: wizard logic, validation, apply
│   ├── services/api/
│   │   ├── configure.ts          # NEW: POST /configure wrapper
│   │   ├── command.ts            # NEW: POST /command wrapper
│   │   └── complete.ts           # NEW: POST /complete wrapper
│   ├── utils/
│   │   ├── network-validation.ts # NEW: IP, subnet, hostname validators
│   │   └── nm-connection.ts      # NEW: NM connection file builder
│   └── types/
│       └── wizard.ts             # NEW: wizard TypeScript types
└── tests/
    ├── unit/
    │   ├── hooks/useConfigWizard.test.ts
    │   ├── utils/network-validation.test.ts
    │   └── utils/nm-connection.test.ts
    └── integration/
        └── config-wizard.test.ts
```

**Structure Decision**: Feature touches both service and mobile app. Service changes are minimal (extend existing handlers + add scripts). Mobile app changes are substantial (new wizard screen with 10+ components). No new packages or architecture patterns introduced — follows existing patterns in both codebases.

## Complexity Tracking

No constitution violations to justify.
