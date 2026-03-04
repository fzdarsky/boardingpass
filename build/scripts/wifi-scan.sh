#!/bin/bash
# wifi-scan.sh - Scan WiFi networks and output JSON
# Usage: wifi-scan.sh
# Called by BoardingPass via the command allow-list.
set -euo pipefail

if ! command -v nmcli &>/dev/null; then
    echo "Error: nmcli not found" >&2
    exit 1
fi

# Scan and output as colon-delimited fields
RAW=$(nmcli -t -f DEVICE,SSID,BSSID,SIGNAL,SECURITY,CHAN,FREQ,RATE device wifi list --rescan yes 2>/dev/null) || {
    echo "Error: WiFi scan failed" >&2
    exit 1
}

# Transform to JSON array
echo "["
FIRST=true
while IFS=: read -r DEVICE SSID BSSID_1 BSSID_2 BSSID_3 BSSID_4 BSSID_5 BSSID_6 SIGNAL SECURITY CHAN FREQ RATE; do
    # Skip empty lines
    [ -z "$DEVICE" ] && continue

    # Reassemble BSSID — nmcli -t escapes colons as \: so each split part
    # carries a trailing backslash (e.g. "AA\"). Strip them to get "AA:BB:...".
    BSSID="${BSSID_1%\\}:${BSSID_2%\\}:${BSSID_3%\\}:${BSSID_4%\\}:${BSSID_5%\\}:${BSSID_6}"

    # Normalize security field
    if [ -z "$SECURITY" ] || [ "$SECURITY" = "--" ]; then
        SECURITY="open"
    fi

    # Strip units from numeric fields (nmcli -t still includes "MHz", "Mbit/s")
    FREQ="${FREQ%% *}"
    CHAN="${CHAN%% *}"

    if [ "$FIRST" = true ]; then
        FIRST=false
    else
        echo ","
    fi

    # Escape SSID for JSON (handle special characters)
    SSID_ESCAPED=$(printf '%s' "$SSID" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g')

    printf '  {"device":"%s","ssid":"%s","bssid":"%s","signal":%d,"security":"%s","channel":%d,"frequency":%d,"rate":"%s"}' \
        "$DEVICE" "$SSID_ESCAPED" "$BSSID" "$SIGNAL" "$SECURITY" "$CHAN" "$FREQ" "$RATE"
done <<< "$RAW"
echo ""
echo "]"
