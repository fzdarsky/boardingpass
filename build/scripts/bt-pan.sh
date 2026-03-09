#!/usr/bin/env bash
# bt-pan.sh — Bluetooth PAN (NAP) server using BlueZ D-Bus API
#
# Replaces bluez-tools' bt-network command, which is not available in RHEL.
# Uses busctl (part of systemd) to register a Network Access Point profile
# on the specified Bluetooth adapter via org.bluez.NetworkServer1.
#
# Usage: bt-pan.sh <adapter> <bridge>
#   adapter  — Bluetooth adapter name (e.g. hci0)
#   bridge   — Bridge interface name for BNEP connections (e.g. br-bp0)

set -euo pipefail

ADAPTER="${1:?Usage: bt-pan.sh <adapter> <bridge>}"
BRIDGE="${2:?Usage: bt-pan.sh <adapter> <bridge>}"
DBUS_PATH="/org/bluez/${ADAPTER}"

cleanup() {
    echo "Unregistering NAP on ${ADAPTER}..."
    busctl call org.bluez "${DBUS_PATH}" org.bluez.NetworkServer1 Unregister s nap 2>/dev/null || true
    if ip link show "${BRIDGE}" &>/dev/null; then
        ip link set "${BRIDGE}" down 2>/dev/null || true
        ip link delete "${BRIDGE}" type bridge 2>/dev/null || true
    fi
}

trap cleanup EXIT

# Create bridge interface
if ! ip link show "${BRIDGE}" &>/dev/null; then
    ip link add name "${BRIDGE}" type bridge
fi
ip link set "${BRIDGE}" up

# Register NAP profile via BlueZ D-Bus API
# This tells BlueZ to bridge incoming Bluetooth PAN (BNEP) connections to the bridge.
busctl call org.bluez "${DBUS_PATH}" org.bluez.NetworkServer1 Register ss nap "${BRIDGE}"
echo "NAP registered on ${ADAPTER}, bridging to ${BRIDGE}"

# Stay alive until stopped by systemd (SIGTERM)
while true; do
    sleep infinity &
    wait $! || break
done
