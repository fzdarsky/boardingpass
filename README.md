<img src="docs/assets/boardingpass-logo.svg" alt="BoardingPass Logo" width="300">

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![Service Tests](https://github.com/fzdarsky/boardingpass/actions/workflows/test-service.yaml/badge.svg)](https://github.com/fzdarsky/boardingpass/actions/workflows/test-service.yaml)
[![Service Build](https://github.com/fzdarsky/boardingpass/actions/workflows/build-service.yaml/badge.svg)](https://github.com/fzdarsky/boardingpass/actions/workflows/build-service.yaml)
[![App Tests](https://github.com/fzdarsky/boardingpass/actions/workflows/test-app.yaml/badge.svg)](https://github.com/fzdarsky/boardingpass/actions/workflows/test-app.yaml)
[![App Build](https://github.com/fzdarsky/boardingpass/actions/workflows/build-app.yaml/badge.svg)](https://github.com/fzdarsky/boardingpass/actions/workflows/build-app.yaml)

BoardingPass is a lightweight bootstrap service for headless Linux devices.

> [!NOTE]
> This is an experimental project. Use at your own risk — no maintenance or support is guaranteed.

## Overview

Headless devices — servers, edge nodes, embedded systems — often lack displays, keyboards, and pre-configured network access. Before they can onboard into management services like [Red Hat Insights](https://www.redhat.com/en/technologies/management/insights) or [Flight Control](https://flightctl.io/), they need network connectivity and credentials. BoardingPass bridges that gap.

BoardingPass provides a minimal service that runs on the headless device. You connect to it from the BoardingPass iOS app or the `boarding` CLI tool to push configuration, run commands, and enroll the device — all authenticated via [SRP-6a](https://en.wikipedia.org/wiki/Secure_Remote_Password_protocol) and encrypted via HTTPS, no SSH or PKI setup required. Once provisioning completes, the service disables itself and becomes inert.

### Features

- **Multi-transport provisioning** — connect over BLE, WiFi, USB, or Ethernet
- **Device discovery** — automatic discovery via mDNS or manual addition by IP address
- **Simple authentication** — scan a QR code or barcode on the device, or enter a pre-shared secret
- **Device inventory** — query system info (board, CPU, TPM, OS, FIPS status) and network interfaces
- **Network configuration** — set hostname, onboarding interface, network access point, IPv4/v6 addresses and DNS, time server, and network proxy
- **Enrollment** — enroll into [Red Hat Insights](https://www.redhat.com/en/technologies/management/insights) or [Flight Control](https://github.com/flightctl/flightctl)

## Installing the BoardingPass Service

### On RPM-based distributions

1. Install the latest RPM directly from [GitHub Releases](https://github.com/fzdarsky/boardingpass/releases):

    ```sh
    VERSION=$(curl -s https://api.github.com/repos/fzdarsky/boardingpass/releases/latest | jq -r .tag_name)
    sudo dnf install -y "https://github.com/fzdarsky/boardingpass/releases/download/${VERSION}/boardingpass-${VERSION#v}-1.$(uname -m).rpm"
    ```

2. If you want to provision through a WiFi or Bluetooth transport, you need to install additional packages:

    ```sh
    # For WiFi AP transport:
    sudo dnf install -y hostapd dnsmasq

    # For Bluetooth/BLE transport:
    sudo dnf install -y bluez

    # For USB transport:
    sudo dnf install -y epel-release
    sudo dnf install -y libimobiledevice
    # You also need a very recent version of usbmuxd, which is not available on RHEL9

3. Start the service:

    ```bash
    sudo systemctl enable --now boardingpass
    ```

    On first start, the service generates a TLS certificate and an SRP verifier derived from a device-unique password. By default, that password is the MAC address of the device's primary network interface — this is often printed as a barcode on the chassis and can be scanned directly by the app.

4. If using the Ethernet transport, open port `9455/tcp` in your firewall:

    ```bash
    sudo firewall-cmd --add-port=9455/tcp
    sudo firewall-cmd --runtime-to-permanent
    ```

    Transient transports (WiFi AP, Bluetooth, USB) create their own network and don't require firewall changes.

See [Configuring the Service](docs/configuring-the-service.md) for transport settings, password generators, command allow-lists, and all other options.

## Installing and Using the BoardingPass App

The BoardingPass iOS app discovers nearby devices, authenticates, and walks you through provisioning with a guided wizard.

1. Install the app via [TestFlight](https://testflight.apple.com/) (request an invite from the project maintainer)
2. The app automatically discovers devices via BLE, WiFi, USB, or mDNS
3. Scan the QR code or barcode on the device to authenticate
4. Configure hostname, network, time server, and enrollment
5. Review and apply changes

For app development and releasing to testers, see [mobile/README.md](mobile/README.md).

## Installing and Using the `boarding` CLI

The `boarding` CLI tool provisions devices from the command line.

Install the `boarding` CLI from the latest release:

```sh
# Linux / macOS
curl -L "https://github.com/fzdarsky/boardingpass/releases/latest/download/boarding-cli_$(uname -s)_$(uname -m).tar.gz" | tar xz boarding
sudo install -m 755 boarding /usr/local/bin/
```

```powershell
# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/fzdarsky/boardingpass/releases/latest/download/boarding-cli_Windows_x86_64.zip" -OutFile boarding-cli.zip
Expand-Archive boarding-cli.zip -DestinationPath .; Remove-Item boarding-cli.zip
```

Here are the most important commands:

```bash
# Authenticate (prompts for password; connection is saved)
boarding pass --host 192.168.1.100

# Query device info
boarding info

# Check network interfaces
boarding connections

# Upload configuration files to /etc
boarding load <path_to_etc_dir_content>

# Execute an allow-listed command
boarding command <command_name>

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
