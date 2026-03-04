#!/bin/bash
# enroll-insights.sh - Enroll device with Red Hat Insights via rhc connect
# Usage: enroll-insights.sh
# Reads credentials from staging file, runs enrollment, then deletes the file.
# Called by BoardingPass via the command allow-list.
set -euo pipefail

STAGING_FILE="/etc/boardingpass/staging/insights.json"

# Exit cleanly if staging file does not exist (not an error)
if [ ! -f "$STAGING_FILE" ]; then
    echo "No Insights staging file found, skipping"
    exit 0
fi

# Read credentials from staging file
ORG_ID=$(cat "$STAGING_FILE" | python3 -c "import sys,json; print(json.load(sys.stdin)['org_id'])" 2>/dev/null) || {
    echo "Error: failed to parse org_id from $STAGING_FILE" >&2
    rm -f "$STAGING_FILE"
    exit 1
}

ACTIVATION_KEY=$(cat "$STAGING_FILE" | python3 -c "import sys,json; print(json.load(sys.stdin)['activation_key'])" 2>/dev/null) || {
    echo "Error: failed to parse activation_key from $STAGING_FILE" >&2
    rm -f "$STAGING_FILE"
    exit 1
}

# Delete staging file before executing enrollment (minimize credential lifetime)
rm -f "$STAGING_FILE"

# Run enrollment
rhc connect --organization "$ORG_ID" --activation-key "$ACTIVATION_KEY" || {
    echo "Error: rhc connect failed" >&2
    exit 1
}

echo "Successfully enrolled with Red Hat Insights"
