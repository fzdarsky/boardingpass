# Configuring the Service

BoardingPass reads its configuration from `/etc/boardingpass/config.yaml`. The RPM package installs a minimal default configuration that enables Ethernet transport only. A fully annotated example is available at [`build/config.yaml`](../build/config.yaml).

## Service Settings

```yaml
service:
  port: 8443                     # HTTPS listen port (shared by all transports)
  tls_cert: "/var/lib/boardingpass/tls/server.crt"  # Auto-generated if missing
  tls_key: "/var/lib/boardingpass/tls/server.key"
  inactivity_timeout: "10m"      # Self-terminate after this idle period
  session_ttl: "30m"             # Authenticated session lifetime
  sentinel_file: "/etc/boardingpass/issued"  # Prevents restart after provisioning
  mdns:
    enabled: true                # Announce via mDNS/Bonjour for automatic discovery
```

TLS certificates are auto-generated on first start if the files don't exist. To use your own certificates, place them at the configured paths before starting the service.

## Transports

BoardingPass supports multiple network transports. All transports share the same HTTPS port and TLS certificates. Transient transports (WiFi, Bluetooth, USB) are created when the service starts and torn down when provisioning completes.

SRP-6a authentication protects the API regardless of transport — an open WiFi AP is safe because the API requires a successful SRP handshake before any data is exchanged.

### Ethernet

Listens on existing wired network interfaces. No additional packages required.

```yaml
transports:
  ethernet:
    enabled: true
    interfaces: []               # Interface names (empty = all non-loopback)
    address: ""                  # Bind address (empty = 0.0.0.0)
```

### WiFi Access Point

Creates a temporary WiFi hotspot. The phone connects to it, discovers the device, and provisions it. The AP is torn down when provisioning completes.

**Required packages:** `hostapd`, `dnsmasq`

```yaml
transports:
  wifi:
    enabled: true
    interface: ""                # WiFi interface (empty = auto-detect)
    ssid: ""                     # Network name (default: BoardingPass-<hostname>)
    # password: "changeme123"    # WPA2 password (min 8 chars); omit for open network
    channel: 6                   # WiFi channel
    address: "10.0.0.1"          # AP gateway IP; phones get DHCP in this /24 subnet
```

### Bluetooth PAN

Creates a Bluetooth Personal Area Network (NAP profile) with BLE advertisement for discovery.

**Required packages:** `bluez`

```yaml
transports:
  bluetooth:
    enabled: true
    adapter: "hci0"              # Bluetooth adapter
    device_name: ""              # BLE advertised name (default: BoardingPass-<hostname>)
    address: "10.0.1.1"          # PAN bridge IP (also advertised via BLE)
```

Two systemd units manage this transport: `boardingpass-bt@` for the PAN bridge and `boardingpass-ble@` for BLE advertisement. BLE advertisement failure is non-fatal.

> **FIPS limitation:** Bluetooth pairing requires the kernel `ecdh_generic` module for Secure Simple Pairing (SSP). On FIPS-enabled systems this module may be unavailable, making Bluetooth PAN unusable. BLE advertisement (discovery only) is unaffected. To use BLE as a discovery beacon for WiFi AP on FIPS systems, set `bluetooth.address` to the WiFi AP address:
>
> ```yaml
> bluetooth:
>   enabled: true
>   address: "10.0.0.1"     # Point to WiFi AP instead of PAN bridge
> wifi:
>   enabled: true
>   address: "10.0.0.1"
> ```

### USB Tethering

Detects USB tethering interfaces when a phone is connected via cable. No additional packages or systemd units needed — the service polls `/sys/class/net/` for USB-backed interfaces (drivers: `cdc_ether`, `rndis_host`, `ipheth`).

```yaml
transports:
  usb:
    enabled: true
    interface_prefix: ""         # Restrict to interfaces with this prefix (empty = all USB)
```

### Enabling Multiple Transports

All transports can be enabled simultaneously:

```yaml
transports:
  ethernet:
    enabled: true
  wifi:
    enabled: true
    interface: "wlan0"
  bluetooth:
    enabled: true
  usb:
    enabled: true
```

### Captive Portal Suppression

When a phone connects to the BoardingPass WiFi AP, the OS normally opens a captive portal browser. The service suppresses this by responding to well-known detection URLs:

- iOS: `GET /hotspot-detect.html` returns success HTML
- Android: `GET /generate_204` returns `204 No Content`

This keeps the phone connected without interrupting the user.

## Authentication and Verifier

BoardingPass uses SRP-6a authentication with device-unique passwords. On first start (`boardingpass init`), the service generates a verifier file at `/etc/boardingpass/verifier` using a password generator script.

### Password Generators

The RPM package includes three password generators in `/usr/lib/boardingpass/generators/`:

| Generator | Source | Use Case |
| --------- | ------ | -------- |
| `primary_mac` | Primary network interface MAC address | Default; often printed as barcode on chassis |
| `board_serial` | DMI board serial number | Printed on motherboard label |
| `tpm_ek` | TPM 2.0 endorsement key fingerprint | High-entropy, requires TPM |

The default generator is `primary_mac`. The MAC address is often printed as a barcode on the device chassis, making it easy to scan with the mobile app. To change the generator, edit `/etc/boardingpass/config.yaml` before the first start or delete the verifier file and restart the service.

### Custom Password Generators

A password generator is any executable that prints the password to stdout and exits with code 0. To use a custom generator:

1. Place your script at e.g. `/usr/lib/boardingpass/generators/my_generator`
2. Make it executable: `chmod 755 /usr/lib/boardingpass/generators/my_generator`
3. Delete the existing verifier: `sudo rm /etc/boardingpass/verifier`
4. Restart the service: `sudo systemctl restart boardingpass`

The service runs `boardingpass init` before starting, which invokes the configured generator and creates the verifier.

### Verifier File

The verifier file (`/etc/boardingpass/verifier`) contains the SRP-6a verifier derived from the device password. It is generated automatically and should not be edited manually. The file is permission-restricted (mode 0400, owned by the `boardingpass` user).

To regenerate the verifier (e.g., after changing the password generator):

```bash
sudo rm /etc/boardingpass/verifier
sudo systemctl restart boardingpass
```

## Command Allow-List

Commands that authenticated clients can execute on the device. Each command has an ID, a path to the executable, optional fixed arguments, and a maximum number of additional parameters.

```yaml
commands:
  - id: "set-hostname"
    path: "/usr/lib/boardingpass/scripts/set-hostname.sh"
    args: []
    max_params: 1               # Accepts 1 additional parameter

  - id: "restart-networkmanager"
    path: "/usr/bin/systemctl"
    args: ["restart", "NetworkManager"]
    max_params: 0               # No additional parameters

  - id: "show-status"
    path: "/usr/lib/boardingpass/scripts/show-status.sh"
    args: []
    max_params: 0
    sudo: false                 # Run without sudo (default: true)
```

Commands run via `sudo` by default. Set `sudo: false` for unprivileged commands. The sudoers file (`/etc/sudoers.d/boardingpass`) must include entries for any command that uses sudo.

## File Provisioning Path Allow-List

Controls which filesystem paths the provisioning API can write to. Only files under these paths are accepted:

```yaml
paths:
  allow_list:
    - "/etc/systemd/"
    - "/etc/NetworkManager/"
    - "/etc/chrony.conf"
    - "/etc/chrony.d/"
    - "/etc/hostname"
```

Paths ending with `/` are treated as directory prefixes. Exact paths (without trailing `/`) match a single file.

## Logging

```yaml
logging:
  level: "info"                  # debug, info, warn, error
  format: "json"                 # json (structured) or human (readable)
```

Logs go to stdout/stderr (captured by systemd journal). Sensitive values (passwords, pairing codes, session tokens) are always redacted.
