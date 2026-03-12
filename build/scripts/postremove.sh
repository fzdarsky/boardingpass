#!/bin/bash
# RPM/DEB postremove script for BoardingPass
# Cleans up directories containing runtime-generated files

set -e

# Determine if this is a complete removal or an upgrade
# RPM: $1 is the number of remaining instances (0 = remove, 1+ = upgrade)
# DEB: $1 is "remove", "purge", "upgrade", etc.
is_removal() {
    case "$1" in
        0|remove|purge)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

if is_removal "$1"; then
    # Remove state directory (contains TLS certs, staging files)
    rm -rf /var/lib/boardingpass

    # Remove config directory (contains config, sentinel, verifier)
    rm -rf /etc/boardingpass
fi

# Reload systemd to pick up removed unit files
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
fi

exit 0
