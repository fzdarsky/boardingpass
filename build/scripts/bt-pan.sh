#!/usr/bin/env bash
# bt-pan.sh — Bluetooth PAN (NAP) server using BlueZ D-Bus API
#
# Replaces bluez-tools' bt-network command, which is not available in RHEL.
# Uses busctl (part of systemd) to register a Network Access Point profile
# on the specified Bluetooth adapter via org.bluez.NetworkServer1.
#
# Usage: bt-pan.sh <adapter> <bridge> <address>
#   adapter  — Bluetooth adapter name (e.g. hci0)
#   bridge   — Bridge interface name for BNEP connections (e.g. br-bp0)
#   address  — IP address with prefix length to assign to bridge (e.g. 10.0.1.1/24)

set -euo pipefail

ADAPTER="${1:?Usage: bt-pan.sh <adapter> <bridge> <address>}"
BRIDGE="${2:?Usage: bt-pan.sh <adapter> <bridge> <address>}"
ADDRESS="${3:?Usage: bt-pan.sh <adapter> <bridge> <address>}"
DBUS_PATH="/org/bluez/${ADAPTER}"

# Set adapter alias to BoardingPass device name (read from env file)
DEVICE_NAME="${BP_DEVICE_NAME:-BoardingPass-$(hostname -s)}"
bluetoothctl system-alias "${DEVICE_NAME}" 2>/dev/null || true

cleanup() {
    echo "Stopping Bluetooth PAN on ${ADAPTER}..."
    # Kill the agent process
    if [[ -n "${AGENT_PID:-}" ]]; then
        kill "${AGENT_PID}" 2>/dev/null || true
        wait "${AGENT_PID}" 2>/dev/null || true
    fi
    busctl call org.bluez "${DBUS_PATH}" org.bluez.NetworkServer1 Unregister s nap 2>/dev/null || true
    if ip link show "${BRIDGE}" &>/dev/null; then
        ip link set "${BRIDGE}" down 2>/dev/null || true
        ip link delete "${BRIDGE}" type bridge 2>/dev/null || true
    fi
}

trap cleanup EXIT

# Register a NoInputNoOutput agent for headless "Just Works" pairing.
# The agent must stay running to handle pairing requests, so we keep
# bluetoothctl alive by holding stdin open via the while-sleep loop.
{
    echo "agent NoInputNoOutput"
    echo "default-agent"
    while true; do sleep infinity; done
} | bluetoothctl &>/dev/null &
AGENT_PID=$!

# Create bridge interface and assign IP
if ! ip link show "${BRIDGE}" &>/dev/null; then
    ip link add name "${BRIDGE}" type bridge
fi
ip link set "${BRIDGE}" up
ip addr add "${ADDRESS}" dev "${BRIDGE}" 2>/dev/null || true

# Register NAP profile via BlueZ D-Bus API
# This tells BlueZ to bridge incoming Bluetooth PAN (BNEP) connections to the bridge.
busctl call org.bluez "${DBUS_PATH}" org.bluez.NetworkServer1 Register ss nap "${BRIDGE}"
echo "NAP registered on ${ADAPTER}, bridging to ${BRIDGE} (${ADDRESS})"

# Stay alive until stopped by systemd (SIGTERM)
while true; do
    sleep infinity &
    wait $! || break
done
