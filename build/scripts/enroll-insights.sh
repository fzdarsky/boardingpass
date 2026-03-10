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

# Verify required tools are available
for cmd in jq rhc; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "Error: '$cmd' is not installed" >&2
        exit 1
    fi
done

# Read credentials from staging file
ORG_ID=$(jq -r '.org_id' "$STAGING_FILE") || {
    echo "Error: failed to parse org_id from $STAGING_FILE" >&2
    rm -f "$STAGING_FILE"
    exit 1
}

ACTIVATION_KEY=$(jq -r '.activation_key' "$STAGING_FILE") || {
    echo "Error: failed to parse activation_key from $STAGING_FILE" >&2
    rm -f "$STAGING_FILE"
    exit 1
}

DISABLE_MGMT=$(jq -r '.disable_remote_management // false' "$STAGING_FILE")

# Delete staging file before executing enrollment (minimize credential lifetime)
rm -f "$STAGING_FILE"

# Build command — disable remote management when Flight Control handles it
RHC_ARGS=(connect --organization "$ORG_ID" --activation-key "$ACTIVATION_KEY")
if [ "$DISABLE_MGMT" = "true" ]; then
    if rhc connect --help 2>&1 | grep -q -- '--disable-feature'; then
        RHC_ARGS+=(--disable-feature remote-management)
    else
        echo "Error: disable_remote_management requested but rhc does not support --disable-feature" >&2
        echo "Hint: --disable-feature requires rhc 0.3+ (RHEL 10+)" >&2
        exit 1
    fi
fi

rhc "${RHC_ARGS[@]}" || {
    echo "Error: rhc connect failed" >&2
    exit 1
}

echo "Successfully enrolled with Red Hat Insights"
