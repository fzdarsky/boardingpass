#!/bin/bash
# set-hostname.sh - Set the system hostname persistently
# Usage: set-hostname.sh -- <hostname>
# Called by BoardingPass via the command allow-list.
set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: set-hostname.sh <hostname>" >&2
    exit 1
fi

HOSTNAME="$1"

# Validate hostname (RFC 1123: alphanumeric + hyphens, max 253 chars)
if ! [[ "$HOSTNAME" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$ ]]; then
    echo "Error: invalid hostname '$HOSTNAME'" >&2
    exit 1
fi

if [ ${#HOSTNAME} -gt 253 ]; then
    echo "Error: hostname exceeds 253 characters" >&2
    exit 1
fi

hostnamectl set-hostname "$HOSTNAME"
echo "Hostname set to: $HOSTNAME"
