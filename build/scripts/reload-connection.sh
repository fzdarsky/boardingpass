#!/bin/bash
# reload-connection.sh - Reload and activate a NetworkManager connection
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

# Reload all connection files from disk
nmcli connection reload

# Activate the specified connection
nmcli connection up "$CONN_NAME" || {
    echo "Error: failed to activate connection '$CONN_NAME'" >&2
    exit 1
}

echo "Connection '$CONN_NAME' reloaded and activated"
