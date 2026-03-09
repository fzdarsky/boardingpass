# Tasks: Transient Transport Provisioning

**Input**: Design documents from `/specs/006-transient-transports/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in spec. Test tasks are omitted. Run `make test-all` after each phase checkpoint.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Extend existing project structure with transport package skeleton and app dependencies

- [x] T001 [P] Add WiFi, Bluetooth, and USB transport config structs to `internal/config/config.go` per data-model.md (WiFiTransport, BluetoothTransport, USBTransport fields in TransportSettings; include validation for password length, channel range, interface prefix)
- [x] T002 [P] Create `internal/transport/` package directory and add `transport.go` with TransportType enum, TransportState enum, and Transport struct per data-model.md state machine
- [x] T003 [P] Extend `DiscoveryMethod` type in `mobile/src/types/device.ts` to include `'wifi' | 'bluetooth' | 'usb'` values; update `DeviceContext.tsx` local Device type to match
- [x] T004 [P] Install `@react-native-community/netinfo` and `react-native-ble-plx` in `mobile/package.json`; update `mobile/app.json` with iOS entitlements (`com.apple.developer.networking.wifi-info`) and Bluetooth permission descriptions (`NSBluetoothAlwaysUsageDescription`); run `npx expo prebuild --platform ios`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

- [x] T005 Implement TransportManager in `internal/transport/manager.go` with: constructor accepting `*config.Config` and `*logging.Logger`; `StartAll(ctx context.Context) error` that iterates enabled transports and starts each (calling transport-specific Start methods); `StopAll(ctx context.Context) error` for shutdown; `ActiveTransports() []Transport` returning transports in active state; non-fatal error handling per FR-011 (log warning, continue with remaining transports)
- [x] T006 Modify `internal/api/server.go` to support multiple listeners: extract listener creation into a method that accepts a list of `(address, port)` bindings; start one goroutine per listener calling `server.Serve(listener)`; collect all listeners for graceful shutdown; keep backward compatibility with single Ethernet binding (FR-012)
- [x] T007 Create discovery manager in `mobile/src/services/discovery/manager.ts` following existing singleton pattern from `mdns.ts`: `DiscoveryManager` class that coordinates mDNS, fallback, WiFi, BLE, and USB discovery services; `startAll()` / `stopAll()` methods; `onDeviceFound(callback)` that aggregates devices from all sources; pass-through to existing mDNS and fallback services initially
- [x] T008 Create transport preference and de-duplication logic in `mobile/src/services/discovery/preference.ts`: `getTransportPriority(method: DiscoveryMethod): number` returning USB=1, bluetooth=2, wifi=3, mdns/fallback=4, manual=5; `deduplicateDevices(devices: Device[]): Device[]` merging entries with same certificate fingerprint or hostname; `selectPreferredTransport(transports: Device[]): Device` picking highest priority

**Checkpoint**: Foundation ready - transport manager skeleton and multi-listener server support in place

---

## Phase 3: User Story 1 - WiFi Access Point Transport (Priority: P1) MVP

**Goal**: System builder configures WiFi transport; service creates open WiFi AP via systemd unit; mobile app discovers device on WiFi hotspot

**Independent Test**: Configure `transports.wifi` in config.yaml with a WiFi interface, start service, verify AP appears, connect phone, discover service in app, shut down service, verify AP removed

### Implementation for User Story 1

- [x] T009 [P] [US1] Create WiFi transport implementation in `internal/transport/wifi.go`: `WiFiTransport` struct implementing transport interface; `Start(ctx)` calls `sudo systemctl start boardingpass-wifi@<interface>` via `os/exec`; `Stop(ctx)` calls `sudo systemctl stop boardingpass-wifi@<interface>`; validate interface exists in `/sys/class/net/` before starting; generate SSID default `BoardingPass-<hostname>` if not configured (FR-001 through FR-004)
- [x] T010 [P] [US1] Create systemd WiFi AP template unit in `build/boardingpass-wifi@.service` per contracts/systemd-units.md: `PartOf=boardingpass.service`, `BindsTo=boardingpass.service`, `After=boardingpass.service`; `ExecStartPre` brings interface up; `ExecStart` runs hostapd with `/etc/boardingpass/hostapd-%i.conf`; `ExecStopPost` flushes IP and brings interface down
- [x] T011 [P] [US1] Create captive portal handler in `internal/api/captive.go`: register route for `/hotspot-detect.html` responding with `<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>` (suppresses iOS captive portal popup per research.md R4); register route for Android `/generate_204` returning HTTP 204
- [x] T012 [P] [US1] Add WiFi transport systemd unit permissions to `build/boardingpass.sudoers`: `boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl start boardingpass-wifi@*` and `boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop boardingpass-wifi@*`
- [x] T013 [US1] Register WiFi transport in TransportManager (`internal/transport/manager.go`): when `config.Transports.WiFi.Enabled` is true, create WiFiTransport instance and add to managed transports; wire Start/Stop into StartAll/StopAll lifecycle
- [x] T014 [P] [US1] Create WiFi discovery service in `mobile/src/services/discovery/wifi.ts` following `fallback.ts` singleton pattern: use `@react-native-community/netinfo` to detect WiFi connection; check SSID pattern `BoardingPass-*`; if SSID unavailable, probe gateway IP with HTTPS HEAD on port 8443; return Device with `discoveryMethod: 'wifi'` and `host` set to gateway IP (FR-014)
- [x] T015 [US1] Integrate WiFi discovery into discovery manager (`mobile/src/services/discovery/manager.ts`): call WiFi discovery on `startAll()`; subscribe to netinfo WiFi state changes to trigger re-scan; pass discovered devices through to `onDeviceFound` callback
- [x] T016 [US1] Wire transport config validation into `internal/config/config.go` validation: if WiFi enabled, `interface` must be non-empty; if `password` set, must be >= 8 chars; `channel` must be 1-165; `address` must be valid IP; add WiFi password to secret redaction list in `internal/logging/` (FR-021)

**Checkpoint**: WiFi AP transport fully functional end-to-end. Service creates AP on start, app discovers device on WiFi, AP torn down on shutdown.

---

## Phase 4: User Story 2 - Bluetooth PAN Transport (Priority: P2)

**Goal**: System builder configures Bluetooth transport; service creates Bluetooth PAN and BLE advertisement via systemd units; mobile app discovers device via BLE scanning

**Independent Test**: Configure `transports.bluetooth` in config.yaml with a Bluetooth adapter, start service, verify device is discoverable and PAN is created, scan with phone BLE, discover service in app, shut down service, verify Bluetooth cleaned up

### Implementation for User Story 2

- [x] T017 [P] [US2] Create Bluetooth transport implementation in `internal/transport/bluetooth.go`: `BluetoothTransport` struct; `Start(ctx)` calls `sudo systemctl start boardingpass-bt@<adapter>` and `sudo systemctl start boardingpass-ble@<adapter>`; `Stop(ctx)` stops both units; validate adapter exists in `/sys/class/bluetooth/` before starting; generate device_name default `BoardingPass-<hostname>` if not configured (FR-005 through FR-007)
- [x] T018 [P] [US2] Create systemd Bluetooth PAN template unit in `build/boardingpass-bt@.service` per contracts/systemd-units.md: `PartOf=boardingpass.service`, `BindsTo=boardingpass.service`; `ExecStartPre` powers on adapter, sets discoverable, sets device name; `ExecStart` creates NAP bridge and assigns IP; `ExecStopPost` removes bridge, disables discoverability
- [x] T019 [P] [US2] Create systemd BLE advertisement template unit in `build/boardingpass-ble@.service` per contracts/systemd-units.md: advertises BoardingPass BLE service UUID; GATT characteristics for device name, IP address, port, certificate fingerprint per contracts/discovery-methods.md
- [x] T020 [P] [US2] Add Bluetooth/BLE transport systemd unit permissions to `build/boardingpass.sudoers`: start/stop permissions for `boardingpass-bt@*` and `boardingpass-ble@*`
- [x] T021 [US2] Register Bluetooth transport in TransportManager (`internal/transport/manager.go`): when `config.Transports.Bluetooth.Enabled` is true, create BluetoothTransport instance and add to managed transports
- [x] T022 [P] [US2] Create BLE discovery service in `mobile/src/services/discovery/bluetooth.ts` following singleton pattern: use `react-native-ble-plx` to scan for BoardingPass BLE service UUID; on device found, connect to GATT server and read device info characteristics (name, IP, port, cert fingerprint); return Device with `discoveryMethod: 'bluetooth'` and host/port from GATT data (FR-015)
- [x] T023 [US2] Integrate BLE discovery into discovery manager (`mobile/src/services/discovery/manager.ts`): call BLE discovery `start()` on `startAll()`; handle BLE permission requests; pass discovered devices through to `onDeviceFound` callback
- [x] T024 [US2] Wire Bluetooth config validation into `internal/config/config.go`: if Bluetooth enabled, validate adapter name; add Bluetooth pairing details to secret redaction list (FR-021)

**Checkpoint**: Bluetooth PAN + BLE discovery fully functional. Service creates PAN and BLE advertisement, app discovers device via BLE, both cleaned up on shutdown.

---

## Phase 5: User Story 3 - USB Tethering Discovery (Priority: P3)

**Goal**: Service detects USB tethering interfaces and listens on them; mobile app discovers device on tethered network

**Independent Test**: Enable USB transport in config, connect phone via USB with tethering enabled, verify service detects interface and listens on it, verify app discovers device, disconnect cable, verify graceful cleanup

### Implementation for User Story 3

- [x] T025 [P] [US3] Create USB transport implementation in `internal/transport/usb.go`: `USBTransport` struct; `Start(ctx)` begins polling `/sys/class/net/` every 2 seconds for interfaces matching configured prefix (default `usb`, also `rndis`); when interface detected with IP assigned, notify TransportManager to bind listener; `Stop(ctx)` cancels polling and closes USB listeners; handle interface disappearance gracefully (FR-008, FR-009)
- [x] T026 [US3] Register USB transport in TransportManager (`internal/transport/manager.go`): when `config.Transports.USB.Enabled` is true, create USBTransport instance; implement dynamic listener addition/removal when USB interfaces appear/disappear (callback from USBTransport to server)
- [x] T027 [US3] Wire dynamic listener support into `internal/api/server.go`: add `AddListener(address string, port int) error` and `RemoveListener(address string) error` methods for USB transport's runtime interface changes; handle graceful connection draining on listener removal
- [x] T028 [P] [US3] Create USB discovery service in `mobile/src/services/discovery/usb.ts` following singleton pattern: use `@react-native-community/netinfo` to detect non-WiFi/non-cellular connections; probe well-known tethering gateway IPs (`172.20.10.1` for iOS, `192.168.42.1` for Android) with HTTPS HEAD on port 8443; return Device with `discoveryMethod: 'usb'` (FR-016)
- [x] T029 [US3] Integrate USB discovery into discovery manager (`mobile/src/services/discovery/manager.ts`): call USB discovery on `startAll()`; subscribe to netinfo state changes for non-WiFi connections; pass discovered devices through to `onDeviceFound` callback

**Checkpoint**: USB tethering transport fully functional. Service detects tethering interface, app discovers device on tethered network, graceful cleanup on disconnect.

---

## Phase 6: User Story 4 - Multi-Transport Simultaneous Operation (Priority: P4)

**Goal**: Multiple transports operate simultaneously; app de-duplicates devices and auto-selects preferred transport

**Independent Test**: Enable Ethernet + WiFi + Bluetooth in config, start service, verify all transports active, verify app shows single device entry with preferred transport, switch transports manually, verify shutdown tears down all transient transports

### Implementation for User Story 4

- [x] T030 [US4] Integrate TransportManager into service startup sequence in `cmd/boardingpass/main.go`: call `transportManager.StartAll(ctx)` after config load and before server start; call `transportManager.StopAll(ctx)` in shutdown sequence after server stop; pass active transport addresses to server for listener binding (FR-012, FR-013)
- [x] T031 [US4] Integrate TransportManager into shutdown flow in `internal/lifecycle/shutdown.go`: extend `GracefulShutdown` to include transport teardown step; ensure transports are stopped within 10-second shutdown timeout (FR-010)
- [x] T032 [US4] Integrate de-duplication into `mobile/src/contexts/DeviceContext.tsx`: modify `ADD_DEVICE` reducer action to call `deduplicateDevices()` from `preference.ts`; store alternate transports on each Device; display preferred transport's discovery method; expose `switchTransport(deviceId, method)` action for manual override (FR-018, FR-019)
- [x] T033 [US4] Update device list UI to show transport indicator and allow manual transport switching: add transport badge/icon to device list item in discovery screen (`mobile/app/index.tsx` or device list component); when device has multiple transports, show dropdown or tap-to-switch UI (FR-019)

**Checkpoint**: All transports operate simultaneously, app shows de-duplicated device list with transport preference, shutdown cleans up all transient transports.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Security hardening, logging, and build integration

- [x] T034 [P] Add WiFi password and Bluetooth pairing details to secret redaction patterns in `internal/logging/` (FR-021); verify no transport secrets appear in log output
- [x] T035 [P] Update `build/config.yaml` with example WiFi, Bluetooth, and USB transport sections showing all configuration options with comments
- [x] T036 Run `make lint-all` and fix all linting errors across service and app
- [x] T037 Run `make test-all` and fix all test failures (existing tests must still pass with new transport code)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately. T001-T004 all parallelizable.
- **Foundational (Phase 2)**: Depends on Setup. T005 depends on T001+T002. T006 independent. T007 depends on T003. T008 depends on T003.
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2). WiFi transport is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational (Phase 2). Can run in parallel with US1 (different files).
- **User Story 3 (Phase 5)**: Depends on Foundational (Phase 2). Can run in parallel with US1/US2 (different files). T027 touches server.go (coordinate with T006).
- **User Story 4 (Phase 6)**: Depends on at least US1 being complete. Integrates all transports into startup/shutdown.
- **Polish (Phase 7)**: Depends on all user stories being complete.

### User Story Dependencies

- **US1 (WiFi)**: Independent after Phase 2. MVP target.
- **US2 (Bluetooth)**: Independent after Phase 2. Can parallelize with US1.
- **US3 (USB)**: Independent after Phase 2. T027 coordinates with T006 (server.go changes).
- **US4 (Multi-transport)**: Depends on US1 minimum; benefits from US2+US3 for full validation.

### Parallel Opportunities

**Phase 1** (all parallel):
- T001 (Go config) || T002 (Go transport types) || T003 (App types) || T004 (App deps)

**Phase 2** (partial parallel):
- T006 (multi-listener) || T008 (preference logic)
- T005 (TransportManager) depends on T001+T002
- T007 (discovery manager) depends on T003

**Phase 3 (US1)** (service || app):
- T009 (wifi.go) || T010 (systemd unit) || T011 (captive.go) || T012 (sudoers) || T014 (wifi.ts)

**Phase 4 (US2)** (service || app):
- T017 (bluetooth.go) || T018 (bt unit) || T019 (ble unit) || T020 (sudoers) || T022 (bluetooth.ts)

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T004)
2. Complete Phase 2: Foundational (T005-T008)
3. Complete Phase 3: User Story 1 - WiFi (T009-T016)
4. **STOP and VALIDATE**: Test WiFi AP transport end-to-end
5. Deploy/demo if ready - device provisioning works over WiFi hotspot

### Incremental Delivery

1. Setup + Foundational -> Foundation ready
2. Add US1 (WiFi) -> Test independently -> Deploy (MVP!)
3. Add US2 (Bluetooth) -> Test independently -> Deploy
4. Add US3 (USB) -> Test independently -> Deploy
5. Add US4 (Multi-transport) -> Test integration -> Deploy
6. Polish -> Final release

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Service-side tasks (Go) and app-side tasks (TypeScript) within the same story can always run in parallel
- WiFi transport (US1) is the MVP - delivers value without Bluetooth or USB
- BLE is used for discovery only (not as API transport) due to iOS PAN limitation
- No new Go runtime dependencies - systemd interaction via `os/exec`
- After each phase checkpoint, run `make lint-all && make test-all`
