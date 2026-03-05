# Research: Enrollment Configuration Wizard

**Feature Branch**: `004-enrollment-flow`
**Date**: 2026-03-02

## RD-01: API Extension Strategy

**Decision**: Minimally extend existing endpoints — add `hostname` to `GET /info` response, add `type`, `speed`, `carrier`, `driver` to `GET /network` interface objects. Extend `POST /command` with optional params. Extend `POST /complete` with optional reboot flag. Fix `/configure` description (sentinel file mention is incorrect).

**Rationale**: The existing four-endpoint API (`/info`, `/network`, `/configure`, `/command`, `/complete`) already provides all the primitives needed for the enrollment wizard. Targeted extensions avoid new endpoint sprawl. Since the API is pre-1.0 and experimental, breaking changes are acceptable if they produce a cleaner API.

**Current `/info` response** (missing hostname):

```json
{
  "tpm": { ... },
  "board": { ... },
  "cpu": { ... },
  "os": { ... },
  "fips_mode": true
}
```

→ Add `hostname` string field at top level.

**Current `/network` interface object** (missing type, speed, carrier, driver):

```json
{
  "name": "eth0",
  "mac_address": "dc:a6:32:12:34:56",
  "link_state": "up",
  "ip_addresses": [...]
}
```

→ Add `type` (string enum), `speed` (int Mbps), `carrier` (bool), `driver` (string).

## RD-02: Command Parameter Extension

**Decision**: Extend the `/command` endpoint to support positional parameters, protected by `--`.

**Config change** — add `max_params` to `CommandDefinition`:

```yaml
commands:
  - id: "set-hostname"
    path: "/usr/bin/hostnamectl"
    args: ["set-hostname", "--static", "--no-ask-password"]
    max_params: 1
```

**Request change** — add optional `params` array to `CommandRequest`:

```json
{
  "id": "set-hostname",
  "params": ["my-device"]
}
```

**Executor behavior**: `sudo /usr/bin/hostnamectl set-hostname --static --no-ask-password -- my-device`

**Validation**:

- Reject if `params` length exceeds `max_params`
- Reject empty params
- Limit individual param length (1024 chars)
- Params passed as exec arguments (not shell-interpolated) → no injection risk
- `--` separator prevents params from being interpreted as options

**Rationale**: Many wizard operations need dynamic arguments (hostname, connection name, interface name). Without params, every dynamic operation would require a staging file + script, turning simple operations into two-step processes. The `--` protection and `max_params` limit maintain security while enabling clean single-call operations.

## RD-03: `/complete` Reboot Extension

**Decision**: Extend `POST /complete` to accept an optional request body with a `reboot` boolean field.

**Behavior**:

- `reboot: false` or omitted → existing behavior: create sentinel, graceful shutdown
- `reboot: true` → create sentinel, send response, schedule reboot after 3-second delay

**Response** (extended status enum):

```json
{
  "status": "rebooting",
  "sentinel_file": "/etc/boardingpass/issued",
  "message": "Provisioning complete. Device will reboot."
}
```

**Rationale**: In deferred mode (same-interface), the app needs the device to reboot after writing config files. A separate reboot command would require calling `/command` after `/complete` — but `/complete` shuts down the service. Adding `reboot` to `/complete` makes it the single, clean final API call. The 3-second delay ensures the HTTP response reaches the mobile app before the connection drops.

## RD-04: Configuration Application Strategy

**Decision**: Two distinct flows leveraging the transactional nature of `POST /configure`.

### Deferred Mode (same interface) — 2 API calls total

The `POST /configure` endpoint writes ALL files in a bundle atomically (all-or-nothing). In deferred mode, the entire device configuration ships as one transaction:

```text
POST /configure → atomic bundle containing:
  - /etc/hostname
  - /etc/NetworkManager/system-connections/boardingpass-enrollment.nmconnection
  - /etc/chrony.d/boardingpass-ntp.conf (if NTP configured)
  - /etc/profile.d/boardingpass-proxy.sh (if proxy configured)
  - /etc/boardingpass/staging/insights.json (if Insights enrollment)
  - /etc/boardingpass/staging/flightctl.json (if FlightCtl enrollment)
  - /etc/systemd/system/boardingpass-enroll.service (if any enrollment)
  - /etc/systemd/system/multi-user.target.wants/boardingpass-enroll.service (symlink, if any enrollment)

POST /complete {"reboot": true} → sentinel + reboot
```

After reboot, the OS naturally applies the config files:

- hostname service reads `/etc/hostname`
- NetworkManager reads `.nmconnection` file
- chrony reads `/etc/chrono.d/*.conf`
- Shell login sources `/etc/profile.d/*.sh`
- Enrollment oneshot service runs after `network-online.target`

### Immediate Mode (different interface) — per-step API calls

Each wizard step writes its config files via `/configure` and applies via `/command`:

| Step | `/configure` writes | `/command` executes |
| ---- | ------------------- | ------------------- |
| 1. Hostname | `/etc/hostname` | `set-hostname -- <name>` |
| 2. Interface | (selection only) | — |
| 3. Addressing | NM connection file | `reload-connection -- <conn>` then `connectivity-test -- <iface> <gw>` |
| 4. Services | NTP + proxy config files | `restart-chronyd` |
| 5. Enrollment | Enrollment staging files | `enroll-insights` / `enroll-flightctl` |
| Done | — | `/complete` (sentinel + shutdown) |

**Rationale**: The transactional `/configure` is a perfect fit for deferred mode — all config files land atomically, guaranteeing the device never has a partially configured state. Immediate mode uses the same file paths but applies them actively via `/command`.

## RD-05: WiFi Scanning via Command

**Decision**: Add `wifi-scan` to the command allow-list (0 params). Script wraps `nmcli` and outputs JSON.

**Script behavior**:

```bash
nmcli -t -f DEVICE,SSID,BSSID,SIGNAL,SECURITY,CHAN,FREQ,RATE device wifi list --rescan yes
```

Transformed to JSON array for easy mobile app parsing.

**Rationale**: WiFi scanning is a read-only query. Implementing WiFi/NetworkManager D-Bus integration in Go would be significant effort with no reuse benefit. A shell script wrapping `nmcli` is simpler, maintainable, and leverages the existing `nmcli` tool that's installed on any system with NetworkManager.

## RD-06: Connectivity Testing via Command

**Decision**: Add `connectivity-test` to the command allow-list (2 params: interface name, gateway IP). Script verifies network connectivity and outputs JSON results.

**Test sequence**:

1. Check interface has an IP address (`ip -j addr show <interface>`)
2. Ping gateway (`ping -c 1 -W 5 <gateway>`)
3. DNS resolution (`getent hosts redhat.com`)
4. Internet reachability (`curl -s -o /dev/null -w '%{http_code}' --max-time 5 http://detectportal.firefox.com/canonical.html`)

**Output**: `{"ip_assigned": true, "gateway_reachable": true, "dns_resolves": true, "internet_reachable": true}`

## RD-07: Network Interface Type & Hardware Detection

**Decision**: Detect interface type from sysfs. Report `driver`, `vendor`, and `model` using sysfs PCI IDs with optional hwdata lookup for human-readable names.

**Type detection logic**:

| Check | Type |
| ----- | ---- |
| `/sys/class/net/<name>/wireless/` exists | `wifi` |
| `/sys/class/net/<name>/bridge/` exists | `bridge` |
| `/sys/class/net/<name>/bonding/` exists | `bond` |
| `/sys/class/net/<name>/device/` absent | `virtual` |
| Default (type=1 in sysfs) | `ethernet` |

**Additional sysfs fields**:

- `speed`: `/sys/class/net/<name>/speed` → int (Mbps), -1 if unknown
- `carrier`: `/sys/class/net/<name>/carrier` → bool (cable present)
- `driver`: `/sys/class/net/<name>/device/driver` → symlink basename (e.g., "e1000e", "iwlwifi")
- `vendor ID`: `/sys/class/net/<name>/device/vendor` → PCI vendor ID (e.g., "0x8086")
- `device ID`: `/sys/class/net/<name>/device/device` → PCI device ID (e.g., "0xa0f0")

**Vendor/Model name resolution**:

1. Read PCI vendor ID from `/sys/class/net/<name>/device/vendor` and device ID from `/sys/class/net/<name>/device/device`
2. If `/usr/share/hwdata/pci.ids` is present, parse it and look up human-readable vendor name (e.g., "Intel Corporation") and device name (e.g., "Wi-Fi 6 AX201")
3. If hwdata is not installed or lookup fails, return the raw hex IDs as strings (e.g., "0x8086", "0xa0f0")
4. Virtual interfaces (no `/sys/class/net/<name>/device/`) report empty vendor and model strings

**pci.ids format** (for parser implementation):

```text
<vendor_hex>  <vendor_name>
\t<device_hex>  <device_name>
```

Example: `8086  Intel Corporation` followed by `\ta0f0  Wi-Fi 6 AX201 160MHz`.

**Rationale**: Sysfs is universally available, needs no dependencies. The hwdata package (`/usr/share/hwdata/pci.ids`) is optional — present on most RHEL 9+ systems but not required, satisfying Constitution III (Minimal Footprint) and IV (Minimal Dependencies). Falling back to raw hex IDs when hwdata is absent ensures the feature always works.

## RD-08: Sentinel File Inconsistency Fix

**Decision**: Fix the OpenAPI spec — remove incorrect sentinel file mention from `/configure` description.

**Findings**:

- OpenAPI line 202: `/configure` says "creates sentinel file" → **WRONG**
- Implementation `configure.go`: does NOT create sentinel → **CORRECT BEHAVIOR**
- Implementation `complete.go`: DOES create sentinel → **CORRECT BEHAVIOR**

The implementation is correct. Only the spec documentation is misleading.

## RD-09: Enrollment in Deferred Mode

**Decision**: Write a systemd oneshot service via `/configure` alongside enrollment staging files. The oneshot runs after `network-online.target` on first boot.

**Oneshot service** (provisioned to `/etc/systemd/system/boardingpass-enroll.service`):

```ini
[Unit]
Description=BoardingPass post-boot enrollment
After=network-online.target
Wants=network-online.target
ConditionPathExists=/etc/boardingpass/staging

[Service]
Type=oneshot
ExecStart=/usr/libexec/boardingpass/enroll-insights.sh
ExecStart=/usr/libexec/boardingpass/enroll-flightctl.sh
ExecStartPost=/bin/systemctl disable boardingpass-enroll.service
RemainAfterExit=no
```

Note: Both ExecStart lines are present; each script checks for its own staging file and exits cleanly if absent.

A symlink at `/etc/systemd/system/multi-user.target.wants/boardingpass-enroll.service` enables it (also written via `/configure`).

**Rationale**: In deferred mode, the enrollment interface isn't configured until after reboot. Enrollment servers may only be reachable via that new interface. A post-boot oneshot is the standard Linux pattern (cloud-init, ignition, firstboot).

**Credential security**: Enrollment scripts read staging files, then immediately delete them — credentials persist on disk only until the enrollment command completes.

## RD-10: Staging File Security for Credentials

**Decision**: Enrollment credentials use staging files (not command params) to keep them off the process table.

**Staging files**:

- `/etc/boardingpass/staging/insights.json` — `{"endpoint": "...", "org_id": "...", "activation_key": "..."}`
- `/etc/boardingpass/staging/flightctl.json` — `{"endpoint": "...", "username": "...", "password": "..."}`

**Security properties**:

- Written with restrictive permissions (mode 0600)
- Scripts delete staging files immediately after reading (before executing enrollment)
- Never visible in process table (`/proc/<pid>/cmdline`)
- In deferred mode, written as part of the atomic `/configure` bundle

**Path allow-list**: `boardingpass/staging/` added to provisioning path allow-list.

## RD-11: Mobile App Wizard Architecture

**Decision**: Single screen (`device/configure.tsx`) with internal step management via `useReducer`. Wizard state in a local `WizardContext` (not global DeviceContext).

**Rationale**: Wizard state is ephemeral (FR-008). A single screen avoids Expo Router navigation complexity and data loss during screen transitions.

**Component structure**:

```text
mobile/src/components/ConfigWizard/
├── WizardContainer.tsx     # Step management, navigation
├── StepIndicator.tsx       # Progress bar
├── HostnameStep.tsx        # Step 1
├── InterfaceStep.tsx       # Step 2 + VLAN
├── WiFiStep.tsx            # Step 2a (conditional)
├── AddressingStep.tsx      # Step 3
├── ServicesStep.tsx        # Step 4
├── EnrollmentStep.tsx      # Step 5
├── ReviewPage.tsx          # Deferred mode review
└── ApplyFeedback.tsx       # Per-step apply status
```

## RD-12: Allow-Listed Commands

| Command ID | Path | Fixed Args | max_params | Purpose |
| ---------- | ---- | ---------- | ---------- | ------- |
| `wifi-scan` | `/usr/libexec/boardingpass/wifi-scan.sh` | — | 0 | Scan WiFi networks |
| `set-hostname` | `/usr/bin/hostnamectl` | `set-hostname --static --no-ask-password` | 1 | Set hostname |
| `reload-connection` | `/usr/libexec/boardingpass/reload-connection.sh` | — | 1 | Reload NM + activate connection |
| `restart-chronyd` | `/usr/bin/systemctl` | `restart chronyd` | 0 | Restart NTP |
| `connectivity-test` | `/usr/libexec/boardingpass/connectivity-test.sh` | — | 2 | Test connectivity (iface, gw) |
| `enroll-insights` | `/usr/libexec/boardingpass/enroll-insights.sh` | — | 0 | Enroll with Insights via `rhc connect` (staging) |
| `enroll-flightctl` | `/usr/libexec/boardingpass/enroll-flightctl.sh` | — | 0 | Enroll with FlightCtl via `flightctl login` (staging) |

All scripts shipped with the BoardingPass package. All output structured JSON on stdout.

**Enrollment commands**:

- **Red Hat Insights**: `rhc connect --organization <ORG_ID> --activation-key <KEY_NAME>` — reads org_id and activation_key from staging file
- **Flight Control**: `flightctl login <URL> --username <USERNAME> --password <PASSWORD>` — reads endpoint, username, password from staging file; alternatively, a curl-based script can be used if the `flightctl` CLI is not available
