# Mobile App Discovery Contracts

Defines the interface contracts for transport-specific device discovery in the mobile app.

## DiscoveryMethod Enumeration

Extended from existing values:

```typescript
type DiscoveryMethod = 'mdns' | 'fallback' | 'manual' | 'wifi' | 'bluetooth' | 'usb';
```

## WiFi Discovery Contract

**Trigger**: App detects phone is connected to a WiFi network.

**Detection method**:
1. Read current SSID via `@react-native-community/netinfo`
2. Match SSID against pattern `BoardingPass-*`
3. If SSID unavailable (no entitlement): probe gateway IP with HTTPS HEAD to `/`
4. If response received: create Device with `discoveryMethod: 'wifi'`

**Device fields populated**:
- `host`: Gateway IP address of current WiFi network
- `port`: 9455 (default)
- `discoveryMethod`: `'wifi'`
- `name`: Extracted from SSID (`BoardingPass-<name>` -> `<name>`) or from `/info` endpoint

**Permissions required**:
- iOS: WiFi entitlement + location permission (for SSID access)
- iOS fallback (no entitlement): Gateway probing only (no SSID matching)
- Android: `ACCESS_FINE_LOCATION` (for SSID access on Android 8+)

## Bluetooth (BLE) Discovery Contract

**Trigger**: App initiates BLE scan.

**Detection method**:
1. Scan for BLE advertisements with BoardingPass service UUID
2. Connect to GATT server, read device info characteristic
3. Extract: device name, IP address, port, certificate fingerprint
4. Create Device with `discoveryMethod: 'bluetooth'`

**BLE Service UUID**: `BBBBBBBB-BBBB-BBBB-BBBB-BBBBBBBBBBBB` (placeholder - to be assigned)

**GATT Characteristics**:
| Characteristic | UUID | Type | Description |
|---------------|------|------|-------------|
| Device Name | `0001` | Read | Human-readable device name |
| IP Address | `0002` | Read | Device IP address (on reachable transport) |
| Port | `0003` | Read | HTTPS port number |
| Cert Fingerprint | `0004` | Read | SHA-256 fingerprint for TOFU pinning |

**Device fields populated**:
- `host`: IP address from BLE characteristic
- `port`: Port from BLE characteristic
- `discoveryMethod`: `'bluetooth'`
- `name`: Device name from BLE characteristic

**Permissions required**:
- iOS: `NSBluetoothAlwaysUsageDescription`
- Android: `BLUETOOTH_SCAN`, `BLUETOOTH_CONNECT`, `ACCESS_FINE_LOCATION`

## USB Tethering Discovery Contract

**Trigger**: App detects network type change to USB/Ethernet tethering.

**Detection method**:
1. Detect network type via `@react-native-community/netinfo`
2. When connected via USB Ethernet (type `'other'` or `'ethernet'` on mobile):
   - Probe well-known tethering gateway IPs: `172.20.10.1` (iOS), `192.168.42.1` (Android)
   - Send HTTPS HEAD to each candidate on port 9455
3. If response received: create Device with `discoveryMethod: 'usb'`

**Device fields populated**:
- `host`: Tethering gateway IP that responded
- `port`: 9455 (default)
- `discoveryMethod`: `'usb'`
- `name`: From `/info` endpoint after connection

**Permissions required**:
- No special permissions (USB tethering is a standard network connection)

## Transport Preference Order

When the same device is discovered via multiple methods, prefer:

| Priority | DiscoveryMethod | Rationale |
|----------|----------------|-----------|
| 1 | `usb` | Dedicated wired link, no contention |
| 2 | `bluetooth` | Close-range dedicated link |
| 3 | `wifi` | Transient AP, shared medium |
| 4 | `mdns` / `fallback` | Likely production interface |
| 5 | `manual` | User-specified, lowest auto-preference |

## De-duplication Strategy

Devices discovered via multiple transports are de-duplicated by:
1. **Certificate fingerprint** (primary): Same TLS cert = same device
2. **Hostname from `/info`** (secondary): Same hostname = likely same device
3. **Device name from mDNS TXT record** (fallback): Same service name

The app stores all discovered transports for a device and displays the preferred one, with an option for the user to switch transports.
