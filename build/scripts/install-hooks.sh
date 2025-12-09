#!/bin/bash
# Installation hooks for BoardingPass RPM/DEB packages
#
# This script provides pre-install, post-install, pre-remove, and post-remove
# hooks for package managers (RPM/DEB).
#
# Usage:
#   RPM spec file:
#     %pre -p /bin/bash
#     %{SOURCE1} pre "$1"
#
#     %post -p /bin/bash
#     %{SOURCE1} post "$1"
#
#   DEB control file:
#     preinst: install-hooks.sh preinst "$@"
#     postinst: install-hooks.sh postinst "$@"

set -e

BOARDINGPASS_USER="boardingpass"
BOARDINGPASS_GROUP="boardingpass"
BOARDINGPASS_UID=990
BOARDINGPASS_GID=990

# Create boardingpass system user and group
create_user() {
    if ! getent group "$BOARDINGPASS_GROUP" >/dev/null 2>&1; then
        echo "Creating boardingpass group..."
        groupadd -r -g "$BOARDINGPASS_GID" "$BOARDINGPASS_GROUP" 2>/dev/null || true
    fi

    if ! getent passwd "$BOARDINGPASS_USER" >/dev/null 2>&1; then
        echo "Creating boardingpass user..."
        useradd -r -u "$BOARDINGPASS_UID" -g "$BOARDINGPASS_GROUP" \
            -d /var/lib/boardingpass \
            -s /sbin/nologin \
            -c "BoardingPass Bootstrap Service" \
            "$BOARDINGPASS_USER" 2>/dev/null || true
    fi
}

# Create required directories
create_directories() {
    echo "Creating directories..."
    mkdir -p /etc/boardingpass
    mkdir -p /var/lib/boardingpass/tls
    mkdir -p /var/lib/boardingpass/staging
    mkdir -p /usr/lib/boardingpass

    # Set ownership
    chown root:root /etc/boardingpass
    chown -R "$BOARDINGPASS_USER:$BOARDINGPASS_GROUP" /var/lib/boardingpass

    # Set permissions
    chmod 0755 /etc/boardingpass
    chmod 0750 /var/lib/boardingpass
    chmod 0700 /var/lib/boardingpass/tls
    chmod 0700 /var/lib/boardingpass/staging
}

# Install default configuration if not exists
install_default_config() {
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
}

# Reload systemd daemon
reload_systemd() {
    if command -v systemctl >/dev/null 2>&1; then
        echo "Reloading systemd..."
        systemctl daemon-reload || true
    fi
}

case "$1" in
    pre|preinst)
        # Pre-installation
        create_user
        ;;

    post|postinst)
        # Post-installation
        create_directories
        install_default_config
        reload_systemd
        ;;

    preun|prerm)
        # Pre-removal
        if command -v systemctl >/dev/null 2>&1; then
            echo "Stopping boardingpass service..."
            systemctl stop boardingpass.service || true
            systemctl disable boardingpass.service || true
        fi
        ;;

    postun|postrm)
        # Post-removal
        reload_systemd

        # Only remove user/group on purge (DEB) or full removal (RPM $1 == 0)
        if [ "$2" = "purge" ] || [ "$2" = "0" ]; then
            echo "Removing boardingpass user and group..."
            userdel "$BOARDINGPASS_USER" 2>/dev/null || true
            groupdel "$BOARDINGPASS_GROUP" 2>/dev/null || true

            echo "Removing directories..."
            rm -rf /var/lib/boardingpass
        fi
        ;;

    *)
        echo "Usage: $0 {pre|post|preun|postun|preinst|postinst|prerm|postrm}" >&2
        exit 1
        ;;
esac

exit 0
