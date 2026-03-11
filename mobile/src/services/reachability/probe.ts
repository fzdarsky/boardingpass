/**
 * Device Reachability Probe
 *
 * Probes a device's BoardingPass service to determine reachability.
 * Uses GET / which returns {"service": "boardingpass"} for a real BoardingPass
 * service. Validates the response to distinguish BoardingPass from other
 * HTTPS services on the same port. Connection-level errors distinguish between
 * port closed (unavailable) and device unreachable (offline).
 */

import { createAPIClient, APIError } from '../api/client';
import type { Device } from '../../types/device';

export type ProbeResult = 'online' | 'offline' | 'unavailable';

const PROBE_TIMEOUT = 5000; // 5 seconds

/** Error codes indicating the port is closed but device may be on the network. */
const PORT_CLOSED_CODES = new Set(['ECONNREFUSED']);

/** Error codes indicating the device is unreachable at the network level. */
const UNREACHABLE_CODES = new Set([
  'ETIMEDOUT',
  'EHOSTUNREACH',
  'ENETUNREACH',
  'ENOTFOUND',
  'ECONNABORTED',
]);

/**
 * Probe a single device to determine if its BoardingPass service is reachable.
 *
 * Returns:
 * - 'online': BoardingPass service responded with correct identity
 * - 'unavailable': port closed, or a different service is running on the port
 * - 'offline': device unreachable (timeout, host unreachable, etc.)
 */
export async function probeDevice(host: string, port: number): Promise<ProbeResult> {
  const client = createAPIClient(host, port, { timeout: PROBE_TIMEOUT });

  try {
    const data = await client.get<{ service?: string }>('/');
    if (data?.service === 'boardingpass') {
      return 'online';
    }
    // Got a response but it's not BoardingPass
    return 'unavailable';
  } catch (error) {
    if (error instanceof APIError) {
      if (PORT_CLOSED_CODES.has(error.code)) {
        return 'unavailable';
      }

      if (UNREACHABLE_CODES.has(error.code)) {
        return 'offline';
      }

      // HTTP error or TLS error from a responding server that isn't BoardingPass
      if (error.status || error.code === 'ERR_NETWORK') {
        return 'unavailable';
      }
    }

    // Unknown errors default to offline
    return 'offline';
  }
}

/**
 * Probe multiple devices in parallel.
 * Returns a map of device ID to probe result.
 */
export async function probeAllDevices(devices: Device[]): Promise<Map<string, ProbeResult>> {
  const results = new Map<string, ProbeResult>();

  const probes = devices.map(async device => {
    const result = await probeDevice(device.host, device.port);
    results.set(device.id, result);
  });

  await Promise.all(probes);
  return results;
}
