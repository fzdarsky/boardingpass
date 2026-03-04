#!/bin/bash
# enroll-flightctl.sh - Enroll device with Flight Control via flightctl login
# Usage: enroll-flightctl.sh
# Reads credentials from staging file, runs enrollment, then deletes the file.
# Called by BoardingPass via the command allow-list.
set -euo pipefail

STAGING_FILE="/etc/boardingpass/staging/flightctl.json"

# Exit cleanly if staging file does not exist (not an error)
if [ ! -f "$STAGING_FILE" ]; then
    echo "No Flight Control staging file found, skipping"
    exit 0
fi

# Read credentials from staging file
ENDPOINT=$(cat "$STAGING_FILE" | python3 -c "import sys,json; print(json.load(sys.stdin)['endpoint'])" 2>/dev/null) || {
    echo "Error: failed to parse endpoint from $STAGING_FILE" >&2
    rm -f "$STAGING_FILE"
    exit 1
}

USERNAME=$(cat "$STAGING_FILE" | python3 -c "import sys,json; print(json.load(sys.stdin)['username'])" 2>/dev/null) || {
    echo "Error: failed to parse username from $STAGING_FILE" >&2
    rm -f "$STAGING_FILE"
    exit 1
}

PASSWORD=$(cat "$STAGING_FILE" | python3 -c "import sys,json; print(json.load(sys.stdin)['password'])" 2>/dev/null) || {
    echo "Error: failed to parse password from $STAGING_FILE" >&2
    rm -f "$STAGING_FILE"
    exit 1
}

# Delete staging file before executing enrollment (minimize credential lifetime)
rm -f "$STAGING_FILE"

# Run enrollment
flightctl login "$ENDPOINT" --username "$USERNAME" --password "$PASSWORD" || {
    echo "Error: flightctl login failed" >&2
    exit 1
}

echo "Successfully enrolled with Flight Control"
