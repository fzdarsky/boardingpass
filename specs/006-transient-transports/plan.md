# Implementation Plan: Transient Transport Provisioning

**Branch**: `006-transient-transports` | **Date**: 2026-03-08 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/006-transient-transports/spec.md`

## Summary

Add WiFi AP, Bluetooth PAN, and USB tethering as transient transport options for BoardingPass device provisioning. The service manages transport lifecycle via systemd template units (WiFi/Bluetooth) and interface monitoring (USB), listening on all active transports simultaneously. The mobile app discovers devices across all transports using WiFi SSID/gateway probing, BLE advertisement scanning, and USB tethering subnet probing. Key research finding: iOS cannot do Bluetooth PAN — BLE is used for discovery only, with actual API traffic over IP.

## Technical Context

**Language/Version**: Go 1.25+ (service), TypeScript 5.x with React Native 0.74+ (mobile app)
**Primary Dependencies**: Go stdlib + `gopkg.in/yaml.v3` (service); React Native, React Native Paper, Expo Router, Axios, `@react-native-community/netinfo`, `react-native-ble-plx` (app)
**Storage**: N/A — transport state is ephemeral (in-memory); config read from `/etc/boardingpass/config.yaml`
**Testing**: Go `testing` + `testify/assert` (service); Jest + React Native Testing Library (app)
**Target Platform**: Linux (RHEL 9+, systemd) for service; iOS 15+ / Android for mobile app
**Project Type**: Service (headless Linux daemon) + Mobile app (React Native/Expo)
**Performance Goals**: Transport setup < 10s; discovery < 5s; teardown < 10s
**Constraints**: No new Go runtime dependencies; CGO_ENABLED=0; single static binary < 10MB; FIPS 140-3 crypto compliance
**Scale/Scope**: Single device provisioning; 1-4 simultaneous transports; 1 mobile app connection at a time

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Frictionless Bootstrapping | PASS | Transient transports eliminate need for pre-existing network infrastructure; open WiFi default reduces credential burden |
| II. Ephemeral & Fail-Safe | PASS | systemd `BindsTo=`/`PartOf=` ensures transports are torn down on service exit or crash; no persistent state |
| III. Minimal Footprint | PASS | No new Go dependencies; systemd units are tiny; service binary size unchanged |
| IV. Minimal Dependencies | PASS (with note) | Service: no new Go dependencies, only `os/exec` for systemctl. System packages (hostapd, bluez) are optional — required only when respective transport is enabled. App: adds `react-native-ble-plx` (new native dependency for BLE) |
| V. Transport Agnostic & Protocol First | PASS | Core design: new transports extend config, reuse existing HTTPS API, no protocol changes. BLE used only for discovery, not as API transport |
| VI. Open Source & Permissive Licensing | PASS | `react-native-ble-plx` is MIT; `@react-native-community/netinfo` is MIT; all compatible |

**Post-Phase 1 Re-check**: All gates still pass. The BLE discovery approach (discovery-only, not transport) aligns with Transport Agnostic principle — BLE provides device location information, actual API calls use IP transport.

## Project Structure

### Documentation (this feature)

```text
specs/006-transient-transports/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: Research findings
├── data-model.md        # Phase 1: Entity definitions
├── quickstart.md        # Phase 1: Developer guide
├── contracts/
│   ├── transport-config.yaml   # Config schema extension
│   ├── systemd-units.md        # systemd unit contracts
│   └── discovery-methods.md    # App discovery contracts
└── checklists/
    └── requirements.md  # Spec quality checklist
```

### Source Code (repository root)

```text
# Service (Go)
internal/
├── config/
│   └── config.go              # Extended: WiFi, Bluetooth, USB transport config structs
├── transport/                 # NEW: Transport lifecycle management
│   ├── manager.go             # TransportManager: orchestrate all transports
│   ├── wifi.go                # WiFi transport: start/stop systemd unit
│   ├── bluetooth.go           # Bluetooth transport: start/stop systemd unit
│   └── usb.go                 # USB transport: interface monitoring
├── api/
│   ├── server.go              # Modified: multi-listener support
│   └── captive.go             # NEW: iOS captive portal response handler
└── lifecycle/
    └── shutdown.go            # Modified: transport teardown integration

cmd/boardingpass/
└── main.go                    # Modified: transport setup in startup sequence

build/
├── boardingpass-wifi@.service  # NEW: WiFi AP systemd template unit
├── boardingpass-bt@.service    # NEW: Bluetooth PAN systemd template unit
├── boardingpass-ble@.service   # NEW: BLE advertisement systemd template unit
└── boardingpass.sudoers        # Modified: add transport unit permissions

# Mobile App (TypeScript/React Native)
mobile/src/
├── types/
│   └── device.ts              # Modified: extended DiscoveryMethod
├── services/discovery/
│   ├── wifi.ts                # NEW: WiFi SSID + gateway discovery
│   ├── bluetooth.ts           # NEW: BLE advertisement discovery
│   ├── usb.ts                 # NEW: USB tethering discovery
│   ├── manager.ts             # NEW: Discovery orchestrator
│   └── preference.ts          # NEW: Transport preference + de-duplication
├── contexts/
│   └── DeviceContext.tsx       # Modified: extended DiscoveryMethod, de-duplication
```

**Structure Decision**: Extends existing project structure. Service adds `internal/transport/` package for transport lifecycle management. Mobile app adds new discovery services under existing `services/discovery/` directory, following the established pattern from `mdns.ts` and `fallback.ts`.

## Complexity Tracking

> No constitution violations to justify. All changes align with existing principles.

| Consideration | Resolution |
| --- | --- |
| `react-native-ble-plx` (new native dep) | Required for BLE discovery on iOS/Android; MIT license; well-maintained; no Go-side equivalent needed |
| hostapd/bluez as system deps | Optional — only needed when WiFi/Bluetooth transports are enabled; not bundled with BoardingPass |
| Multi-listener server | Minimal change — Go `net/http` supports multiple listeners natively via goroutines |
