#!/bin/bash
# enroll-flightctl.sh - Enroll device with Flight Control
# Usage: enroll-flightctl.sh
# Reads credentials from staging file, logs in, requests enrollment certificate,
# installs agent config, and restarts the flightctl-agent service.
# Called by BoardingPass via the command allow-list.
set -euo pipefail

STAGING_FILE="/etc/boardingpass/staging/flightctl.json"
AGENT_CONFIG="/etc/flightctl/config.yaml"

# Exit cleanly if staging file does not exist (not an error)
if [ ! -f "$STAGING_FILE" ]; then
    echo "No Flight Control staging file found, skipping"
    exit 0
fi

# Verify required tools are available
for cmd in jq flightctl systemctl; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "Error: '$cmd' is not installed" >&2
        exit 1
    fi
done

# Verify flightctl-agent is installed (enrollment only makes sense with the agent)
if ! systemctl list-unit-files flightctl-agent.service | grep -q flightctl-agent; then
    echo "Error: flightctl-agent service is not installed" >&2
    echo "Hint: install the flightctl-agent package before enrolling" >&2
    exit 1
fi

# Read credentials from staging file
ENDPOINT=$(jq -r '.endpoint' "$STAGING_FILE") || {
    echo "Error: failed to parse endpoint from $STAGING_FILE" >&2
    rm -f "$STAGING_FILE"
    exit 1
}

TOKEN=$(jq -r '.token // empty' "$STAGING_FILE")
USERNAME=$(jq -r '.username // empty' "$STAGING_FILE")
PASSWORD=$(jq -r '.password // empty' "$STAGING_FILE")

# Validate that we have at least one auth method
if [ -z "$TOKEN" ] && { [ -z "$USERNAME" ] || [ -z "$PASSWORD" ]; }; then
    echo "Error: staging file must contain either 'token' or both 'username' and 'password'" >&2
    rm -f "$STAGING_FILE"
    exit 1
fi

# Delete staging file before executing enrollment (minimize credential lifetime)
rm -f "$STAGING_FILE"

# Create isolated temp directory for flightctl client config
TMPDIR=$(mktemp -d)
chmod 700 "$TMPDIR"
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

# Step 1: Login to Flight Control API
if [ -n "$TOKEN" ]; then
    flightctl login "$ENDPOINT" --token "$TOKEN" --config-dir "$TMPDIR" -k || {
        echo "Error: flightctl login failed (token auth)" >&2
        exit 1
    }
else
    # Note: -p exposes password in /proc/PID/cmdline; unavoidable with current
    # flightctl CLI. The isolated temp config dir limits exposure window.
    flightctl login "$ENDPOINT" -u "$USERNAME" -p "$PASSWORD" --config-dir "$TMPDIR" -k || {
        echo "Error: flightctl login failed (password auth)" >&2
        exit 1
    }
fi

# Step 2: Request enrollment certificate
flightctl certificate request \
    --signer=enrollment \
    --expiration=365d \
    --output=embedded \
    --config-dir "$TMPDIR" \
    -d "$TMPDIR" > "$TMPDIR/config.yaml" || {
    echo "Error: flightctl certificate request failed" >&2
    exit 1
}

# Step 3: Install agent config
mkdir -p "$(dirname "$AGENT_CONFIG")"
install -m 0600 "$TMPDIR/config.yaml" "$AGENT_CONFIG" || {
    echo "Error: failed to install $AGENT_CONFIG" >&2
    exit 1
}

# Step 4: Restart flightctl-agent to pick up new config
systemctl restart flightctl-agent || {
    echo "Error: failed to restart flightctl-agent" >&2
    exit 1
}

echo "Successfully enrolled with Flight Control"
