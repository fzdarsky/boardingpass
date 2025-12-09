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
- [ ] Mobile app for iOS/Android (React Native)
- [ ] Additional transport support (WiFi AP mode, BLE, USB gadget)
- [ ] Enhanced monitoring and observability
- [ ] Integration with bootc ecosystem

---

**Made with ❤️ for the headless device community**
