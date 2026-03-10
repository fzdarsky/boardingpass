# CLI Reference

The `boarding` CLI tool provides a command-line interface for provisioning BoardingPass devices. It handles SRP-6a authentication, TLS certificate validation, and session management.

## Installation

### From Binary Release

Download and install the latest release (auto-detects OS and architecture):

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

### From Source

Requires Go 1.25+:

```bash
make build-cli
sudo cp _output/bin/boarding /usr/local/bin/
```

### Verify Installation

```bash
boarding --help
```

## Configuration

The CLI supports three configuration methods with clear precedence: **Flags > Environment Variables > Config File**.

### Command-Line Flags

```bash
boarding pass --host 192.168.1.100 --port 8443 --username admin
```

### Environment Variables

```bash
export BOARDING_HOST=192.168.1.100
export BOARDING_PORT=8443
export BOARDING_CA_CERT=/path/to/ca.pem
```

### Config File

The config file at `~/.config/boardingpass/config.yaml` is automatically created and updated after successful authentication:

```yaml
host: 192.168.1.100
port: 8443
ca_cert: /path/to/ca.pem  # optional
```

After the first `boarding pass --host ...`, subsequent commands remember the connection.

## Global Flags

- `-y, --assumeyes` — Automatically answer 'yes' to prompts (e.g., TLS certificate acceptance)

## Commands

### `boarding pass` — Authenticate

Authenticate with a BoardingPass device using SRP-6a.

```bash
boarding pass --host <host> [--port <port>] [--username <user>] [--password <pass>]
```

| Flag | Env Var | Default | Description |
| ---- | ------- | ------- | ----------- |
| `--host` | `BOARDING_HOST` | — | Hostname or IP address |
| `--port` | `BOARDING_PORT` | 8443 | Service port |
| `--username` | — | (prompts) | Username |
| `--password` | — | (prompts) | Password |
| `--ca-cert` | `BOARDING_CA_CERT` | — | Custom CA certificate bundle |

```bash
# Interactive
boarding pass --host 192.168.1.100

# Non-interactive (for scripts)
boarding pass -y --host 192.168.1.100 --username admin --password secret123

# With custom CA certificate
boarding pass --host internal.corp --ca-cert /etc/ssl/ca.pem --username admin
```

### `boarding info` — Query System Information

Retrieve hardware and software information from the device.

```bash
boarding info [--output yaml|json]
```

### `boarding connections` — Query Network Interfaces

List network interfaces and their configuration.

```bash
boarding connections [--output yaml|json]
```

### `boarding load` — Upload Configuration

Upload configuration files from a local directory to the device. Files are uploaded recursively, maintaining directory structure.

```bash
boarding load <directory>
```

Limits: maximum 100 files, 10 MB total size.

### `boarding command` — Execute Command

Execute an allow-listed command on the device. Command output (stdout/stderr) is displayed and the exit code is preserved.

```bash
boarding command <command-id>
```

### `boarding complete` — Complete Provisioning

Signal provisioning completion and terminate the session. The service finalizes provisioning and shuts down. The local session token is deleted.

```bash
boarding complete
```

## CI/CD Pipeline Example

```bash
#!/bin/bash
set -euo pipefail

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
```

## Session Management

The CLI automatically manages session tokens:

- **Storage:** `~/.cache/boardingpass/session-<host>-<port>.token`
- **Permissions:** 0600 (owner read/write only)
- **Auto-loading:** Tokens are loaded automatically for subsequent commands
- **Cleanup:** Tokens are deleted on `boarding complete`
- **Multiple devices:** Separate tokens for each host:port combination

```bash
# View active sessions
ls -la ~/.cache/boardingpass/

# Clear all sessions manually
rm -f ~/.cache/boardingpass/session-*.token
```

## TLS Certificate Handling

The CLI uses **Trust-On-First-Use (TOFU)** for self-signed certificates:

```bash
# First connection prompts for certificate acceptance
boarding pass --host 192.168.1.100 --username admin
# WARNING: Unknown TLS certificate fingerprint
#   Host: 192.168.1.100:8443
#   Fingerprint: SHA256:a1b2c3d4...
# Do you want to accept this certificate? (yes/no): yes
```

The fingerprint is saved to `~/.config/boardingpass/known_certs.yaml`. Future connections validate against the stored fingerprint.

For production environments, use a custom CA certificate instead of TOFU:

```bash
boarding pass --host internal.corp --ca-cert /etc/ssl/ca-bundle.pem --username admin
```

## Security Best Practices

1. **Don't save passwords in config files** — pass via flag or let it prompt
2. **Use environment variables in CI/CD** for connection parameters
3. **Verify certificate fingerprints** on first connection in production
4. **Use custom CA certificates** when possible (avoid TOFU in production)
5. **Run `boarding complete`** after provisioning to clean up sessions

## Troubleshooting

**"not authenticated" or "no active session"**
Session expired (30-minute TTL) or no token found. Re-authenticate with `boarding pass`.

**"connection refused"**
Service not running or unreachable. Check `systemctl status boardingpass`, network connectivity, and firewall (port 8443).

**"certificate fingerprint mismatch"**
Server certificate changed. If expected, remove the entry from `~/.config/boardingpass/known_certs.yaml` and re-connect. If unexpected, investigate.

**"command not permitted" or "not in allow-list"**
Command ID not in the server's allow-list. Check server configuration or contact the administrator.
