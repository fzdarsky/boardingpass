#!/bin/bash
# set-ntp.sh - Configure an NTP server in chrony
# Usage: set-ntp.sh -- <ntp-server>
# Called by BoardingPass via the command allow-list.
set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: set-ntp.sh <ntp-server>" >&2
    exit 1
fi

NTP_SERVER="$1"

# Validate NTP server (hostname or IP)
if ! [[ "$NTP_SERVER" =~ ^[a-zA-Z0-9._:-]+$ ]]; then
    echo "Error: invalid NTP server '$NTP_SERVER'" >&2
    exit 1
fi

CHRONY_CONF="/etc/chrony.conf"
CHRONY_SOURCES_DIR="/etc/chrony.d"

# Prefer chrony.d/ drop-in directory if available
if [ -d "$CHRONY_SOURCES_DIR" ]; then
    echo "server $NTP_SERVER iburst" > "$CHRONY_SOURCES_DIR/boardingpass.conf"
    echo "NTP source written to $CHRONY_SOURCES_DIR/boardingpass.conf"
elif [ -f "$CHRONY_CONF" ]; then
    # Append to main config if no drop-in directory
    echo "server $NTP_SERVER iburst" >> "$CHRONY_CONF"
    echo "NTP source appended to $CHRONY_CONF"
else
    echo "Error: chrony configuration not found" >&2
    exit 1
fi

# Restart chronyd and force sync
systemctl restart chronyd
chronyc makestep

echo "NTP server set to: $NTP_SERVER"
