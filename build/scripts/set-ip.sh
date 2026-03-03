#!/bin/bash
# set-ip.sh - Configure a static IPv4 address on a network interface
# Usage: set-ip.sh -- <interface> <ip/prefix> <gateway> [dns]
# Called by BoardingPass via the command allow-list.
set -euo pipefail

if [ $# -lt 3 ]; then
    echo "Usage: set-ip.sh <interface> <ip/prefix> <gateway> [dns]" >&2
    exit 1
fi

IFACE="$1"
IP_PREFIX="$2"
GATEWAY="$3"
DNS="${4:-}"

# Validate interface name (alphanumeric + dots for VLANs)
if ! [[ "$IFACE" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo "Error: invalid interface name '$IFACE'" >&2
    exit 1
fi

# Check if a NetworkManager connection exists for this interface
CONN_NAME=$(nmcli -t -f NAME,DEVICE connection show --active 2>/dev/null | grep ":${IFACE}$" | cut -d: -f1 | head -1)

if [ -z "$CONN_NAME" ]; then
    # No active connection; try any connection for this device
    CONN_NAME=$(nmcli -t -f NAME,DEVICE connection show 2>/dev/null | grep ":${IFACE}$" | cut -d: -f1 | head -1)
fi

if [ -z "$CONN_NAME" ]; then
    echo "Error: no NetworkManager connection found for interface '$IFACE'" >&2
    exit 1
fi

# Apply static IP configuration
nmcli connection modify "$CONN_NAME" \
    ipv4.method manual \
    ipv4.addresses "$IP_PREFIX" \
    ipv4.gateway "$GATEWAY"

if [ -n "$DNS" ]; then
    nmcli connection modify "$CONN_NAME" ipv4.dns "$DNS"
fi

# Reactivate connection to apply changes
nmcli connection up "$CONN_NAME"

echo "Static IP $IP_PREFIX configured on $IFACE (gateway: $GATEWAY)"
