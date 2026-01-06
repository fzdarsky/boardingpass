#!/bin/bash
# RPM/DEB postinstall script for BoardingPass
# Creates directories, installs default config, and reloads systemd

set -e

BOARDINGPASS_USER="boardingpass"
BOARDINGPASS_GROUP="boardingpass"

# Create required directories
echo "Creating directories..."
mkdir -p /etc/boardingpass
mkdir -p /var/lib/boardingpass/tls
mkdir -p /var/lib/boardingpass/staging
mkdir -p /usr/lib/boardingpass

# Set ownership
chown root:root /etc/boardingpass
# Only set ownership if user exists (might not be fully available in some contexts)
if getent passwd "$BOARDINGPASS_USER" >/dev/null 2>&1; then
    chown -R "$BOARDINGPASS_USER:$BOARDINGPASS_GROUP" /var/lib/boardingpass
else
    echo "Warning: $BOARDINGPASS_USER user not yet available, skipping ownership change"
fi

# Set permissions
chmod 0755 /etc/boardingpass
chmod 0750 /var/lib/boardingpass
chmod 0700 /var/lib/boardingpass/tls
chmod 0700 /var/lib/boardingpass/staging

# Install default configuration if not exists
if [ ! -f /etc/boardingpass/config.yaml ]; then
    echo "Installing default configuration..."
    cat > /etc/boardingpass/config.yaml <<'EOF'
service:
  inactivity_timeout: "10m"
  session_ttl: "30m"
  sentinel_file: "/etc/boardingpass/issued"

transports:
  ethernet:
    enabled: true
    interfaces: []
    address: ""
    port: 8443
    tls_cert: "/var/lib/boardingpass/tls/server.crt"
    tls_key: "/var/lib/boardingpass/tls/server.key"

commands:
  - id: "reboot"
    path: "/usr/sbin/systemctl"
    args: ["reboot", "--force"]
  - id: "restart-networkmanager"
    path: "/usr/bin/systemctl"
    args: ["restart", "NetworkManager"]
  - id: "restart-chronyd"
    path: "/usr/bin/systemctl"
    args: ["restart", "chronyd"]

logging:
  level: "info"
  format: "json"

paths:
  allow_list:
    - "/etc/systemd/"
    - "/etc/NetworkManager/"
EOF
    chown root:root /etc/boardingpass/config.yaml
    chmod 0644 /etc/boardingpass/config.yaml
fi

# Reload systemd daemon
if command -v systemctl >/dev/null 2>&1; then
    echo "Reloading systemd..."
    systemctl daemon-reload || true
fi

exit 0
