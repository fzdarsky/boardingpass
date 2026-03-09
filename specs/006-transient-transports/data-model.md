# Data Model: Transient Transport Provisioning

**Feature Branch**: `006-transient-transports`
**Created**: 2026-03-08

## Entities

### TransportType (Enumeration)

Identifies the kind of network transport.

| Value | Description |
|-------|-------------|
| `ethernet` | Pre-existing wired Ethernet (current default) |
| `wifi` | WiFi access point created by BoardingPass |
| `bluetooth` | Bluetooth PAN created by BoardingPass (service-side); BLE discovery (app-side) |
| `usb` | USB tethering interface (phone provides network to device) |

### TransportState (Enumeration)

Lifecycle state of a transport instance.

| Value | Description |
|-------|-------------|
| `disabled` | Not configured or explicitly disabled in config |
| `starting` | Transport setup in progress (systemd unit starting) |
| `active` | Transport is operational and HTTPS listener is bound |
| `failed` | Transport setup failed (non-fatal, logged as warning) |
| `stopping` | Transport teardown in progress |
| `stopped` | Transport has been torn down |

**State Transitions**:

```
disabled ──(config enabled)──> starting
starting ──(setup success)──> active
starting ──(setup failure)──> failed
active ──(shutdown signal)──> stopping
active ──(interface lost)──> stopping
stopping ──(cleanup done)──> stopped
failed ──(retry/restart)──> starting
```

### Transport (Service-Side)

Represents a single configured transport instance on the BoardingPass service.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| type | TransportType | Yes | Kind of transport |
| enabled | bool | Yes | Whether this transport is enabled in config |
| interface | string | Yes | Network interface name (e.g., `wlan0`, `bt0`, `usb0`) |
| address | string | Yes | IP address to bind on this interface |
| port | int | Yes | HTTPS port (shared across transports, default 8443) |
| state | TransportState | No | Current lifecycle state (runtime only, not persisted) |
| systemdUnit | string | No | Name of the systemd unit managing this transport (for wifi/bluetooth) |

### WiFiTransportConfig (Service-Side Config)

WiFi-specific configuration fields in `config.yaml`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| enabled | bool | Yes | false | Enable WiFi AP transport |
| interface | string | Yes | — | WiFi interface name (e.g., `wlan0`) |
| ssid | string | No | `BoardingPass-<hostname>` | Access point SSID |
| password | string | No | — (open) | WPA2-PSK password; empty = open network |
| channel | int | No | 6 | WiFi channel |
| address | string | No | `10.0.0.1` | IP address on AP interface |

### BluetoothTransportConfig (Service-Side Config)

Bluetooth-specific configuration fields in `config.yaml`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| enabled | bool | Yes | false | Enable Bluetooth PAN transport |
| adapter | string | No | `hci0` | Bluetooth adapter name |
| device_name | string | No | `BoardingPass-<hostname>` | Bluetooth discoverable name |
| address | string | No | `10.0.1.1` | IP address on PAN interface |

### USBTransportConfig (Service-Side Config)

USB tethering configuration fields in `config.yaml`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| enabled | bool | Yes | false | Enable USB tethering detection |
| interface_prefix | string | No | `usb` | Prefix to match USB network interfaces |
| address | string | No | (auto) | IP address; auto-detected from tethered interface |

### DiscoveryMethod (App-Side, Extended Enumeration)

Extended from existing `'mdns' | 'fallback' | 'manual'`.

| Value | Description |
|-------|-------------|
| `mdns` | Discovered via mDNS/Bonjour (existing) |
| `fallback` | Discovered via well-known fallback IP (existing) |
| `manual` | Manually entered by user (existing) |
| `wifi` | Discovered on a BoardingPass WiFi hotspot |
| `bluetooth` | Discovered via BLE advertisement |
| `usb` | Discovered on USB tethered network |

### TransportPreference (App-Side)

Order in which the app prefers transports when the same device is reachable via multiple methods.

| Priority | Transport | Rationale |
|----------|-----------|-----------|
| 1 (highest) | USB | Dedicated point-to-point, no wireless contention |
| 2 | Bluetooth | Close-range, dedicated link |
| 3 | WiFi | Transient AP, shared medium |
| 4 | Ethernet/mDNS | Likely production interface |

## Relationships

```text
TransportSettings (config.yaml)
├── EthernetTransport (existing)
├── WiFiTransportConfig (new)
├── BluetoothTransportConfig (new)
└── USBTransportConfig (new)

TransportManager (runtime)
├── manages 0..N Transport instances
├── starts/stops systemd units for wifi/bluetooth
└── monitors interfaces for usb

Device (mobile app)
├── has 1 DiscoveryMethod (primary)
├── may have N alternate transports
└── de-duplicated by device identity (hostname, certificate fingerprint)
```

## Validation Rules

- `WiFiTransportConfig.interface` must exist in `/sys/class/net/` and support AP mode
- `WiFiTransportConfig.password`, if set, must be at least 8 characters (WPA2 minimum)
- `WiFiTransportConfig.channel` must be 1-14 (2.4GHz) or a valid 5GHz channel
- `BluetoothTransportConfig.adapter` must exist in `/sys/class/bluetooth/`
- `USBTransportConfig.interface_prefix` must be a valid interface name prefix (alphanumeric)
- `Transport.port` must be 1-65535 (shared across all transports)
- At least one transport must be enabled and successfully activated for the service to accept connections
