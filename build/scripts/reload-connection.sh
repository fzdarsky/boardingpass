#!/bin/bash
# reload-connection.sh - Reload and optionally activate a NetworkManager connection
# Usage: reload-connection.sh -- <connection-name> [provisioning-interface]
#
# When provisioning-interface is provided (immediate mode):
#   1. Reloads connection files from disk
#   2. Protects the provisioning interface's default route with metric 1
#   3. Activates the connection (nmcli connection up)
#
# When only connection-name is provided (deferred mode):
#   1. Reloads connection files from disk (activation deferred to reboot)
#
# Called by BoardingPass via the command allow-list.
set -euo pipefail
[ "${1:-}" = "--" ] && shift

if [ $# -lt 1 ]; then
    echo "Usage: reload-connection.sh <connection-name> [provisioning-interface]" >&2
    exit 1
fi

CONN_NAME="$1"
PROV_IFACE="${2:-}"

# Validate connection name (alphanumeric + hyphens + underscores + dots)
if ! [[ "$CONN_NAME" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo "Error: invalid connection name '$CONN_NAME'" >&2
    exit 1
fi

# Validate provisioning interface name if provided
if [ -n "$PROV_IFACE" ] && ! [[ "$PROV_IFACE" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo "Error: invalid interface name '$PROV_IFACE'" >&2
    exit 1
fi

# Reload all connection files from disk so NetworkManager picks up the new profile.
nmcli connection reload

if [ -n "$PROV_IFACE" ]; then
    # Immediate mode: protect provisioning interface route, then activate.
    #
    # Activating a connection with a gateway adds a default route (e.g. metric 100
    # for ethernet). To prevent this from hijacking traffic away from the
    # provisioning interface, we temporarily add a high-priority (metric 1) default
    # route via the provisioning interface. This route is ephemeral and disappears
    # on reboot.
    PROV_GW=$(ip route show default dev "$PROV_IFACE" 2>/dev/null | awk '/default/ {print $3; exit}' || true)
    if [ -n "$PROV_GW" ]; then
        ip route replace default via "$PROV_GW" dev "$PROV_IFACE" metric 1 2>/dev/null || true
        echo "Protected provisioning route via $PROV_IFACE (metric 1)"
    fi

    nmcli connection up "$CONN_NAME"
    echo "Connection '$CONN_NAME' reloaded and activated"
else
    # Deferred mode: reload only. The profile has autoconnect=true, so
    # NetworkManager will activate it when the carrier is detected or on reboot.
    echo "Connection '$CONN_NAME' reloaded"
fi
