#!/bin/bash
# set-dns.sh - Configure DNS servers on a network interface
# Usage: set-dns.sh -- <interface> <dns1> [dns2]
# Called by BoardingPass via the command allow-list.
set -euo pipefail
[ "${1:-}" = "--" ] && shift

if [ $# -lt 2 ]; then
    echo "Usage: set-dns.sh <interface> <dns1> [dns2]" >&2
    exit 1
fi

IFACE="$1"
shift
DNS_SERVERS="$*"

# Validate interface name
if ! [[ "$IFACE" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo "Error: invalid interface name '$IFACE'" >&2
    exit 1
fi

# Find NetworkManager connection for interface
CONN_NAME=$(nmcli -t -f NAME,DEVICE connection show 2>/dev/null | grep ":${IFACE}$" | cut -d: -f1 | head -1)

if [ -z "$CONN_NAME" ]; then
    echo "Error: no NetworkManager connection found for interface '$IFACE'" >&2
    exit 1
fi

# Convert space-separated DNS to comma-separated for nmcli
DNS_CSV=$(echo "$DNS_SERVERS" | tr ' ' ',')

nmcli connection modify "$CONN_NAME" ipv4.dns "$DNS_CSV"
nmcli connection up "$CONN_NAME"

echo "DNS servers set to: $DNS_SERVERS on $IFACE"
