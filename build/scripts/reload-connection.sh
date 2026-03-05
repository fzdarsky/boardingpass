#!/bin/bash
# reload-connection.sh - Reload a NetworkManager connection profile from disk
# Usage: reload-connection.sh -- <connection-name>
# Called by BoardingPass via the command allow-list.
set -euo pipefail
[ "${1:-}" = "--" ] && shift

if [ $# -lt 1 ]; then
    echo "Usage: reload-connection.sh <connection-name>" >&2
    exit 1
fi

CONN_NAME="$1"

# Validate connection name (alphanumeric + hyphens + underscores)
if ! [[ "$CONN_NAME" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo "Error: invalid connection name '$CONN_NAME'" >&2
    exit 1
fi

# Reload all connection files from disk so NetworkManager picks up the new profile.
# We intentionally do NOT force-activate (nmcli connection up) because during
# provisioning we may be communicating over a different interface. Force-activating
# a connection with a gateway would add a competing default route whose lower metric
# (e.g. 100 for ethernet vs 600 for WiFi) could hijack traffic away from the
# provisioning interface, breaking the session. The profile has autoconnect=true,
# so NetworkManager will activate it when the carrier is detected or on reboot.
nmcli connection reload

echo "Connection '$CONN_NAME' reloaded"
