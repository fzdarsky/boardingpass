#!/bin/bash
# show-status.sh - Display system status summary
# Usage: show-status.sh
# Called by BoardingPass via the command allow-list.
set -euo pipefail

echo "=== System Status ==="
echo "Hostname: $(hostname)"
echo "Uptime: $(uptime -p 2>/dev/null || uptime)"
echo ""

echo "=== Network ==="
if command -v nmcli &>/dev/null; then
    nmcli -t -f DEVICE,STATE,CONNECTION device status 2>/dev/null || echo "(NetworkManager not available)"
else
    ip -brief addr show 2>/dev/null || echo "(ip command not available)"
fi
echo ""

echo "=== NTP ==="
if command -v chronyc &>/dev/null; then
    chronyc tracking 2>/dev/null | head -5 || echo "(chronyd not running)"
else
    echo "(chrony not installed)"
fi
