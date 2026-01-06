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
- **Transport Agnostic**: RESTful API works over any network transport (Ethernet, WiFi, BLE, USB)
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
  password_generator: "/usr/lib/boardingpass/password-generator"

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

Create `~/.config/boardingpass/config.yaml` (Linux/Unix):
```yaml
host: 192.168.1.100
port: 8443
```

Then run commands without flags:
```bash
boarding pass --username admin
```

### Quick Start

#### Basic Provisioning Workflow

```bash
# 1. Authenticate
boarding pass --host 192.168.1.100 --username admin
# Prompts for password, stores session token

# 2. Query device information
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

# Authenticate (non-interactive)
boarding pass --username admin --password "${DEVICE_PASSWORD}"

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
# Interactive (prompts for credentials)
boarding pass --host 192.168.1.100

# Non-interactive (for scripts)
boarding pass --host 192.168.1.100 --username admin --password secret123

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

```
boardingpass/
├── cmd/boardingpass/       # Main service binary
├── internal/               # Private packages
│   ├── api/               # HTTP handlers and routing
│   ├── auth/              # SRP-6a authentication
│   ├── provisioning/      # Configuration management
│   ├── command/           # Command execution
│   └── lifecycle/         # Service lifecycle
├── pkg/protocol/          # Public protocol definitions
├── tests/                 # Test suites
├── build/                 # Build scripts and configs
└── docs/                  # Documentation
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
- [ ] Mobile app for iOS/Android (React Native)
- [ ] Additional transport support (WiFi AP mode, BLE, USB gadget)
- [ ] Enhanced monitoring and observability
- [ ] Integration with bootc ecosystem

---

**Made with ❤️ for the headless device community**
