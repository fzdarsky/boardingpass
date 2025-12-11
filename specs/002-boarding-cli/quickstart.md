# Quickstart: Boarding CLI Tool

**Feature**: 002-boarding-cli
**Date**: 2025-12-10
**Purpose**: Quick start guide for installing and using the `boarding` CLI tool

## Overview

The `boarding` CLI tool enables developers and CI systems to interact with BoardingPass services for device provisioning. It provides commands for authentication, system inspection, configuration upload, and command execution.

## Installation

### From Binary Release (Recommended)

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -LO https://github.com/your-org/boardingpass/releases/latest/download/boarding-cli_linux_amd64.tar.gz
tar -xzf boarding-cli_linux_amd64.tar.gz
sudo mv boarding /usr/local/bin/

# macOS (arm64/Apple Silicon)
curl -LO https://github.com/your-org/boardingpass/releases/latest/download/boarding-cli_darwin_arm64.tar.gz
tar -xzf boarding-cli_darwin_arm64.tar.gz
sudo mv boarding /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/your-org/boardingpass/releases/latest/download/boarding-cli_windows_amd64.zip" -OutFile "boarding-cli.zip"
Expand-Archive -Path boarding-cli.zip -DestinationPath .
Move-Item boarding.exe C:\Windows\System32\
```

### From Source

Requires Go 1.25+ installed:

```bash
git clone https://github.com/your-org/boardingpass.git
cd boardingpass
make build-cli
sudo cp _output/bin/boarding /usr/local/bin/
```

### Verify Installation

```bash
boarding --help
```

## Configuration

The CLI supports three configuration methods (in order of precedence):

###1. Command-Line Flags (Highest Priority)

```bash
boarding pass --host 192.168.1.100 --port 8443 --username admin
```

### 2. Environment Variables (Medium Priority)

```bash
export BOARDING_HOST=192.168.1.100
export BOARDING_PORT=8443
boarding pass --username admin
```

### 3. Config File (Lowest Priority)

Create `~/.config/boardingpass/config.yaml` (Linux/Unix) or `%APPDATA%\boardingpass\config.yaml` (Windows):

```yaml
host: 192.168.1.100
port: 8443
```

Then run commands without flags:

```bash
boarding pass --username admin
```

## Quick Start Examples

### Example 1: Basic Provisioning Workflow

```bash
# Step 1: Authenticate
boarding pass --host 192.168.1.100 --username admin
# Prompts for password, stores session token

# Step 2: Query system information
boarding info
# Displays CPU, board, TPM, OS, FIPS status in YAML

# Step 3: Check network interfaces
boarding connections
# Displays network interface details in YAML

# Step 4: Upload configuration
boarding load /path/to/config-directory
# Uploads all files in directory to device

# Step 5: Execute command
boarding command "systemctl restart networking"
# Executes command on device, shows output

# Step 6: Complete provisioning
boarding complete
# Triggers device to finalize and logout
```

### Example 2: CI/CD Pipeline (Non-Interactive)

```bash
#!/bin/bash
set -euo pipefail

# Use environment variables for connection
export BOARDING_HOST=${DEVICE_IP}
export BOARDING_PORT=8443

# Authenticate with flags (no interactive prompts)
boarding pass --username "${DEVICE_USER}" --password "${DEVICE_PASSWORD}"

# Query device info and save as artifact
boarding info -o json > device-info.json

# Upload configuration
boarding load ./device-configs/edge-node/

# Run post-provision script
boarding command "/opt/bootstrap/post-provision.sh"

# Complete provisioning
boarding complete

echo "Provisioning completed successfully"
```

### Example 3: Custom CA Certificate

If your BoardingPass service uses certificates signed by a custom CA:

```bash
# Option 1: Via flag
boarding pass --host internal.example.com --ca-cert /path/to/ca-bundle.pem --username admin

# Option 2: Via config file
cat > ~/.config/boardingpass/config.yaml <<EOF
host: internal.example.com
port: 8443
ca_cert: /etc/ssl/certs/company-ca.pem
EOF

boarding pass --username admin
```

### Example 4: Self-Signed Certificates (TOFU)

For self-signed certificates, the CLI uses Trust-on-First-Use:

```bash
boarding pass --host 192.168.1.100 --username admin
```

Output:
```
WARNING: Unknown TLS certificate fingerprint
  Host: 192.168.1.100:8443
  Fingerprint: SHA256:a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6

Do you want to accept this certificate? (yes/no): yes

Certificate accepted. Authentication proceeding...
```

The fingerprint is saved to `~/.config/boardingpass/known_certs.yaml` and future connections will not prompt.

## Command Reference

### `boarding pass` - Authenticate

Authenticate with BoardingPass service using SRP-6a protocol.

**Usage**:
```bash
boarding pass [flags]
```

**Flags**:
- `--username` - Username (required, or will prompt)
- `--password` - Password (optional, prompts if not provided)
- `--host` - BoardingPass service hostname/IP
- `--port` - BoardingPass service port (default: 8443)
- `--ca-cert` - Path to custom CA certificate bundle

**Examples**:
```bash
# Interactive (prompts for username and password)
boarding pass

# With username flag (prompts for password)
boarding pass --username admin

# Fully non-interactive
boarding pass --username admin --password secret123

# With custom CA
boarding pass --host internal.corp --ca-cert /etc/ssl/ca.pem --username admin
```

**Output**:
```
Password: ********
Authenticating with 192.168.1.100:8443...
Authentication successful. Session valid for 30 minutes.
```

---

### `boarding info` - Query System Information

Query device system information (CPU, board, TPM, OS, FIPS status).

**Usage**:
```bash
boarding info [flags]
```

**Flags**:
- `-o, --output` - Output format: `yaml` (default) or `json`

**Examples**:
```bash
# YAML output (default)
boarding info

# JSON output
boarding info -o json

# Pipe to jq
boarding info -o json | jq '.cpu.model'
```

**Output (YAML)**:
```yaml
cpu:
  model: Intel(R) Xeon(R) CPU E5-2686 v4 @ 2.30GHz
  cores: 4
  architecture: x86_64
board:
  vendor: Amazon EC2
  model: c5.xlarge
tpm:
  version: "2.0"
  manufacturer: "0x1014"
os:
  distribution: Red Hat Enterprise Linux
  version: "9.3"
  kernel: 5.14.0-362.8.1.el9_3.x86_64
fips:
  enabled: true
  mode: enforcing
```

---

### `boarding connections` - Query Network Interfaces

Query device network interface configuration.

**Usage**:
```bash
boarding connections [flags]
```

**Flags**:
- `-o, --output` - Output format: `yaml` (default) or `json`

**Examples**:
```bash
# YAML output (default)
boarding connections

# JSON output
boarding connections -o json
```

**Output (YAML)**:
```yaml
interfaces:
  - name: eth0
    mac: 00:0a:95:9d:68:16
    state: up
    addresses:
      - ip: 192.168.1.100
        prefix: 24
        family: ipv4
  - name: eth1
    mac: 00:0a:95:9d:68:17
    state: down
    addresses: []
```

---

### `boarding load` - Upload Configuration

Upload configuration files from a directory to the device.

**Usage**:
```bash
boarding load <directory> [flags]
```

**Arguments**:
- `<directory>` - Path to configuration directory

**Examples**:
```bash
# Upload all files in directory
boarding load /path/to/config-directory

# Upload current directory
boarding load .
```

**Output**:
```
Uploading configuration from /path/to/config-directory...
Uploading: 47/100 files (8.2 MB / 10.0 MB) [========>    ] 82%
Upload complete. Configuration applied successfully.
```

**Constraints**:
- Maximum 100 files per upload
- Maximum 10 MB total size
- Files are validated against server-side allow-list

---

### `boarding command` - Execute Command

Execute an allow-listed command on the device.

**Usage**:
```bash
boarding command "<command-string>" [flags]
```

**Arguments**:
- `<command-string>` - Command to execute (quote if contains spaces)

**Examples**:
```bash
# Restart networking
boarding command "systemctl restart networking"

# Check service status
boarding command "systemctl status sshd"

# Run custom script
boarding command "/opt/provision/post-install.sh"
```

**Output**:
```
Executing: systemctl restart networking
---
(command output appears here)
---
Command completed with exit code: 0
```

**Note**: Command must be in server-side allow-list, or you'll receive:
```
Error: Command not permitted by server allow-list
```

---

### `boarding complete` - Complete Provisioning

Terminate session and signal BoardingPass service that provisioning is complete.

**Usage**:
```bash
boarding complete
```

**Examples**:
```bash
# Complete provisioning
boarding complete
```

**Output**:
```
Provisioning complete. Session terminated.
BoardingPass service will create sentinel file and prepare for shutdown.
```

**Effect**:
- Calls `/complete` endpoint on server
- Deletes local session token
- Server creates sentinel file (`/var/lib/boardingpass/provisioning.complete`)
- Server prepares for graceful shutdown

---

## Troubleshooting

### Error: "Not authenticated"

**Cause**: No valid session token found.

**Solution**:
```bash
boarding pass --username admin
```

### Error: "Session expired"

**Cause**: Session token TTL (30 minutes) has expired.

**Solution**:
```bash
boarding pass --username admin
```

### Error: "Connection refused"

**Cause**: BoardingPass service is not running or not reachable.

**Solution**:
- Verify service is running: `systemctl status boardingpass`
- Check network connectivity: `ping <host>`
- Verify firewall rules allow port 8443

### Error: "Certificate rejected by user"

**Cause**: You declined the TLS certificate prompt.

**Solution**:
- Re-run command and accept certificate, OR
- Use `--ca-cert` flag with proper CA bundle

### Error: "Certificate fingerprint mismatch"

**Cause**: Server certificate changed since first connection (possible MITM attack or legitimate cert rotation).

**Solution**:
- If cert rotation is expected, edit `~/.config/boardingpass/known_certs.yaml` and remove old entry
- Re-run command to accept new certificate
- If unexpected, investigate for potential security incident

### Error: "Command not permitted"

**Cause**: Command is not in server-side allow-list.

**Solution**:
- Check server logs for allowed commands
- Contact administrator to add command to allow-list
- Use `load` command to provision scripts instead

## Advanced Usage

### Multiple BoardingPass Services

The CLI supports sessions to multiple services simultaneously (different tokens per host:port):

```bash
# Provision device 1
boarding pass --host 192.168.1.100 --username admin
boarding info

# Provision device 2 (different session)
boarding pass --host 192.168.1.101 --username admin
boarding info

# Sessions are independent
```

### Session Management

View current sessions:
```bash
ls -la ~/.cache/boardingpass/
# Shows session-*.token files
```

Clear all sessions:
```bash
rm -f ~/.cache/boardingpass/session-*.token
```

Clear single session:
```bash
boarding complete  # Clears current session
```

### Output Processing

```bash
# Extract specific fields with jq
boarding info -o json | jq '.cpu.cores'

# Convert YAML to JSON
boarding info | yq eval -o=json

# Save to file
boarding info > device-$(date +%Y%m%d).yaml
```

## Security Best Practices

1. **Use config file for persistent settings, flags for sensitive values**:
   ```bash
   # config.yaml has host/port
   # Pass username/password via flags (not saved)
   boarding pass --username admin --password "${PASSWORD}"
   ```

2. **In CI/CD, use environment variables**:
   ```bash
   export BOARDING_HOST=${DEVICE_IP}
   boarding pass --username admin --password "${DEVICE_SECRET}"
   ```

3. **Verify certificate fingerprints** on first connection (write down expected fingerprint before deployment)

4. **Use custom CA certificates** for production environments (avoid TOFU in production)

5. **Rotate session tokens** by running `boarding complete` after provisioning

## Getting Help

```bash
# General help
boarding --help

# Command-specific help
boarding pass --help
boarding info --help
boarding load --help
```

## Next Steps

- See [spec.md](./spec.md) for detailed feature specification
- See [plan.md](./plan.md) for implementation architecture
- Report issues: https://github.com/your-org/boardingpass/issues
