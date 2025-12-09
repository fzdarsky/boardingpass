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

Create a password generator script at `/usr/lib/boardingpass/password-generator`:

```bash
cat <<'EOF' | sudo tee /usr/lib/boardingpass/password-generator
#!/bin/bash
# Output device-unique password from board serial number
dmidecode -s system-serial-number | tr -d '[:space:]'
EOF

sudo chmod 500 /usr/lib/boardingpass/password-generator
```

Create the SRP verifier configuration at `/etc/boardingpass/verifier`:

```bash
# Generate a unique salt
SALT=$(openssl rand -base64 32)

cat <<EOF | sudo tee /etc/boardingpass/verifier
{
  "username": "boardingpass",
  "salt": "${SALT}",
  "password_generator": "/usr/lib/boardingpass/password-generator"
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

BoardingPass uses device-unique passwords generated from hardware identifiers. The password generator script at `/usr/lib/boardingpass/password-generator` outputs the device password.

### Common Password Sources

**Board serial number** (most common):
```bash
#!/bin/bash
dmidecode -s system-serial-number | tr -d '[:space:]'
```

**Network MAC address**:
```bash
#!/bin/bash
cat /sys/class/net/eth0/address | tr -d ':'
```

**Combined sources** (stronger):
```bash
#!/bin/bash
echo "$(dmidecode -s system-serial-number | tr -d '[:space:]')-$(cat /sys/class/net/eth0/address | tr -d ':')"
```

**TPM endorsement key** (requires tpm2-tools):
```bash
#!/bin/bash
tpm2_getcap handles-endorsement | grep -A1 "persistent-handle" | tail -1 | tr -d '[:space:]'
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
FROM registry.redhat.io/rhel9/rhel-bootc:9.4

# Install BoardingPass
ADD https://github.com/fzdarsky/boardingpass/releases/download/v0.1.0/boardingpass-0.1.0-1.x86_64.rpm /tmp/
RUN rpm -ivh /tmp/boardingpass-0.1.0-1.x86_64.rpm && rm /tmp/boardingpass-0.1.0-1.x86_64.rpm

# Configure authentication
COPY verifier.json /etc/boardingpass/verifier
COPY password-generator.sh /usr/lib/boardingpass/password-generator
RUN chmod 400 /etc/boardingpass/verifier && chmod 500 /usr/lib/boardingpass/password-generator

# Configure service
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
