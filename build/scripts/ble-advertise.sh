#!/usr/bin/env bash
# ble-advertise.sh — BLE GATT server for BoardingPass device discovery
#
# Registers a BLE advertisement and GATT service so that the mobile app
# can discover the device and read its connection info (IP, port, cert
# fingerprint) without pairing first.
#
# Uses bluetoothctl (BlueZ 5.x) to register a GATT application via D-Bus.
#
# Usage: ble-advertise.sh <adapter>
#   adapter — Bluetooth adapter name (e.g. hci0)
#
# Environment (set by boardingpass service or read from config):
#   BP_DEVICE_NAME      — BLE advertised name (default: BoardingPass-<hostname>)
#   BP_ADDRESS          — IP address the API is reachable on (default: 10.0.1.1)
#   BP_PORT             — HTTPS port (default: 9455)
#   BP_CERT_FINGERPRINT — TLS certificate SHA-256 fingerprint (optional)

set -euo pipefail

ADAPTER="${1:?Usage: ble-advertise.sh <adapter>}"

DEVICE_NAME="${BP_DEVICE_NAME:-BoardingPass-$(hostname -s)}"
ADDRESS="${BP_ADDRESS:-10.0.1.1}"
PORT="${BP_PORT:-9455}"
CERT_FINGERPRINT="${BP_CERT_FINGERPRINT:-}"

# UUIDs must match the mobile app (mobile/src/services/discovery/bluetooth.ts)
SERVICE_UUID="BBBBBBBB-BBBB-BBBB-BBBB-BBBBBBBBBBBB"
CHAR_DEVICE_NAME="00000001-BBBB-BBBB-BBBB-BBBBBBBBBBBB"
CHAR_IP_ADDRESS="00000002-BBBB-BBBB-BBBB-BBBBBBBBBBBB"
CHAR_PORT="00000003-BBBB-BBBB-BBBB-BBBBBBBBBBBB"
CHAR_CERT_FINGERPRINT="00000004-BBBB-BBBB-BBBB-BBBBBBBBBBBB"

# Convert string to space-separated hex bytes for bluetoothctl
str_to_hex() {
    echo -n "$1" | od -A n -t x1 | tr -d '\n'
}

cleanup() {
    echo "Stopping BLE advertisement on ${ADAPTER}..."
    # Unregister advertisement and GATT application
    bluetoothctl <<-EOF 2>/dev/null || true
menu advertise
off
back
menu gatt
unregister-application
back
EOF
}

trap cleanup EXIT

DEVICE_NAME_HEX=$(str_to_hex "$DEVICE_NAME")
ADDRESS_HEX=$(str_to_hex "$ADDRESS")
PORT_HEX=$(str_to_hex "$PORT")
FINGERPRINT_HEX=$(str_to_hex "$CERT_FINGERPRINT")

# Register GATT application and advertisement via bluetoothctl
bluetoothctl <<EOF
# Select adapter
select ${ADAPTER}

# Register GATT service and characteristics
menu gatt
register-service ${SERVICE_UUID}
register-characteristic ${CHAR_DEVICE_NAME} read
${DEVICE_NAME_HEX}
register-characteristic ${CHAR_IP_ADDRESS} read
${ADDRESS_HEX}
register-characteristic ${CHAR_PORT} read
${PORT_HEX}
register-characteristic ${CHAR_CERT_FINGERPRINT} read
${FINGERPRINT_HEX}
register-application
back

# Configure and start advertisement
menu advertise
uuids ${SERVICE_UUID}
name ${DEVICE_NAME}
on
back
EOF

echo "BLE advertisement active on ${ADAPTER}: ${DEVICE_NAME} (${ADDRESS}:${PORT})"

# Stay alive until stopped by systemd (SIGTERM)
while true; do
    sleep infinity &
    wait $! || break
done
