# systemd Unit Contracts

Defines the interface between the BoardingPass service and the systemd units managing transient transports.

## WiFi AP Unit: `boardingpass-wifi@.service`

**Template instance**: Interface name (e.g., `boardingpass-wifi@wlan0.service`)

**Lifecycle**:
- BoardingPass calls `sudo systemctl start boardingpass-wifi@<interface>`
- BoardingPass calls `sudo systemctl stop boardingpass-wifi@<interface>`
- If BoardingPass exits, systemd stops this unit automatically via `BindsTo=`

**Expected behavior**:
- `ExecStartPre`: Bring interface up, configure IP address
- `ExecStart`: Run `hostapd` with config at `/etc/boardingpass/hostapd-<interface>.conf`
- `ExecStopPost`: Flush IP address, bring interface down

**Config dependency**: `/etc/boardingpass/hostapd-<interface>.conf` must exist before starting.

**Success indicator**: Unit reaches `active (running)` state.

**Failure behavior**: Unit fails to start; BoardingPass logs warning and continues with other transports.

## Bluetooth PAN Unit: `boardingpass-bt@.service`

**Template instance**: Adapter name (e.g., `boardingpass-bt@hci0.service`)

**Lifecycle**:
- BoardingPass calls `sudo systemctl start boardingpass-bt@<adapter>`
- BoardingPass calls `sudo systemctl stop boardingpass-bt@<adapter>`
- If BoardingPass exits, systemd stops this unit automatically via `BindsTo=`

**Expected behavior**:
- `ExecStartPre`: Power on adapter, set discoverable, set device name
- `ExecStart`: Create NAP bridge, assign IP, advertise PAN service
- `ExecStopPost`: Remove NAP bridge, disable discoverability

**Config dependency**: Bluetooth adapter `<adapter>` must be physically present.

**Success indicator**: Unit reaches `active (running)` state and PAN bridge interface exists.

**Failure behavior**: Unit fails to start; BoardingPass logs warning and continues with other transports.

## BLE Advertisement Unit: `boardingpass-ble@.service`

**Template instance**: Adapter name (e.g., `boardingpass-ble@hci0.service`)

**Lifecycle**:
- Started alongside Bluetooth PAN unit (can be combined or separate)
- Advertises BoardingPass BLE service UUID with device connection info

**Expected behavior**:
- `ExecStart`: Start BLE GATT server advertising service UUID and connection details
- GATT characteristics: device name, IP address, port, certificate fingerprint

**Success indicator**: BLE advertisement is visible to scanning devices.

## Sudoers Requirements

The following must be added to `/etc/sudoers.d/boardingpass`:

```
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl start boardingpass-wifi@*
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop boardingpass-wifi@*
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl start boardingpass-bt@*
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop boardingpass-bt@*
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl start boardingpass-ble@*
boardingpass ALL=(ALL) NOPASSWD: /usr/bin/systemctl stop boardingpass-ble@*
```
