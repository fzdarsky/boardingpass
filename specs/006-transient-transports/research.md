# Research: Transient Transport Provisioning

**Feature Branch**: `006-transient-transports`
**Created**: 2026-03-08

## R1: Bluetooth PAN Support on iOS and Android

### Decision
Bluetooth PAN (NAP profile) is **not feasible on iOS**. iOS has no public API for initiating Bluetooth PAN connections, and the NAP profile is not exposed to third-party apps. Android supports Bluetooth PAN via `BluetoothPan` system API, but it requires `BLUETOOTH_PRIVILEGED` permission (system apps only) on Android 12+.

### Rationale
- iOS CoreBluetooth framework only exposes BLE (Bluetooth Low Energy), not Classic Bluetooth profiles like PAN/NAP
- There is no public iOS API to join a Bluetooth PAN — this is a hard platform limitation, not a permissions issue
- Android's `BluetoothPan` class exists but is marked `@SystemApi` since Android 12, making it inaccessible to regular Play Store apps
- Even on older Android versions, PAN connection required hidden APIs via reflection, which is fragile and rejected by Google Play policies

### Alternative: BLE with Custom GATT Service
A practical alternative is using BLE (Bluetooth Low Energy) with a custom GATT service:
- The Linux device advertises a BLE service with a characteristic containing the device's IP/port information
- The mobile app scans for the BLE advertisement, reads the connection info, and connects via the IP network
- This is a **discovery mechanism** rather than a transport — actual API traffic still flows over IP (WiFi, USB, or another network path)
- BLE is fully supported on both iOS (CoreBluetooth) and Android (android.bluetooth.le)
- React Native libraries: `react-native-ble-plx` or `react-native-ble-manager`

### Alternatives Considered
| Alternative | Why Rejected |
|-------------|-------------|
| Bluetooth PAN (NAP) | No iOS support, restricted Android API |
| Bluetooth SPP (Serial Port Profile) | iOS does not expose RFCOMM to third-party apps |
| Wi-Fi Direct / P2P | iOS does not support Wi-Fi Direct; Android support is inconsistent |
| BLE data transfer (full API over BLE) | BLE MTU limits (512 bytes) make it impractical for config bundles; use BLE for discovery only |

### Impact on Spec
FR-005, FR-006, FR-015 and User Story 2 should be reframed: the service-side still creates a Bluetooth PAN for IP connectivity, but the mobile app uses BLE for **discovery** rather than connecting to the PAN directly. The mobile app discovers the device's IP address via BLE advertisement, then connects over whatever IP transport is available (WiFi AP, USB tethering, or if the PAN is reachable on Android, over PAN). This is consistent with the transport-agnostic architecture.

---

## R2: WiFi AP Creation on Linux with hostapd and systemd

### Decision
Use `hostapd` managed by a systemd template unit (`boardingpass-wifi@.service`) for WiFi AP creation. The template unit takes the interface name as instance parameter.

### Rationale
- `hostapd` is the standard Linux WiFi AP daemon, available on all enterprise distributions (RHEL 9+, Fedora, Ubuntu)
- systemd template units (`boardingpass-wifi@<interface>.service`) allow per-interface configuration
- `BindsTo=boardingpass.service` ensures the AP is torn down if BoardingPass exits
- `hostapd` configuration supports both open and WPA2-PSK modes via a single config template
- The service only needs `sudo systemctl start/stop boardingpass-wifi@<iface>` — minimal privilege escalation

### Configuration Template
```ini
# /etc/boardingpass/hostapd-%i.conf (generated or provided by system builder)
interface=%i
driver=nl80211
ssid=BoardingPass-<hostname>
hw_mode=g
channel=6
ieee80211n=1

# Open mode (default) — no auth lines needed
# WPA2-PSK mode (when password configured):
# wpa=2
# wpa_passphrase=<password>
# wpa_key_mgmt=WPA-PSK
# rsn_pairwise=CCMP
```

### systemd Unit Template
```ini
# /etc/systemd/system/boardingpass-wifi@.service
[Unit]
Description=BoardingPass WiFi AP on %i
BindsTo=boardingpass.service
After=boardingpass.service
PartOf=boardingpass.service

[Service]
Type=simple
ExecStartPre=/usr/sbin/ip link set %i up
ExecStart=/usr/sbin/hostapd /etc/boardingpass/hostapd-%i.conf
ExecStartPost=/usr/sbin/ip addr add <address>/24 dev %i
ExecStopPost=/usr/sbin/ip addr flush dev %i
ExecStopPost=/usr/sbin/ip link set %i down
```

### DHCP for WiFi AP Clients
Phones connecting to the WiFi AP need a DHCP-assigned IP address. Options:
- **dnsmasq** (recommended): Lightweight DNS/DHCP server, available on RHEL 9+, can run as a companion systemd unit alongside hostapd. Config: `dhcp-range=10.0.0.2,10.0.0.50,255.255.255.0,12h`
- **systemd-networkd DHCPServer**: Built-in to systemd-networkd, simpler but less flexible
- The hostapd unit's `ExecStartPost` or a companion `boardingpass-dhcp@.service` unit can manage this

### Alternatives Considered
| Alternative | Why Rejected |
|-------------|-------------|
| NetworkManager AP mode | Heavier dependency, less predictable on headless systems |
| iw/hostapd direct exec | No process supervision, no crash recovery, orphaned APs on crash |
| wpa_supplicant AP mode | Limited AP features compared to hostapd |

---

## R3: USB Tethering Interface Detection

### Decision
Detect USB tethering interfaces by monitoring `/sys/class/net/` for new interfaces matching configurable prefixes. iOS and Android use different subnet ranges.

### Rationale
- iOS USB tethering creates interfaces named `eth*` or `usb*` with subnet `172.20.10.0/28` (gateway `172.20.10.1`)
- Android USB tethering creates `usb0` or `rndis0` with subnet `192.168.42.0/24` (gateway `192.168.42.129`) or `192.168.43.0/24`
- The Linux kernel udev/netlink events can signal new interface creation
- Polling `/sys/class/net/` every 2-3 seconds is a simple, reliable approach without additional dependencies
- The service should listen on any new interface matching the configured prefix, regardless of subnet

### Detection Strategy
1. On startup: enumerate `/sys/class/net/` for existing USB interfaces
2. Periodically (every 2s): re-scan for new interfaces matching prefix
3. When new interface detected: wait for IP assignment (poll for address), then bind HTTPS listener
4. When interface disappears: gracefully close listener on that interface

### Interface Identification
```
# Check if interface is USB-backed:
readlink /sys/class/net/<iface>/device/driver
# Contains "cdc_ether", "rndis_host", or "ipheth" for USB tethering
```

### Alternatives Considered
| Alternative | Why Rejected |
|-------------|-------------|
| udev rules + systemd units | More complex setup, harder for system builders |
| netlink socket monitoring | Requires additional Go dependency or raw socket handling |
| Fixed IP assumption | iOS and Android use different subnets; must be dynamic |

---

## R4: WiFi/Transport Discovery in React Native Mobile App

### Decision
Use `@react-native-community/netinfo` for detecting current network type and SSID. Use gateway probing (HTTPS HEAD to gateway IP) for BoardingPass detection. For BLE discovery, use `react-native-ble-plx`.

### Rationale
- `@react-native-community/netinfo` provides `type` (wifi/cellular/ethernet), `details.ssid`, and `details.ipAddress`
- iOS requires `com.apple.developer.networking.wifi-info` entitlement + location permission for SSID access
- SSID pattern matching (`BoardingPass-*`) identifies BoardingPass WiFi APs
- Gateway probing works regardless of SSID access — send HTTPS HEAD to the gateway IP and check response
- iOS shows "No Internet Connection" banner for open WiFi without internet, but the connection remains functional
- `react-native-ble-plx` supports BLE scanning on both iOS and Android with a consistent API

### iOS Permissions Required
- WiFi SSID: `NSLocationWhenInUseUsageDescription` + WiFi entitlement (paid Apple Developer account)
- BLE scanning: `NSBluetoothAlwaysUsageDescription` or `NSBluetoothPeripheralUsageDescription`
- Without paid account: fallback to gateway probing (no SSID needed) + manual IP entry

### Discovery Flow
1. **WiFi detection**: Check if connected to WiFi → check SSID pattern → probe gateway for BoardingPass service
2. **BLE detection**: Scan for BLE advertisements with BoardingPass service UUID → read device info characteristic → connect via advertised IP
3. **USB detection**: Check if connected via "USB Ethernet" → probe well-known USB tethering subnets (172.20.10.x, 192.168.42.x)

### React Native Libraries
| Library | Purpose | Platform Support |
|---------|---------|-----------------|
| `@react-native-community/netinfo` | Network type, SSID, IP | iOS + Android |
| `react-native-ble-plx` | BLE scanning/GATT | iOS + Android |
| Existing `axios` | HTTP probing | iOS + Android |

### Alternatives Considered
| Alternative | Why Rejected |
|-------------|-------------|
| `react-native-wifi-reborn` | iOS can't programmatically connect to WiFi; only useful for scanning |
| Native modules for each transport | Excessive complexity; existing libraries cover needs |
| Multipeer Connectivity (iOS) | Apple-only, not cross-platform |

### iOS Captive Portal Behavior
When connecting to an open WiFi AP without internet access, iOS will:
1. Show a "No Internet Connection" notification/banner
2. Keep the WiFi connection active (does not auto-disconnect)
3. May show a captive portal WebView — can be suppressed by responding to captive portal probes
4. The app can still make HTTPS requests to the BoardingPass service on the local network

The BoardingPass service should respond to Apple's captive portal detection URL (`/hotspot-detect.html`) with a `<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>` response to prevent the captive portal popup.
