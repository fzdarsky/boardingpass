#!/bin/bash
# RPM/DEB preinstall script for BoardingPass
# Creates the boardingpass user and group

set -e

BOARDINGPASS_USER="boardingpass"
BOARDINGPASS_GROUP="boardingpass"

# Create boardingpass system user and group
# Let the system assign UID/GID to avoid conflicts
if ! getent group "$BOARDINGPASS_GROUP" >/dev/null 2>&1; then
    echo "Creating boardingpass group..."
    groupadd -r "$BOARDINGPASS_GROUP"
fi

if ! getent passwd "$BOARDINGPASS_USER" >/dev/null 2>&1; then
    echo "Creating boardingpass user..."
    useradd -r -g "$BOARDINGPASS_GROUP" \
        -d /var/lib/boardingpass \
        -s /sbin/nologin \
        -c "BoardingPass Bootstrap Service" \
        "$BOARDINGPASS_USER"
fi

exit 0
