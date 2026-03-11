/**
 * Device Reachability Probe
 *
 * Probes a device's BoardingPass service to determine reachability.
 * Uses POST /auth/srp/init with an empty body — any HTTP response (even 400)
 * confirms the service is running. Connection-level errors distinguish between
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
 * - 'online': service responded (any HTTP status)
 * - 'unavailable': port is closed (ECONNREFUSED)
 * - 'offline': device unreachable (timeout, host unreachable, etc.)
 */
export async function probeDevice(host: string, port: number): Promise<ProbeResult> {
  const client = createAPIClient(host, port, { timeout: PROBE_TIMEOUT });

  try {
    await client.post('/auth/srp/init', {});
    // Any successful response means the service is running
    return 'online';
  } catch (error) {
    if (error instanceof APIError) {
      // HTTP error responses (e.g. 400) still mean the service is running
      if (error.status) {
        return 'online';
      }

      if (PORT_CLOSED_CODES.has(error.code)) {
        return 'unavailable';
      }

      if (UNREACHABLE_CODES.has(error.code)) {
        return 'offline';
      }

      // ERR_NETWORK often indicates TLS handshake issues — service is likely running
      if (error.code === 'ERR_NETWORK') {
        return 'online';
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
