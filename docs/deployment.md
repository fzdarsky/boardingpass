# BoardingPass Deployment Guide

**Version**: 0.1.0
**Last Updated**: 2025-12-09

## Overview

BoardingPass is designed to be simple to deploy. Install the package, configure authentication, and start the service.

---

## Quick Start

### Install Package

**RHEL-based systems (RHEL 9+, Rocky, AlmaLinux)**:
```bash
sudo dnf install https://github.com/fzdarsky/boardingpass/releases/download/v0.1.0/boardingpass-0.1.0-1.x86_64.rpm
```

**Debian-based systems (Debian, Ubuntu)**:
```bash
wget https://github.com/fzdarsky/boardingpass/releases/download/v0.1.0/boardingpass_0.1.0_amd64.deb
sudo apt install ./boardingpass_0.1.0_amd64.deb
```

### Configure Authentication

Use one of the pre-packaged password generators or create a custom one.

**Option 1: Use a pre-packaged generator** (recommended):

```bash
# Generate a unique salt
SALT=$(openssl rand -base64 32)

cat <<EOF | sudo tee /etc/boardingpass/verifier
{
  "username": "boardingpass",
  "salt": "${SALT}",
  "password_generator": "/usr/lib/boardingpass/generators/board_serial"
}
EOF

sudo chmod 400 /etc/boardingpass/verifier
```

**Option 2: Create a custom generator**:

```bash
# Create custom generator in generators directory
cat <<'EOF' | sudo tee /usr/lib/boardingpass/generators/custom
#!/bin/bash
# Output device-unique password from your custom logic
dmidecode -s system-serial-number | tr -d '[:space:]'
EOF

sudo chmod 755 /usr/lib/boardingpass/generators/custom

# Reference it in verifier config
SALT=$(openssl rand -base64 32)

cat <<EOF | sudo tee /etc/boardingpass/verifier
{
  "username": "boardingpass",
  "salt": "${SALT}",
  "password_generator": "/usr/lib/boardingpass/generators/custom"
}
EOF

sudo chmod 400 /etc/boardingpass/verifier
```

### Start the Service

```bash
sudo systemctl enable --now boardingpass.service
```

That's it! The service is now running and ready for provisioning.

---

## Configuration

The main configuration file is `/etc/boardingpass/config.yaml`. Here's a minimal example:

```yaml
service:
  inactivity_timeout: "10m"
  session_ttl: "30m"
  sentinel_file: "/etc/boardingpass/issued"

transports:
  ethernet:
    enabled: true
    port: 8443

provisioning:
  allowed_paths:
    - /etc/systemd/system/
    - /etc/NetworkManager/system-connections/

commands:
  - id: "reboot"
    path: "/usr/bin/systemctl"
    args: ["reboot"]
  - id: "restart-networkmanager"
    path: "/usr/bin/systemctl"
    args: ["restart", "NetworkManager"]

logging:
  level: "info"
  format: "json"
```

### Configuration Options

**service**:
- `inactivity_timeout`: How long to wait before shutting down due to inactivity (e.g., "10m", "30m")
- `session_ttl`: How long session tokens remain valid (e.g., "30m", "1h")
- `sentinel_file`: Path to the sentinel file that prevents the service from running after provisioning

**transports.ethernet**:
- `enabled`: Whether to enable Ethernet transport (true/false)
- `port`: HTTPS port to listen on (default: 8443)
- `address`: IP address to bind to (leave empty for all interfaces)
- `tls_cert`: Path to TLS certificate (auto-generated if missing)
- `tls_key`: Path to TLS private key (auto-generated if missing)

**provisioning.allowed_paths**:
- List of directories where configuration files can be written
- Only files under these paths will be accepted

**commands**:
- List of allowed commands that can be executed via the API
- Each command has an `id`, `path`, and `args`

**logging**:
- `level`: Log level (debug, info, warn, error)
- `format`: Log format (json, human)

---

## Password Generation

BoardingPass uses device-unique passwords generated from hardware identifiers.

### Pre-Packaged Password Generators

The BoardingPass RPM/DEB packages include three pre-packaged password generators in `/usr/lib/boardingpass/generators/`:

**1. `board_serial`** - Board serial number from DMI (recommended):
- Uses: `/sys/class/dmi/id/board_serial`
- Best for enterprise hardware with unique serial numbers
- Most secure option (hardware-bound, unchangeable)

**2. `tpm_ek`** - TPM 2.0 endorsement key hash:
- Requires: `tpm2-tools` package
- Best for devices with TPM 2.0
- Hardware-bound and cannot be changed

**3. `primary_mac`** - Primary network interface MAC address:
- Uses: Primary ethernet interface MAC
- Fallback when DMI/TPM unavailable
- Less secure (MAC addresses can be changed)

To use a pre-packaged generator, reference it in your verifier configuration:

```bash
cat <<EOF | sudo tee /etc/boardingpass/verifier
{
  "username": "boardingpass",
  "salt": "$(openssl rand -base64 32)",
  "password_generator": "/usr/lib/boardingpass/generators/board_serial"
}
EOF
```

### Custom Password Generators

You can create custom password generators in `/usr/lib/boardingpass/generators/`:

```bash
sudo tee /usr/lib/boardingpass/generators/product_uuid <<'EOF'
#!/bin/bash
cat /sys/class/dmi/id/product_uuid
EOF
sudo chmod 755 /usr/lib/boardingpass/generators/product_uuid
```

Then reference it in your verifier configuration:
```bash
"password_generator": "/usr/lib/boardingpass/generators/product_uuid"
```

The password should be printed on the device label during manufacturing for the bootstrap operator.

---

## Provisioning Workflow

1. **Discover device**: Find the device IP address on your network
2. **Authenticate**: Use the device password (from the label or password generator script) to perform SRP-6a authentication
3. **Query device info**: GET /info and GET /network to verify device identity and connectivity
4. **Provision configuration**: POST /configure with a configuration bundle (JSON)
5. **Execute commands**: POST /command to restart services or reboot
6. **Complete provisioning**: POST /complete to create the sentinel file and shut down the service

After provisioning is complete, the service will not start again (sentinel file prevents it).

---

## Bootc Integration

For bootc-based immutable systems, include BoardingPass in your Containerfile:

```dockerfile
FROM registry.redhat.io/rhel9/rhel-bootc:9.7

# Copy and install BoardingPass RPM
COPY boardingpass_*_linux_amd64.rpm /tmp/boardingpass.rpm
RUN dnf install -y /tmp/boardingpass.rpm && \
    dnf clean all && \
    rm -f /tmp/boardingpass.rpm

# Configure authentication using pre-packaged generator
RUN cat > /etc/boardingpass/verifier <<EOF
{
  "username": "boardingpass",
  "salt": "$(echo -n "your-salt-here" | base64)",
  "password_generator": "/usr/lib/boardingpass/generators/board_serial"
}
EOF

# (Optional) Or create a custom generator
RUN cat > /usr/lib/boardingpass/generators/custom <<'EOF'
#!/bin/bash
cat /sys/class/dmi/id/product_uuid
EOF
RUN chmod 755 /usr/lib/boardingpass/generators/custom

# Configure service (optional - RPM includes default config)
COPY config.yaml /etc/boardingpass/config.yaml

# Enable service
RUN systemctl enable boardingpass.service
```

---

## Firewall Configuration

BoardingPass listens on port 8443 by default. Open this port in your firewall:

**firewalld** (RHEL, Rocky, AlmaLinux):
```bash
sudo firewall-cmd --permanent --add-port=8443/tcp
sudo firewall-cmd --reload
```

**ufw** (Debian, Ubuntu):
```bash
sudo ufw allow 8443/tcp
```

---

## Verification

Check if the service is running:

```bash
sudo systemctl status boardingpass.service
```

Test the API:

```bash
curl -k https://<device-ip>:8443/info
# Should return: {"error":"unauthorized",...}
```

View logs:

```bash
sudo journalctl -u boardingpass.service -f
```

---

## Troubleshooting

### Service won't start after provisioning

This is expected. The sentinel file (`/etc/boardingpass/issued`) prevents the service from starting on provisioned devices.

To re-provision a device (testing only):
```bash
sudo rm /etc/boardingpass/issued
sudo systemctl restart boardingpass.service
```

### Authentication fails

Check that the password generator script works:
```bash
sudo /usr/lib/boardingpass/password-generator
```

Verify the verifier configuration exists:
```bash
sudo cat /etc/boardingpass/verifier
```

### Cannot connect to the service

1. Check if the service is running: `sudo systemctl status boardingpass.service`
2. Check if the port is open: `sudo ss -tlnp | grep 8443`
3. Check the firewall: Ensure port 8443 is allowed
4. Check the logs: `sudo journalctl -u boardingpass.service`

### Configuration provisioning fails

Check the `allowed_paths` in `/etc/boardingpass/config.yaml`. Files can only be written to directories in this list.

---

## Uninstallation

**RHEL-based systems**:
```bash
sudo dnf remove boardingpass
```

**Debian-based systems**:
```bash
sudo apt remove boardingpass
```

---

## References

- **Development Guide**: [development.md](development.md)
- **API Documentation**: [api.md](api.md)
- **Security Guide**: [security.md](security.md)
- **OpenAPI Specification**: [../specs/001-boardingpass-api/contracts/openapi.yaml](../specs/001-boardingpass-api/contracts/openapi.yaml)

---

**Document Status**: Complete
**Last Updated**: 2025-12-09
