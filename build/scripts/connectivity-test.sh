#!/bin/bash
# connectivity-test.sh - Test network connectivity and output JSON results
# Usage: connectivity-test.sh -- <interface> <gateway>
# Called by BoardingPass via the command allow-list.
set -euo pipefail
[ "${1:-}" = "--" ] && shift

if [ $# -lt 2 ]; then
    echo "Usage: connectivity-test.sh <interface> <gateway>" >&2
    exit 1
fi

IFACE="$1"
GATEWAY="$2"

# Validate interface name
if ! [[ "$IFACE" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo "Error: invalid interface name '$IFACE'" >&2
    exit 1
fi

# Validate gateway (IPv4 or IPv6)
if ! [[ "$GATEWAY" =~ ^[0-9a-fA-F.:]+$ ]]; then
    echo "Error: invalid gateway address '$GATEWAY'" >&2
    exit 1
fi

# 1. Check interface has an IP address
IP_ASSIGNED=false
if ip -j addr show "$IFACE" 2>/dev/null | grep -q '"local"'; then
    IP_ASSIGNED=true
fi

# 2. Ping gateway
GATEWAY_REACHABLE=false
if ping -c 1 -W 5 "$GATEWAY" &>/dev/null; then
    GATEWAY_REACHABLE=true
fi

# 3. DNS resolution
DNS_RESOLVES=false
if getent hosts redhat.com &>/dev/null; then
    DNS_RESOLVES=true
fi

# 4. Internet reachability
INTERNET_REACHABLE=false
HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 http://detectportal.firefox.com/canonical.html 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    INTERNET_REACHABLE=true
fi

printf '{"ip_assigned":%s,"gateway_reachable":%s,"dns_resolves":%s,"internet_reachable":%s}\n' \
    "$IP_ASSIGNED" "$GATEWAY_REACHABLE" "$DNS_RESOLVES" "$INTERNET_REACHABLE"
