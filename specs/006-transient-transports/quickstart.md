# Quickstart: Transient Transport Provisioning

**Feature Branch**: `006-transient-transports`

## Overview

This feature adds WiFi AP, Bluetooth PAN, and USB tethering as transient transport options for BoardingPass device provisioning. The service creates/tears down transports via systemd units. The mobile app discovers devices across all transports.

## Architecture at a Glance

```
                      Mobile App
                    ┌─────────────┐
                    │  Discovery   │
                    │  Manager     │
                    │  ┌─────────┐ │
                    │  │WiFi Disc│ │──── WiFi AP ────┐
                    │  │BLE Disc │ │──── BLE Adv ────┤
                    │  │USB Disc │ │──── USB Tether ──┤
                    │  │mDNS Disc│ │──── Ethernet ────┤
                    │  └─────────┘ │                  │
                    └─────────────┘                  ▼
                                              ┌──────────────┐
                                              │  BoardingPass │
                                              │  Service      │
                                              │  ┌──────────┐ │
                                              │  │Transport │ │
                                              │  │Manager   │ │
                                              │  │ WiFi AP  │ │◄── systemd: boardingpass-wifi@wlan0
                                              │  │ BT PAN   │ │◄── systemd: boardingpass-bt@hci0
                                              │  │ USB mon  │ │◄── poll /sys/class/net/
                                              │  │ Ethernet │ │◄── existing
                                              │  └──────────┘ │
                                              └──────────────┘
```

## Key Components

### Service-Side (Go)

| Component | Location | Purpose |
|-----------|----------|---------|
| Transport config types | `internal/config/config.go` | WiFi, Bluetooth, USB config structs |
| TransportManager | `internal/transport/manager.go` | Lifecycle orchestration (start/stop systemd units) |
| USB monitor | `internal/transport/usb.go` | Poll for USB tethering interfaces |
| Multi-listener server | `internal/api/server.go` | Bind HTTPS to multiple interfaces |
| Captive portal handler | `internal/api/captive.go` | Respond to iOS captive portal probes |
| systemd unit templates | `build/boardingpass-wifi@.service` | WiFi AP unit (runs hostapd) |
| systemd unit templates | `build/boardingpass-bt@.service` | Bluetooth PAN unit |
| sudoers extension | `build/boardingpass.sudoers` | Allow start/stop of transport units |

### App-Side (TypeScript/React Native)

| Component | Location | Purpose |
|-----------|----------|---------|
| Extended DiscoveryMethod | `mobile/src/types/device.ts` | Add `wifi`, `bluetooth`, `usb` |
| WiFi discovery service | `mobile/src/services/discovery/wifi.ts` | SSID detection + gateway probing |
| BLE discovery service | `mobile/src/services/discovery/bluetooth.ts` | BLE scan + GATT read |
| USB discovery service | `mobile/src/services/discovery/usb.ts` | Tethering subnet probing |
| Discovery orchestrator | `mobile/src/services/discovery/manager.ts` | Coordinate all discovery methods |
| Transport preference | `mobile/src/services/discovery/preference.ts` | De-duplication + auto-selection |
| DeviceContext update | `mobile/src/contexts/DeviceContext.tsx` | Extended DiscoveryMethod type |

## Dependencies

### Service-Side
- **No new Go dependencies** — systemd interaction via `os/exec` calling `systemctl`
- **System packages** (installed by system builder, not by BoardingPass):
  - `hostapd` — WiFi AP daemon
  - `bluez` — Bluetooth stack (provides `bluetoothctl` and BlueZ D-Bus API)
  - systemd unit template files (shipped with BoardingPass package)

### App-Side
- `@react-native-community/netinfo` — Network type and SSID detection (may already be present)
- `react-native-ble-plx` — BLE scanning and GATT client (**new native dependency**)

## Development Workflow

### Service changes

1. Extend `config.go` with WiFi/Bluetooth/USB transport config structs
2. Create `internal/transport/` package with TransportManager
3. Modify `server.go` to support multiple listeners
4. Add captive portal handler for iOS WiFi compatibility
5. Create systemd unit templates in `build/`
6. Extend sudoers file
7. Update `cmd/boardingpass/main.go` startup/shutdown sequence

### App changes

1. Extend `DiscoveryMethod` type in `device.ts`
2. Create WiFi, BLE, USB discovery services following existing `mdns.ts` pattern
3. Create discovery manager to coordinate all methods
4. Add transport preference logic
5. Update DeviceContext with de-duplication
6. Install and configure `react-native-ble-plx` (requires `npx expo prebuild`)

### Testing approach

- **Unit tests**: Transport config validation, discovery method selection, de-duplication logic
- **Integration tests**: TransportManager with mock systemd, discovery services with mock responses
- **Contract tests**: Config schema validation, BLE GATT characteristic format
- **E2E tests**: Full flow on physical hardware (WiFi AP creation, BLE discovery, USB tethering)

## Critical Decisions

1. **BLE for discovery, not transport**: iOS cannot do Bluetooth PAN. BLE is used only to discover device IP/port; actual API traffic flows over IP.
2. **systemd units, not in-process**: Transport lifecycle managed by systemd for crash safety and process supervision.
3. **Open WiFi by default**: SRP provides security; WiFi password is opt-in for compliance.
4. **No new Go dependencies**: systemd interaction uses `os/exec` + `systemctl`, consistent with minimal dependency principle.
