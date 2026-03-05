# Data Model: Enrollment Configuration Wizard

**Feature Branch**: `004-enrollment-flow`
**Date**: 2026-03-02

## Service-Side Entities (Go)

### Extended: SystemInfo

Extends the existing `SystemInfo` response with a `hostname` field.

| Field | Type | Description |
|-------|------|-------------|
| hostname | string | Current system hostname (from `os.Hostname()`) |
| tpm | TPMInfo | (existing) |
| board | BoardInfo | (existing) |
| cpu | CPUInfo | (existing) |
| os | OSInfo | (existing) |
| fips_mode | bool | (existing) |

### Extended: NetworkInterface

Extends the existing `NetworkInterface` with hardware metadata.

| Field | Type | Description |
|-------|------|-------------|
| name | string | Interface name, e.g., "eth0" (existing) |
| mac_address | string | MAC address, colon-separated hex (existing) |
| link_state | string | "up" or "down" (existing) |
| ip_addresses | []IPAddress | Assigned IP addresses (existing) |
| type | string | Interface type: "ethernet", "wifi", "bridge", "bond", "vlan", "virtual" |
| speed | int | Link speed in Mbps, -1 if unknown |
| carrier | bool | Cable/link detected (true = connected) |
| driver | string | Kernel driver name, e.g., "e1000e", "iwlwifi". Empty if virtual |
| vendor | string | Hardware vendor name (e.g., "Intel Corporation") or hex PCI ID (e.g., "0x8086") if hwdata unavailable. Empty if virtual |
| model | string | Hardware model name (e.g., "Wi-Fi 6 AX201") or hex PCI ID (e.g., "0xa0f0") if hwdata unavailable. Empty if virtual |

### Extended: CommandDefinition

Extends the allow-list configuration to support positional parameters.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Command identifier (existing) |
| path | string | Path to executable (existing) |
| args | []string | Fixed arguments (existing) |
| max_params | int | Maximum positional parameters allowed (default: 0) |

### Extended: CommandRequest

| Field | Type | Description |
|-------|------|-------------|
| id | string | Command ID from allow-list (existing) |
| params | []string | Positional parameters, appended after `--` (optional, default: []) |

### Extended: CompleteRequest (new)

The `/complete` endpoint now accepts an optional request body.

| Field | Type | Description |
|-------|------|-------------|
| reboot | bool | If true, create sentinel and reboot (3s delay). If false/omitted, create sentinel and shut down gracefully. |

### Extended: CompleteResponse

| Field | Type | Description |
|-------|------|-------------|
| status | string | "shutting_down" or "rebooting" (extended enum) |
| sentinel_file | string | Path to created sentinel file (existing) |
| message | string | Human-readable message (existing) |

## Mobile App Entities (TypeScript)

### WizardState

Ephemeral state for the configuration wizard. Lives in a local `WizardContext`, not persisted.

| Field | Type | Description |
|-------|------|-------------|
| currentStep | number | Current wizard step (1-5, 6=review) |
| maxReachedStep | number | Highest step reached (prevents skipping forward) |
| applyMode | "immediate" \| "deferred" \| null | Determined when enrollment interface is selected in Step 2 |
| serviceInterfaceName | string \| null | Interface the BoardingPass service is running on |
| hostname | HostnameConfig | Step 1 config |
| networkInterface | InterfaceConfig | Step 2 config |
| addressing | AddressingConfig | Step 3 config |
| services | ServicesConfig | Step 4 config |
| enrollment | EnrollmentConfig | Step 5 config |
| stepApplyStatus | Record\<number, ApplyStatus\> | Per-step apply result (immediate mode) |

### ApplyStatus

| Field | Type | Description |
|-------|------|-------------|
| status | "pending" \| "applying" \| "success" \| "failed" | Apply state |
| error | string \| null | Error message if failed |
| connectivityResult | ConnectivityResult \| null | Result of connectivity test (Step 3 only) |

### HostnameConfig

| Field | Type | Description |
|-------|------|-------------|
| hostname | string | Desired hostname |

Validation: RFC 1123 — alphanumeric + hyphens, 1–63 chars per label, 253 max total, no leading/trailing hyphens.

### InterfaceConfig

| Field | Type | Description |
|-------|------|-------------|
| interfaceName | string | Selected interface name |
| interfaceType | string | Type: "ethernet", "wifi", etc. |
| vlanId | number \| null | Optional VLAN ID (1–4094) |
| wifi | WiFiConfig \| null | WiFi-specific config (if type is "wifi") |

### WiFiConfig

| Field | Type | Description |
|-------|------|-------------|
| ssid | string | Selected SSID |
| bssid | string | Selected BSSID (for multi-AP disambiguation) |
| security | string | Security type: "open", "wpa2", "wpa3", etc. |
| password | string \| null | WiFi password (null for open networks) |

### WiFiNetwork (read-only, from wifi-scan command)

| Field | Type | Description |
|-------|------|-------------|
| device | string | Interface name (e.g., "wlan0") |
| ssid | string | Network name |
| bssid | string | Access point MAC address |
| signal | number | Signal strength (0-100) |
| security | string | Security type |
| channel | number | WiFi channel number |
| frequency | number | Frequency in MHz |
| band | string | Frequency band: "2.4 GHz", "5 GHz", or "6 GHz" (derived from frequency) |
| rate | string | Data rate (e.g., "54 Mbit/s") |

**Band derivation from frequency**: 2400–2483 MHz → "2.4 GHz", 5150–5850 MHz → "5 GHz", 5925–7125 MHz → "6 GHz".

### AddressingConfig

| Field | Type | Description |
|-------|------|-------------|
| ipv4 | IPv4Config | IPv4 configuration |
| ipv6 | IPv6Config | IPv6 configuration |

### IPv4Config

| Field | Type | Description |
|-------|------|-------------|
| method | "dhcp" \| "static" | Addressing method |
| address | string \| null | Static IP address (required if static) |
| subnetMask | string \| null | Subnet mask in dotted-decimal or CIDR (required if static) |
| gateway | string \| null | Gateway IP (required if static) |
| dnsAuto | boolean | Automatically configure DNS (default: true) |
| dnsPrimary | string \| null | Primary DNS server (required if dnsAuto=false) |
| dnsSecondary | string \| null | Secondary DNS server (optional) |

### IPv6Config

| Field | Type | Description |
|-------|------|-------------|
| method | "dhcp" \| "static" \| "disabled" | Addressing method |
| address | string \| null | IPv6 address with prefix, e.g., "2001:db8::1/64" (required if static) |
| gateway | string \| null | Gateway IP (required if static) |
| dnsAuto | boolean | Automatically configure DNS (default: true) |
| dnsPrimary | string \| null | Primary IPv6 DNS server (required if dnsAuto=false) |
| dnsSecondary | string \| null | Secondary IPv6 DNS server (optional) |

### ServicesConfig

| Field | Type | Description |
|-------|------|-------------|
| ntp | NTPConfig | NTP configuration |
| proxy | ProxyConfig \| null | HTTP proxy (null = no proxy) |

### NTPConfig

| Field | Type | Description |
|-------|------|-------------|
| mode | "automatic" \| "manual" | NTP mode |
| servers | string[] | Manual NTP server hostnames/IPs (required if manual) |

### ProxyConfig

| Field | Type | Description |
|-------|------|-------------|
| hostname | string | Proxy hostname or IP |
| port | number | Proxy port (1–65535) |
| username | string \| null | Optional auth username |
| password | string \| null | Optional auth password |

### EnrollmentConfig

| Field | Type | Description |
|-------|------|-------------|
| insights | InsightsConfig \| null | Red Hat Insights config (null = disabled) |
| flightControl | FlightControlConfig \| null | Flight Control config (null = disabled) |

### InsightsConfig

| Field | Type | Description |
|-------|------|-------------|
| endpoint | string | Service endpoint (default: `https://cert-api.access.redhat.com`) |
| orgId | string | Organisation ID |
| activationKey | string | Activation Key |

### FlightControlConfig

| Field | Type | Description |
|-------|------|-------------|
| endpoint | string | Service endpoint |
| username | string | Username |
| password | string | Password |

### ConnectivityResult (from connectivity-test command)

| Field | Type | Description |
|-------|------|-------------|
| ipAssigned | boolean | Interface has an IP address |
| gatewayReachable | boolean | Gateway responds to ping |
| dnsResolves | boolean | DNS resolution works |
| internetReachable | boolean | Can reach external host |

## Entity Relationships

```
WizardState
├── HostnameConfig (1:1)
├── InterfaceConfig (1:1)
│   └── WiFiConfig (0:1, conditional on type="wifi")
├── AddressingConfig (1:1)
│   ├── IPv4Config (1:1)
│   └── IPv6Config (1:1)
├── ServicesConfig (1:1)
│   ├── NTPConfig (1:1)
│   └── ProxyConfig (0:1)
├── EnrollmentConfig (1:1)
│   ├── InsightsConfig (0:1)
│   └── FlightControlConfig (0:1)
└── ApplyStatus (1:N, per step)
    └── ConnectivityResult (0:1, step 3 only)
```

## State Transitions

### WizardState.applyMode

```
null → "immediate"   (when selected interface ≠ service interface)
null → "deferred"    (when selected interface = service interface)
```

Determined once in Step 2, immutable for remainder of wizard. If user goes back to Step 2 and changes interface, mode is re-evaluated.

### ApplyStatus.status (immediate mode only)

```
"pending" → "applying" → "success"
                       → "failed" → "applying" (retry)
```
