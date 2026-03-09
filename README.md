# BoardingPass

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![Build](https://github.com/fzdarsky/boardingpass/workflows/Build/badge.svg)](https://github.com/fzdarsky/boardingpass/actions)
[![Coverage](https://img.shields.io/badge/coverage-72%25-yellow.svg)](https://github.com/fzdarsky/boardingpass)

**BoardingPass** is a lightweight, ephemeral bootstrap service for headless Linux devices, exposing a RESTful API over HTTPS with SRP-6a authentication.

## Overview

BoardingPass solves the headless device bootstrap problem by providing a frictionless, secure way to provision configuration and execute system commands on devices without display, keyboard, or pre-configured SSH access.

### Key Features

- **Frictionless Authentication**: SRP-6a protocol with device-unique passwords (no PKI setup required)
- **Ephemeral Operation**: Service automatically terminates after provisioning completion
- **Atomic Configuration**: All-or-nothing configuration bundle application with automatic rollback
- **Minimal Footprint**: < 10MB binary, < 50MB RAM, zero runtime dependencies beyond systemd
- **Multi-Transport**: Built-in WiFi AP, Bluetooth PAN, USB tethering, and Ethernet — all managed automatically
- **Mobile App**: iOS/Android app for device discovery, authentication, and provisioning
- **FIPS 140-3 Ready**: Uses Go stdlib crypto only for compliance

### Use Cases

- Initial provisioning of bootc-based container images
- Headless Raspberry Pi or embedded device setup
- Remote edge device bootstrapping
- Secure device onboarding without pre-shared keys

## Quick Start

### Prerequisites

- Linux system with systemd 250+
- Go 1.25+ (for building from source)
- RHEL 9+ / Rocky Linux 9+ / Debian 12+ / Ubuntu 22.04+ LTS

### Installation

#### From Release Packages

```bash
# RPM-based distributions (RHEL, Rocky, Alma)
sudo rpm -ivh boardingpass-*.rpm

# Debian-based distributions
sudo dpkg -i boardingpass_*.deb
sudo apt-get install -f
```

#### From Source

```bash
# Clone the repository
git clone https://github.com/fzdarsky/boardingpass.git
cd boardingpass

# Build
make build

# Install
sudo cp _output/bin/boardingpass /usr/local/bin/
sudo cp build/boardingpass.service /usr/lib/systemd/system/
sudo cp build/boardingpass.sudoers /etc/sudoers.d/boardingpass
```

### Configuration

Create `/etc/boardingpass/config.yaml`:

```yaml
listen_addr: ":8443"
tls:
  cert_path: "/var/lib/boardingpass/tls/cert.pem"
  key_path: "/var/lib/boardingpass/tls/key.pem"
  auto_generate: true  # Auto-generate self-signed cert at boot

auth:
  verifier_file: "/etc/boardingpass/verifier.json"
  password_generator: "/usr/lib/boardingpass/generators/board_serial"

provisioning:
  allowed_paths:
    - "/etc/systemd/"
    - "/etc/NetworkManager/"
    - "/etc/myapp/"
  max_bundle_size: 10485760  # 10MB

commands:
  allowed:
    - id: "reload-systemd"
      cmd: "systemctl daemon-reload"
    - id: "restart-network"
      cmd: "systemctl restart NetworkManager"

lifecycle:
  inactivity_timeout: "10m"
  sentinel_file: "/etc/boardingpass/issued"
```

### Transient Transports

BoardingPass can automatically create and tear down network transports so that a phone can reach a headless device even when no existing network is available. All transient transports are **ephemeral** — they are created when the service starts and removed when provisioning completes.

> **Security note**: SRP-6a authentication protects the API regardless of transport. An open WiFi AP is safe because the provisioning API requires a successful SRP handshake before any data is exchanged.
>
> **Packaging note**: The BoardingPass RPM/DEB does not depend on any transport-specific packages. The system builder is responsible for including the required packages in the OS image for each transport they enable.

#### Transport Requirements

| Transport | System Packages | Hardware | Systemd Units |
| --- | --- | --- | --- |
| Ethernet | *(none)* | Wired NIC | *(none)* |
| WiFi AP | `hostapd`, `dnsmasq` | WiFi adapter with AP mode | `boardingpass-wifi@`, `boardingpass-dnsmasq@` |
| Bluetooth PAN | `bluez` | Bluetooth adapter (HCI) | `boardingpass-bt@`, `boardingpass-ble@` |
| USB Tethering | *(none)* | USB port | *(none, kernel drivers only)* |

#### WiFi Access Point

Creates a temporary WiFi hotspot. The phone connects to it, discovers the device, and provisions it.

```yaml
# In /etc/boardingpass/config.yaml
transports:
  wifi:
    enabled: true
    interface: "wlan0"         # WiFi interface (empty = auto-detect)
    ssid: "BoardingPass-edge1" # Network name (default: BoardingPass-<hostname>)
    # password: "changeme123"  # Optional WPA2 password (min 8 chars)
    channel: 6                 # WiFi channel
    address: "10.0.0.1"        # AP gateway IP
```

The AP is managed via systemd template units (`boardingpass-wifi@.service`). If the WiFi interface is unavailable, the service logs a warning and continues — it never blocks startup.

#### Bluetooth PAN

Creates a Bluetooth Personal Area Network (NAP profile) with BLE advertisement for discovery. Useful for devices with Bluetooth but no WiFi.

```yaml
transports:
  bluetooth:
    enabled: true
    adapter: "hci0"                  # Bluetooth adapter
    device_name: "BoardingPass-rpi4" # BLE advertised name
    address: "10.0.1.1"             # PAN bridge IP
```

Two systemd units manage this transport: `boardingpass-bt@` for the PAN bridge (uses BlueZ D-Bus API via `busctl`) and `boardingpass-ble@` for BLE advertisement. BLE advertisement failure is non-fatal — the PAN still works if the phone pairs manually.

#### USB Tethering

Detects USB tethering interfaces (phone → device) and automatically starts listening on them. No systemd units or extra packages needed — the service polls `/sys/class/net/` for USB-backed network interfaces using kernel drivers (`cdc_ether`, `rndis_host`, `ipheth`).

```yaml
transports:
  usb:
    enabled: true
    interface_prefix: "usb"  # Matches usb* and rndis* interfaces
```

When a phone enables USB tethering and connects via cable, the service detects the new interface within 2 seconds and begins serving. When the cable is disconnected, the listener is removed automatically.

#### Multiple Transports

All transports can be enabled simultaneously. The HTTPS port and TLS certificates are configured once under `service:` and shared by all transports:

```yaml
service:
  port: 8443
  tls_cert: "/var/lib/boardingpass/tls/server.crt"
  tls_key: "/var/lib/boardingpass/tls/server.key"

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

The mobile app de-duplicates devices discovered via multiple transports and shows the preferred one (USB > Bluetooth > WiFi > mDNS > manual).

#### Captive Portal Suppression

When a phone connects to the BoardingPass WiFi AP, the OS normally opens a captive portal browser. The service suppresses this by responding to well-known captive portal detection URLs:

- iOS: `GET /hotspot-detect.html` → returns `<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>`
- Android: `GET /generate_204` → returns `204 No Content`

This keeps the phone connected without interrupting the user with a browser popup.

### Usage

1. **Start the service** (if not already running):

   ```bash
   sudo systemctl start boardingpass
   ```

2. **Authenticate** using SRP-6a:
   ```bash
   # Step 1: Initiate handshake
   curl -X POST https://device-ip:8443/auth/srp/init \
     -H "Content-Type: application/json" \
     -d '{"username": "operator", "A": "<client_public_key>"}'

   # Step 2: Verify and get session token
   curl -X POST https://device-ip:8443/auth/srp/verify \
     -H "Content-Type: application/json" \
     -d '{"M1": "<client_proof>"}'
   ```

3. **Query device info**:
   ```bash
   curl -X GET https://device-ip:8443/info \
     -H "Authorization: Bearer <session_token>"
   ```

4. **Provision configuration**:
   ```bash
   curl -X POST https://device-ip:8443/configure \
     -H "Authorization: Bearer <session_token>" \
     -H "Content-Type: application/json" \
     -d @config-bundle.json
   ```

5. **Execute system commands**:
   ```bash
   curl -X POST https://device-ip:8443/command \
     -H "Authorization: Bearer <session_token>" \
     -H "Content-Type: application/json" \
     -d '{"command_id": "reload-systemd"}'
   ```

6. **Complete provisioning**:
   ```bash
   curl -X POST https://device-ip:8443/complete \
     -H "Authorization: Bearer <session_token>"
   ```

## CLI Tool

BoardingPass provides the `boarding` CLI tool for easy device provisioning without manual API calls. The CLI handles authentication, TLS certificate validation, session management, and provides a simple interface for all provisioning operations.

### Installation

#### From Binary Release (Recommended)

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -LO https://github.com/fzdarsky/boardingpass/releases/latest/download/boarding-cli_linux_amd64.tar.gz
tar -xzf boarding-cli_linux_amd64.tar.gz
sudo mv boarding /usr/local/bin/

# macOS (arm64/Apple Silicon)
curl -LO https://github.com/fzdarsky/boardingpass/releases/latest/download/boarding-cli_darwin_arm64.tar.gz
tar -xzf boarding-cli_darwin_arm64.tar.gz
sudo mv boarding /usr/local/bin/
```

#### From Source

Requires Go 1.25+ installed:

```bash
# Build CLI from source
make build-cli

# Install to system
sudo cp _output/bin/boarding /usr/local/bin/
```

#### Verify Installation

```bash
boarding --help
```

### Configuration

The CLI supports three configuration methods with clear precedence:

**1. Command-Line Flags (Highest Priority)**
```bash
boarding pass --host 192.168.1.100 --port 8443 --username admin
```

**2. Environment Variables (Medium Priority)**
```bash
export BOARDING_HOST=192.168.1.100
export BOARDING_PORT=8443
boarding pass --username admin
```

**3. Config File (Lowest Priority)**

The config file is automatically created and updated after successful authentication, remembering your last connection:

```bash
# First time: specify host
boarding pass --host 192.168.1.100 --username admin

# Subsequent commands: host is remembered
boarding info
boarding connections
boarding complete
```

You can also manually create/edit `~/.config/boardingpass/config.yaml` (Linux/Unix):
```yaml
host: 192.168.1.100
port: 8443
ca_cert: /path/to/ca.pem  # optional
```

### Quick Start

#### Basic Provisioning Workflow

```bash
# 1. Authenticate (connection is saved for subsequent commands)
boarding pass --host 192.168.1.100 --username admin
# Prompts for password, stores session token and connection details

# 2. Query device information (no --host needed)
boarding info
# Displays CPU, board, TPM, OS, FIPS status in YAML

# 3. Check network interfaces
boarding connections
# Displays network interface details

# 4. Upload configuration files
boarding load /path/to/config-directory
# Uploads all files in directory to device

# 5. Execute allow-listed commands
boarding command echo-test
# Executes command on device, shows output

# 6. Complete provisioning
boarding complete
# Triggers device to finalize and logout
```

#### CI/CD Pipeline (Non-Interactive)

```bash
#!/bin/bash
set -euo pipefail

# Use environment variables for connection
export BOARDING_HOST=${DEVICE_IP}
export BOARDING_PORT=8443

# Authenticate (non-interactive, auto-accept TLS certificate)
boarding pass -y --username admin --password "${DEVICE_PASSWORD}"

# Query device info and save as artifact
boarding info -o json > device-info.json

# Upload configuration
boarding load ./device-configs/edge-node/

# Run post-provision script
boarding command post-provision-script

# Complete provisioning
boarding complete

echo "Provisioning completed successfully"
```

### Global Flags

These flags work with all commands:
- `-y, --assumeyes` - Automatically answer 'yes' to prompts (e.g., TLS certificate acceptance)

### Commands

#### `boarding pass` - Authenticate

Authenticate with the BoardingPass service using SRP-6a protocol.

```bash
boarding pass --host 192.168.1.100 --username admin [--password SECRET]
```

**Flags:**
- `--host` - BoardingPass service hostname or IP (can use `BOARDING_HOST` env var)
- `--port` - BoardingPass service port (default: 8443, can use `BOARDING_PORT` env var)
- `--username` - Username for authentication (prompts if not provided)
- `--password` - Password for authentication (prompts securely if not provided)
- `--ca-cert` - Path to custom CA certificate bundle (can use `BOARDING_CA_CERT` env var)

**Example:**
```bash
# Interactive (prompts for credentials and certificate acceptance)
boarding pass --host 192.168.1.100

# Non-interactive (for scripts/automation)
boarding pass -y --host 192.168.1.100 --username admin --password secret123

# With custom CA certificate
boarding pass --host internal.corp --ca-cert /etc/ssl/ca.pem --username admin
```

#### `boarding info` - Query System Information

Retrieve hardware and software information from the device.

```bash
boarding info [--output yaml|json]
```

**Flags:**
- `--output` - Output format: `yaml` (default) or `json`

**Example:**
```bash
# YAML output (default)
boarding info

# JSON output
boarding info --output json
```

#### `boarding connections` - Query Network Interfaces

List network interfaces and their configuration.

```bash
boarding connections [--output yaml|json]
```

**Flags:**
- `--output` - Output format: `yaml` (default) or `json`

**Example:**
```bash
# View network interfaces
boarding connections

# JSON output for scripting
boarding connections --output json
```

#### `boarding load` - Upload Configuration

Upload configuration files from a local directory to the device.

```bash
boarding load <directory>
```

**Limits:**
- Maximum 100 files
- Maximum 10 MB total size

**Example:**
```bash
# Upload all files from a directory
boarding load /path/to/config

# Files are uploaded recursively maintaining directory structure
```

#### `boarding command` - Execute Command

Execute an allow-listed command on the device.

```bash
boarding command <command-id>
```

**Example:**
```bash
# Execute system command
boarding command systemctl-status

# Command output (stdout/stderr) is displayed
# Exit code is preserved
```

#### `boarding complete` - Complete Provisioning

Signal provisioning completion and terminate the session.

```bash
boarding complete
```

This triggers the BoardingPass service to finalize provisioning and shut down. The session token is deleted locally.

**Example:**
```bash
# Complete the provisioning session
boarding complete
```

### Environment Variables

The CLI supports configuration via environment variables with the following precedence:
**Flags > Environment Variables > Config File**

Available environment variables:
- `BOARDING_HOST` - Default hostname or IP address
- `BOARDING_PORT` - Default port (default: 8443)
- `BOARDING_CA_CERT` - Path to custom CA certificate bundle

**Example:**
```bash
# Set default connection parameters
export BOARDING_HOST=192.168.1.100
export BOARDING_PORT=8443

# Commands will use these defaults
boarding pass --username admin
boarding info
boarding complete
```

### Session Management

The CLI automatically manages session tokens:

- **Storage Location**: `~/.cache/boardingpass/session-<host>-<port>.token`
- **Permissions**: Tokens are saved with restrictive 0600 permissions for security
- **Auto-Loading**: Tokens are automatically loaded for subsequent commands
- **Cleanup**: Tokens are deleted when running `boarding complete`
- **Multiple Devices**: Different tokens are maintained for each host:port combination

View active sessions:
```bash
ls -la ~/.cache/boardingpass/
```

Clear all sessions:
```bash
rm -f ~/.cache/boardingpass/session-*.token
```

### TLS Certificate Handling

The CLI uses **Trust-On-First-Use (TOFU)** for self-signed certificates:

```bash
# First connection prompts for certificate acceptance
boarding pass --host 192.168.1.100 --username admin

# Output:
# WARNING: Unknown TLS certificate fingerprint
#   Host: 192.168.1.100:8443
#   Fingerprint: SHA256:a1b2c3d4...
#
# Do you want to accept this certificate? (yes/no): yes
```

The fingerprint is saved to `~/.config/boardingpass/known_certs.yaml` and future connections validate against it.

**Custom CA Certificates:**
```bash
# Use custom CA bundle
boarding pass --host internal.corp --ca-cert /etc/ssl/ca-bundle.pem --username admin

# Or via environment variable
export BOARDING_CA_CERT=/etc/ssl/ca-bundle.pem
boarding pass --host internal.corp --username admin
```

### Security Best Practices

1. **Avoid saving passwords in config files**:
   ```bash
   # Good: Pass password via flag or let it prompt
   boarding pass --username admin --password "${DEVICE_PASSWORD}"

   # Bad: Don't put passwords in config.yaml
   ```

2. **Use environment variables in CI/CD**:
   ```bash
   export BOARDING_HOST=${DEVICE_IP}
   boarding pass --username admin --password "${DEVICE_SECRET}"
   ```

3. **Verify certificate fingerprints** on first connection in production environments

4. **Use custom CA certificates** for production (avoid TOFU when possible)

5. **Rotate sessions** by running `boarding complete` after provisioning

### Troubleshooting

**Error: "not authenticated" or "no active session"**

*Cause*: No valid session token found or session expired (30-minute TTL)

*Solution*:
```bash
boarding pass --username admin
```

**Error: "connection refused"**

*Cause*: BoardingPass service not running or not reachable

*Solution*:
- Verify service is running: `systemctl status boardingpass`
- Check network connectivity: `ping <host>`
- Verify firewall allows port 8443

**Error: "certificate fingerprint mismatch"**

*Cause*: Server certificate changed (cert rotation or potential MITM attack)

*Solution*:
- If expected, remove old entry from `~/.config/boardingpass/known_certs.yaml`
- Re-run command to accept new certificate
- If unexpected, investigate for security incident

**Error: "command not permitted" or "not in allow-list"**

*Cause*: Command ID not in server-side allow-list

*Solution*:
- Check server configuration for allowed command IDs
- Contact administrator to add command to allow-list

## Mobile App

BoardingPass includes a React Native mobile app (iOS/Android) for discovering and provisioning headless devices from a phone.

### Features

- **Multi-transport discovery**: Finds devices via mDNS, WiFi AP detection, BLE scanning, USB tethering, and manual IP entry
- **SRP-6a authentication**: Secure login using device-specific connection codes
- **Certificate pinning**: Trust-On-First-Use (TOFU) with fingerprint verification
- **Guided provisioning**: Step-by-step wizard for hostname, DNS, NTP, WiFi, and enrollment configuration
- **Review & apply**: Preview all changes before committing to the device

### Getting Started

```bash
# Install dependencies
make install-deps-app

# Generate TypeScript types from OpenAPI spec
make generate-app

# Build and run on iOS simulator
make build-app-ios
make run-app-ios

# Run on a connected physical iOS device
make run-app-ios-device
```

### Device Discovery

The app discovers BoardingPass devices using multiple methods simultaneously:

| Method    | How it works                                          | Priority |
|-----------|-------------------------------------------------------|----------|
| USB       | Detects USB tethering via NetInfo, probes gateway IPs | 1 (best) |
| Bluetooth | BLE scan for BoardingPass service UUID                | 2        |
| WiFi      | Detects BoardingPass SSIDs, probes gateway IP         | 3        |
| mDNS      | Bonjour/Zeroconf `_boardingpass._tcp` service browse  | 4        |
| Manual    | User enters IP:port directly                          | 5        |

When the same device is found via multiple transports, the app **de-duplicates** by certificate fingerprint or device name and shows the highest-priority transport. A transport badge (with icon) in the device list indicates how each device was discovered.

### Requirements

- iOS 15+ or Android 12+
- Node.js 18+ and npm
- Xcode 15+ (for iOS development)
- For mDNS discovery on iOS physical devices: paid Apple Developer Program membership

See [CLAUDE.md](CLAUDE.md) for detailed mobile development workflow, troubleshooting, and coding standards.

## Development

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Run linters
make lint

# Generate coverage report
make coverage
```

### Project Structure

```text
boardingpass/
├── cmd/boardingpass/       # Main service binary
├── internal/               # Private packages
│   ├── api/               # HTTP handlers, routing, captive portal
│   ├── auth/              # SRP-6a authentication
│   ├── transport/         # WiFi, Bluetooth, USB transport handlers
│   ├── provisioning/      # Configuration management
│   ├── command/           # Command execution
│   ├── lifecycle/         # Service lifecycle
│   ├── config/            # YAML config loading and validation
│   └── logging/           # JSON logging with secret redaction
├── pkg/protocol/          # Public protocol definitions
├── mobile/                # React Native mobile app
│   ├── app/               # Expo Router screens
│   └── src/               # Components, services, hooks, types
├── tests/                 # Test suites
├── build/                 # systemd units, sudoers, container config
└── specs/                 # Feature specifications and plans
```

## API Documentation

Full API documentation is available in [docs/api.md](docs/api.md).

OpenAPI 3.1 specification: [specs/001-boardingpass-api/contracts/openapi.yaml](specs/001-boardingpass-api/contracts/openapi.yaml)

## Security

- **Authentication**: SRP-6a with device-unique passwords
- **Transport**: TLS 1.3+ with self-signed or custom certificates
- **Brute Force Protection**: Progressive delays (1s → 2s → 5s → 60s lockout)
- **Session Management**: 30-minute token expiry
- **Path Validation**: Allow-list enforcement for configuration writes
- **Command Execution**: Restricted to predefined allow-list via sudo

See [docs/security.md](docs/security.md) for detailed security considerations.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Architecture

BoardingPass follows a strict constitution focused on:

1. **Frictionless Bootstrapping**: No PKI setup, simple device-label passwords
2. **Ephemeral Operation**: Service terminates after provisioning
3. **Minimal Footprint**: < 10MB binary, < 50MB RAM
4. **Minimal Dependencies**: Go stdlib preferred, static linking
5. **Transport Agnostic**: Protocol-first design
6. **Open Source**: Apache 2.0 license

See [specs/001-boardingpass-api/plan.md](specs/001-boardingpass-api/plan.md) for the complete implementation plan.

## Support

- **Issues**: [GitHub Issues](https://github.com/fzdarsky/boardingpass/issues)
- **Documentation**: [docs/](docs/)
- **Discussions**: [GitHub Discussions](https://github.com/fzdarsky/boardingpass/discussions)

## Roadmap

- [x] Core service implementation (SRP auth, configuration provisioning, command execution)
- [x] CLI tool for device provisioning (`boarding` command)
- [x] Mobile app for iOS/Android (React Native) — device discovery, authentication, guided provisioning
- [x] Transient transport support (WiFi AP, Bluetooth PAN, USB tethering)
- [ ] Enhanced monitoring and observability
- [ ] Integration with bootc ecosystem

---

**Made with ❤️ for the headless device community**
