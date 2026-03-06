#!/bin/bash
# connectivity-test.sh - Test network connectivity and output JSON results
# Usage: connectivity-test.sh -- <interface> [gateway]
# If gateway is omitted, the default gateway from the routing table is used.
# Called by BoardingPass via the command allow-list.
set -euo pipefail
[ "${1:-}" = "--" ] && shift

if [ $# -lt 1 ]; then
    echo "Usage: connectivity-test.sh <interface> [gateway]" >&2
    exit 1
fi

IFACE="$1"
GATEWAY="${2:-}"

# Validate interface name
if ! [[ "$IFACE" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo "Error: invalid interface name '$IFACE'" >&2
    exit 1
fi

# Auto-detect gateway from routing table if not provided.
# The "|| true" guards prevent set -e + pipefail from killing the script
# when ip-route finds no matching default route.
if [ -z "$GATEWAY" ]; then
    GATEWAY=$(ip route show default dev "$IFACE" 2>/dev/null | awk '/default/ {print $3; exit}' || true)
    # Fallback: try any default route regardless of interface
    if [ -z "$GATEWAY" ]; then
        GATEWAY=$(ip route show default 2>/dev/null | awk '/default/ {print $3; exit}' || true)
    fi
fi

# Validate gateway if we have one (IPv4 or IPv6)
if [ -n "$GATEWAY" ] && ! [[ "$GATEWAY" =~ ^[0-9a-fA-F.:]+$ ]]; then
    echo "Error: invalid gateway address '$GATEWAY'" >&2
    exit 1
fi

# 1. Check link/carrier (is a cable plugged in?)
LINK_UP=false
if [ "$(cat /sys/class/net/"$IFACE"/carrier 2>/dev/null || echo 0)" = "1" ]; then
    LINK_UP=true
fi

# 2. Check interface has an IP address
IP_ASSIGNED=false
if ip -j addr show "$IFACE" 2>/dev/null | grep -q '"local"'; then
    IP_ASSIGNED=true
fi

# 3. Ping gateway via the provisioning interface (skip if no gateway found)
GATEWAY_REACHABLE=false
if [ -n "$GATEWAY" ]; then
    if ping -c 1 -W 5 -I "$IFACE" "$GATEWAY" &>/dev/null; then
        GATEWAY_REACHABLE=true
    fi
fi

# 4. DNS resolution
# getent has no interface binding — this tests system-wide resolution.
DNS_RESOLVES=false
if getent hosts redhat.com &>/dev/null; then
    DNS_RESOLVES=true
fi

# 5. Internet reachability via the provisioning interface
INTERNET_REACHABLE=false
HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' --interface "$IFACE" --max-time 5 http://detectportal.firefox.com/canonical.html 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    INTERNET_REACHABLE=true
fi

printf '{"link_up":%s,"ip_assigned":%s,"gateway_reachable":%s,"dns_resolves":%s,"internet_reachable":%s}\n' \
    "$LINK_UP" "$IP_ASSIGNED" "$GATEWAY_REACHABLE" "$DNS_RESOLVES" "$INTERNET_REACHABLE"
