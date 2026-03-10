<img src="docs/assets/boardingpass-logo.svg" alt="BoardingPass Logo" width="400">

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![Service Build](https://github.com/fzdarsky/boardingpass/actions/workflows/build.yaml/badge.svg)](https://github.com/fzdarsky/boardingpass/actions/workflows/build.yaml)
[![Service Tests](https://github.com/fzdarsky/boardingpass/actions/workflows/unit-test.yaml/badge.svg)](https://github.com/fzdarsky/boardingpass/actions/workflows/unit-test.yaml)
[![App Builds&Tests](https://github.com/fzdarsky/boardingpass/actions/workflows/mobile-ci.yml/badge.svg)](https://github.com/fzdarsky/boardingpass/actions/workflows/mobile-ci.yml)

BoardingPass is a lightweight bootstrap service for headless Linux devices.

> [!NOTE]
> This is an experimental project. Use at your own risk — no maintenance or support is guaranteed.

## Overview

Headless devices — servers, edge nodes, embedded systems — often lack displays, keyboards, and pre-configured network access. Before they can onboard into management services like [Red Hat Insights](https://www.redhat.com/en/technologies/management/insights) or [Flight Control](https://flightctl.io/), they need network connectivity and credentials. BoardingPass bridges that gap.

A minimal service runs on the headless device. You connect to it from an iOS app or the `boarding` CLI tool to push configuration, run commands, and enroll the device — all authenticated and encrypted via [SRP-6a](https://en.wikipedia.org/wiki/Secure_Remote_Password_protocol), no SSH or PKI setup required. Once provisioning completes, the service disables itself and becomes inert.

### Features

- **Multi-transport provisioning** — connect over BLE, WiFi, USB, or Ethernet
- **Device discovery** — automatic discovery via mDNS or manual addition by IP address
- **Simple authentication** — scan a QR code or barcode on the device, or enter a pre-shared secret
- **Device inventory** — query system info (board, CPU, TPM, OS, FIPS status) and network interfaces
- **Network configuration** — set hostname, onboarding interface, network access point, IPv4/v6 addresses, time server, and network proxy
- **Enrollment** — enroll into [Red Hat Insights](https://www.redhat.com/en/technologies/management/insights) or [Flight Control](https://github.com/flightctl/flightctl)

## Installing the Service

### From RPM

RPM packages are available from the [GitHub Releases](https://github.com/fzdarsky/boardingpass/releases) page.

```bash
sudo dnf install boardingpass-<version>.rpm
```

The RPM has no transport-specific dependencies. Install additional packages depending on which transports you enable:

| Transport | Additional Packages |
| --------- | ------------------- |
| Ethernet | *(none)* |
| WiFi AP | `hostapd`, `dnsmasq` |
| Bluetooth | `bluez` |
| USB | *(none)* |

### Starting the Service

```bash
sudo systemctl enable --now boardingpass
```

On first start, the service generates a TLS certificate and an SRP verifier derived from a device-unique password. By default, the password is the MAC address of the device's primary network interface — this is often printed as a barcode on the chassis and can be scanned directly by the app.

If using the Ethernet transport, open port 8443/tcp in your firewall:

```bash
sudo firewall-cmd --add-port=8443/tcp
sudo firewall-cmd --runtime-to-permanent
```

Transient transports (WiFi AP, Bluetooth, USB) create their own network and don't require firewall changes.

### Configuration

See [Configuring the Service](docs/configuring-the-service.md) for transport settings, password generators, command allow-lists, and all other options.

## Installing and Using the App

The BoardingPass iOS app discovers nearby devices, authenticates, and walks you through provisioning with a guided wizard.

1. Install the app from TestFlight (link TBD)
2. The app automatically discovers devices via BLE, WiFi, USB, or mDNS
3. Scan the QR code or barcode on the device to authenticate
4. Configure hostname, network, time server, and enrollment
5. Review and apply changes

For app development, see [mobile/README.md](mobile/README.md).

## Installing and Using the CLI

The `boarding` CLI tool provisions devices from the command line.

### Installation

```bash
# From release archive
curl -LO https://github.com/fzdarsky/boardingpass/releases/latest/download/boarding-cli_linux_amd64.tar.gz
tar -xzf boarding-cli_linux_amd64.tar.gz
sudo install -m 755 boarding /usr/local/bin/
```

### Quick Start

```bash
# Authenticate (prompts for password; connection is saved)
boarding pass --host 192.168.1.100

# Query device info
boarding info

# Check network interfaces
boarding connections

# Upload configuration files
boarding load /path/to/config-directory

# Execute an allow-listed command
boarding command set-hostname

# Complete provisioning (service shuts down)
boarding complete
```

For the full command reference, flags, environment variables, and CI/CD examples, see [CLI Reference](docs/cli-reference.md).

## Security

BoardingPass uses SRP-6a authentication with device-unique passwords, TLS 1.3+, progressive brute-force protection, and allow-list enforcement for both file writes and command execution. See [Security](docs/security.md) for details.

## Documentation

- [Configuring the Service](docs/configuring-the-service.md) — transport settings, authentication, commands, paths
- [CLI Reference](docs/cli-reference.md) — full command documentation
- [API Documentation](docs/api.md) — REST API reference
- [Security](docs/security.md) — security model and considerations
- [Development Guide](docs/development.md) — building, testing, contributing

## License

Apache License 2.0 — see [LICENSE](LICENSE).
