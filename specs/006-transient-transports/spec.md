# Feature Specification: Transient Transport Provisioning

**Feature Branch**: `006-transient-transports`
**Created**: 2026-03-08
**Status**: Draft
**Input**: User description: "Add provisioning via transient Bluetooth, WiFi, or USB cable transports into BoardingPass. Service creates and tears down Bluetooth/WiFi access points based on config. Mobile app discovers devices via Bluetooth, WiFi, and USB tethering."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - WiFi Access Point Transport (Priority: P1)

A system builder configures a headless Linux device to create a transient WiFi access point for provisioning. When the BoardingPass service starts and a WiFi interface is available, it creates an open (passwordless) WiFi hotspot. A mobile app user connects their phone to this hotspot and discovers the BoardingPass service. Since SRP authentication provides the security layer, the WiFi AP is open by default to minimize friction — the system builder may optionally configure WPA2-PSK if transport-level encryption is also desired. When provisioning completes and the service disables itself, the WiFi access point is automatically removed.

**Why this priority**: WiFi provides the broadest compatibility across phones and devices. Most headless Linux devices with wireless capability can host an access point, and all modern phones support WiFi. This transport enables provisioning in environments with no existing network infrastructure.

**Independent Test**: Can be fully tested by configuring WiFi transport in config.yaml, verifying the access point appears, connecting a phone, discovering the service, and confirming the access point is removed on shutdown.

**Acceptance Scenarios**:

1. **Given** a config.yaml with WiFi transport enabled and a WiFi-capable device, **When** the BoardingPass service starts, **Then** an open WiFi access point is created on the specified interface before the HTTPS server begins listening.
2. **Given** a running WiFi access point created by BoardingPass, **When** a mobile app user opens the app and connects their phone to the hotspot, **Then** the app discovers the BoardingPass service on the hotspot network and includes it in the device list.
3. **Given** a running WiFi access point, **When** the BoardingPass service shuts down (sentinel file, inactivity timeout, or signal), **Then** the WiFi access point is removed and the interface is restored to its prior state.
4. **Given** a config.yaml with WiFi transport enabled but no WiFi interface available, **When** the service starts, **Then** it logs a warning and continues without creating a WiFi access point (non-fatal).
5. **Given** a mobile app user connected to the BoardingPass WiFi hotspot, **When** they scan for devices, **Then** the device is discovered automatically and shown with a "wifi" discovery method indicator.

---

### User Story 2 - Bluetooth PAN Transport (Priority: P2)

A system builder configures a headless Linux device to create a transient Bluetooth Personal Area Network (PAN) for provisioning. When the BoardingPass service starts and a Bluetooth adapter is available, it makes the device discoverable and sets up a Bluetooth PAN. A mobile app user pairs their phone via Bluetooth and discovers the BoardingPass service over the PAN. When provisioning completes, the Bluetooth PAN is torn down and discoverability is disabled.

**Why this priority**: Bluetooth PAN provides a close-range, cable-free transport that works without WiFi hardware on the device. It is especially useful for minimal embedded devices that have Bluetooth but no WiFi adapter. However, Bluetooth PAN has lower bandwidth and more complex pairing than WiFi, making it a secondary option.

**Independent Test**: Can be fully tested by configuring Bluetooth transport in config.yaml, verifying the device becomes discoverable, pairing a phone, discovering the service over PAN, and confirming Bluetooth is cleaned up on shutdown.

**Acceptance Scenarios**:

1. **Given** a config.yaml with Bluetooth transport enabled and a Bluetooth adapter available, **When** the BoardingPass service starts, **Then** the device becomes discoverable, a Bluetooth PAN is created, and the HTTPS server listens on the PAN interface.
2. **Given** a Bluetooth PAN created by BoardingPass, **When** a mobile app user pairs their phone and joins the PAN, **Then** the app discovers the BoardingPass service and includes it in the device list with a "bluetooth" discovery method indicator.
3. **Given** a running Bluetooth PAN, **When** the BoardingPass service shuts down, **Then** the Bluetooth PAN is removed, discoverability is disabled, and the adapter is restored to its prior state.
4. **Given** a config.yaml with Bluetooth transport enabled but no Bluetooth adapter available, **When** the service starts, **Then** it logs a warning and continues without creating a Bluetooth PAN (non-fatal).
5. **Given** a Bluetooth PAN connection, **When** the phone moves out of Bluetooth range, **Then** the app marks the device as offline and can reconnect when back in range.

---

### User Story 3 - USB Tethering Discovery (Priority: P3)

A mobile app user connects their phone to a headless Linux device via USB cable and enables USB tethering on their phone. The phone shares a network connection with the device, and a USB network interface appears on the device. The BoardingPass service detects this interface and listens on it, allowing the app to discover and provision the device over the tethered connection.

**Why this priority**: USB tethering provides a reliable, wired connection that requires no wireless hardware and works in RF-restricted environments. However, it requires physical cable access and user action (enabling tethering), making it less seamless than WiFi or Bluetooth. It serves as a reliable fallback when wireless transports are unavailable.

**Independent Test**: Can be fully tested by connecting a phone via USB, enabling tethering, verifying the service detects the USB network interface, and confirming the app discovers the device over the tethered connection.

**Acceptance Scenarios**:

1. **Given** a config.yaml with USB transport enabled, **When** a phone with tethering enabled is connected via USB and a USB network interface appears, **Then** the BoardingPass service detects the interface and starts listening on it.
2. **Given** a BoardingPass service listening on a USB tethered interface, **When** the mobile app user scans for devices, **Then** the app discovers the service on the tethered network and includes it in the device list with a "usb" discovery method indicator.
3. **Given** a USB tethered connection, **When** the USB cable is disconnected or tethering is disabled, **Then** the service stops listening on the removed interface gracefully.
4. **Given** a config.yaml with USB transport enabled but no USB network interface present, **When** the service starts, **Then** it periodically checks for new USB network interfaces and begins listening when one appears.

---

### User Story 4 - Multi-Transport Simultaneous Operation (Priority: P4)

A system builder configures multiple transports (e.g., Ethernet + WiFi + Bluetooth) in the same config.yaml. The BoardingPass service creates all configured transports and listens on all of them simultaneously. A mobile app user can discover and connect to the device via any available transport.

**Why this priority**: Supporting multiple simultaneous transports maximizes flexibility and ensures the device is reachable regardless of what connectivity the operator has available. This builds on the individual transport implementations.

**Independent Test**: Can be tested by enabling multiple transports in config and verifying the service is reachable via each one independently.

**Acceptance Scenarios**:

1. **Given** a config.yaml with Ethernet, WiFi, and Bluetooth transports all enabled, **When** the service starts, **Then** all three transports are created and the HTTPS server listens on all transport interfaces.
2. **Given** multiple active transports, **When** the mobile app scans for devices, **Then** the same physical device may appear multiple times (once per transport) and is de-duplicated in the device list based on device identity.
3. **Given** multiple active transports, **When** the service shuts down, **Then** all transient transports (WiFi AP, Bluetooth PAN) are torn down, but pre-existing transports (Ethernet, USB) are left unchanged.

---

### Edge Cases

- What happens when a WiFi interface is in use by another process (e.g., as a client)? The service must not take over an interface that is already connected — it should only use interfaces explicitly assigned in config.
- What happens when the Bluetooth adapter is already paired with another device? The service should allow additional pairings without disrupting existing connections.
- What happens when an unauthorized user connects to the open WiFi AP? The SRP authentication layer provides security — transport-level access alone does not grant API access. All API operations still require a valid SRP session.
- What happens when the phone switches from one transport to another mid-session? The session token remains valid as long as TTL has not expired; the app must re-discover the device on the new transport and resume using the existing token.
- What happens when network setup commands (hostapd, bluetoothctl) are not installed or transport systemd units are missing? The service should detect missing dependencies at startup and log an error for that transport, continuing with other configured transports.
- What happens when USB tethering provides an unexpected subnet? The service should listen on the USB interface regardless of subnet and the app should scan the tethered interface's gateway/subnet for the service.

## Requirements *(mandatory)*

### Functional Requirements

#### Service-Side: Transport Management

- **FR-001**: The service configuration MUST support a `transports.wifi` section with fields for `enabled`, `interface`, `ssid`, `password` (optional), `channel`, and `address`.
- **FR-002**: When WiFi transport is enabled and the specified interface is available, the service MUST create a WiFi access point before starting the HTTPS server. The access point MUST be open (no password) by default. If a `password` is configured, it MUST use WPA2-PSK (AES) or stronger.
- **FR-003**: The WiFi access point SSID MUST default to `BoardingPass-<hostname>` when not explicitly configured.
- **FR-004**: The WiFi access point MUST default to open (no password). The system builder MAY configure a WPA2-PSK password for additional transport-level encryption when required by deployment policy.
- **FR-005**: The service configuration MUST support a `transports.bluetooth` section with fields for `enabled`, `adapter`, `device_name`, and `address`.
- **FR-006**: When Bluetooth transport is enabled and the specified adapter is available, the service MUST make the adapter discoverable, create a Bluetooth PAN, and assign the configured IP address before starting the HTTPS server.
- **FR-007**: The Bluetooth device name MUST default to `BoardingPass-<hostname>` when not explicitly configured.
- **FR-008**: The service configuration MUST support a `transports.usb` section with fields for `enabled`, `interface_prefix`, and `address`.
- **FR-009**: When USB transport is enabled, the service MUST monitor for USB network interfaces matching the configured prefix (default: `usb`) and begin listening on them when they appear.
- **FR-010**: When the service shuts down (sentinel file, inactivity timeout, or signal), it MUST stop all transient transport systemd units (WiFi AP, Bluetooth PAN), which in turn tear down the access points and restore interfaces to their prior state. As a safety net, the systemd unit relationships MUST also ensure transient transports are stopped if the BoardingPass service exits unexpectedly.
- **FR-011**: Transport setup failures MUST be non-fatal — the service MUST log a warning and continue starting with remaining transports.
- **FR-012**: The service MUST listen for HTTPS connections on all enabled and successfully created transport interfaces simultaneously.
- **FR-013**: The service MUST manage transient transport lifecycle via systemd units (e.g., a templated `boardingpass-wifi@.service` that runs `hostapd`, a `boardingpass-bt@.service` for Bluetooth PAN) rather than invoking transport setup commands directly from the service process. The BoardingPass service starts the relevant units on startup and stops them on shutdown. This approach leverages systemd's dependency management, process supervision, and cleanup guarantees — if the BoardingPass service crashes, systemd's `BindsTo=`/`PartOf=` directives ensure transient transports are automatically torn down.

#### App-Side: Transport Discovery

- **FR-014**: The mobile app MUST scan for BoardingPass devices on WiFi networks the phone is connected to, detecting whether the current network is a BoardingPass hotspot (by SSID pattern or gateway probing).
- **FR-015**: The mobile app MUST scan for BoardingPass devices via Bluetooth, discovering devices advertising the BoardingPass service and connecting over Bluetooth PAN.
- **FR-016**: The mobile app MUST detect USB tethered connections and scan for BoardingPass devices on the tethered network interface.
- **FR-017**: Each discovered device MUST include a `discoveryMethod` indicator showing the transport used (`wifi`, `bluetooth`, `usb`, `mdns`, `fallback`, `manual`).
- **FR-018**: When the same physical device is discovered via multiple transports, the app MUST de-duplicate it in the device list and automatically select the preferred transport in this order: USB, Bluetooth, WiFi, Ethernet. This order minimizes the chance that the interface used for provisioning is also the interface the device uses for production traffic or enrollment.
- **FR-019**: The app MUST allow the user to manually override the automatic transport selection when multiple transports are available for the same device.

#### Security

- **FR-020**: Transport-level access (WiFi, Bluetooth, USB) MUST NOT bypass SRP authentication — all API requests still require a valid session token.
- **FR-021**: When a WiFi password is configured, it MUST be redacted from all logs. Bluetooth pairing details MUST be redacted from all logs. Consistent with existing secret redaction.
- **FR-022**: When the system builder configures a WiFi password, the access point MUST use WPA2-PSK (AES) or stronger encryption. When no password is configured, the access point operates as an open network — SRP authentication provides the security layer.

### Key Entities

- **Transport**: A network transport layer (Ethernet, WiFi, Bluetooth, USB) with lifecycle management (create, monitor, teardown). Each transport has an enabled state, interface binding, and IP address configuration.
- **TransportManager**: Orchestrates the lifecycle of all configured transports — creates them before the HTTPS server starts and tears them down on shutdown.
- **DiscoveryMethod**: Extended enumeration of how a device was found (`mdns`, `fallback`, `manual`, `wifi`, `bluetooth`, `usb`), displayed in the app's device list.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A mobile app user can discover and provision a device via WiFi hotspot within 2 minutes of powering on the device, without any pre-existing network infrastructure.
- **SC-002**: A mobile app user can discover and provision a device via Bluetooth PAN within 3 minutes, including pairing time.
- **SC-003**: A mobile app user can discover and provision a device via USB tethering within 1 minute of connecting the cable and enabling tethering.
- **SC-004**: When the service shuts down, all transient transports are fully torn down within 10 seconds, leaving no residual WiFi networks or Bluetooth pairings visible.
- **SC-005**: Transport setup failures do not prevent the service from starting — at least one configured transport must be operational for the service to accept connections.
- **SC-006**: The same provisioning workflow (authenticate, inspect, configure, apply) works identically regardless of which transport is used, with no transport-specific steps required from the user.
- **SC-007**: When multiple transports are available for the same device, the app presents a unified device entry rather than duplicate entries.

## Design Rationale

### Open WiFi by default

The WiFi access point defaults to open (no password) because SRP-6a authentication already secures all API access. Adding a WiFi password would create a second credential that users must know, with no meaningful security gain — an attacker who connects to the open AP still cannot call any API without completing SRP authentication. This reduces the provisioning workflow to: connect to WiFi, open app, authenticate with device password. System builders who require transport-level encryption (e.g., compliance policies) can optionally configure WPA2-PSK.

### Transport preference order (USB > Bluetooth > WiFi > Ethernet)

When multiple transports reach the same device, the app prefers dedicated/transient transports over production interfaces: USB and Bluetooth are dedicated point-to-point links that exist solely for provisioning; WiFi AP is transient but shared; Ethernet is typically the device's production interface. By preferring USB > Bluetooth > WiFi > Ethernet, the app minimizes the risk of provisioning traffic contending with the device's production enrollment or management traffic on the same interface.

### systemd units for transport lifecycle

Transport setup and teardown is delegated to systemd units rather than executing commands directly from the BoardingPass process. This was chosen over in-process command execution for several reasons:

- **Crash safety**: If the BoardingPass service crashes, systemd's `BindsTo=`/`PartOf=` directives ensure the WiFi AP and Bluetooth PAN units are automatically stopped — no orphaned access points.
- **Process supervision**: systemd restarts `hostapd` or `bluetoothctl` if they crash, without BoardingPass needing its own process monitoring logic.
- **Separation of concerns**: The BoardingPass service only needs to call `systemctl start/stop` — all transport-specific configuration (hostapd.conf, bluetoothctl scripts) lives in the unit files and their configs, managed by the system builder.
- **Consistent with existing patterns**: The BoardingPass service itself runs as a systemd unit (`boardingpass.service`), so using companion units for transports is a natural extension.
- **Sudo scope reduction**: The service only needs `sudo systemctl start/stop boardingpass-*` rather than sudo access to hostapd, ip, bluetoothctl, and other low-level tools directly.

## Assumptions

- The system builder is responsible for ensuring that WiFi and Bluetooth hardware interfaces specified in config.yaml are present and not in use by other services (e.g., NetworkManager managing WiFi client connections on the same interface).
- The headless Linux device has the necessary userspace tools installed (`hostapd` for WiFi AP, `bluetoothctl`/`busctl` for Bluetooth PAN, `ip` for interface management) and the corresponding systemd unit templates deployed alongside the BoardingPass service. The service does not install these tools or units — they are part of the system image or deployment package.
- WiFi AP mode requires a wireless interface that supports AP mode (most modern WiFi chipsets do). The system builder verifies this when selecting the interface.
- USB tethering is initiated by the phone user — the service only needs to detect and listen on the resulting USB network interface; it does not configure the phone.
- Bluetooth PAN uses the Network Access Point (NAP) profile, which is widely supported on both iOS and Android.
- Multiple transports can operate simultaneously without resource conflicts, assuming distinct interfaces are configured for each.
- The existing SRP authentication and TLS certificate infrastructure applies equally to all transports — no transport-specific authentication changes are needed.
