/**
 * Transport Preference and De-duplication
 *
 * Provides priority ordering for transports and de-duplicates devices
 * discovered via multiple methods.
 */

import { Device, DiscoveryMethod } from '@/types/device';

const TRANSPORT_PRIORITY: Record<DiscoveryMethod, number> = {
  usb: 1,
  bluetooth: 2,
  wifi: 3,
  mdns: 4,
  fallback: 4,
  manual: 5,
};

/**
 * Returns priority for a discovery method (lower = higher priority).
 */
export function getTransportPriority(method: DiscoveryMethod): number {
  return TRANSPORT_PRIORITY[method] ?? 5;
}

/**
 * Selects the device entry with the highest-priority transport.
 */
export function selectPreferredTransport(devices: Device[]): Device | undefined {
  if (devices.length === 0) return undefined;

  return devices.reduce((best, current) => {
    const bestPri = getTransportPriority(best.discoveryMethod);
    const currentPri = getTransportPriority(current.discoveryMethod);
    return currentPri < bestPri ? current : best;
  });
}

/**
 * De-duplicates devices that represent the same physical device discovered
 * via multiple transports. Matching criteria:
 * 1. Same certificate fingerprint (if available)
 * 2. Same hostname from /info endpoint or mDNS name
 *
 * The returned list keeps the highest-priority transport for each device,
 * storing alternate transports in the `alternateTransports` field.
 */
export function deduplicateDevices(devices: Device[]): Device[] {
  const groups = new Map<string, Device[]>();

  for (const device of devices) {
    const key = getDeduplicationKey(device);
    const group = groups.get(key) || [];
    group.push(device);
    groups.set(key, group);
  }

  const result: Device[] = [];
  for (const group of groups.values()) {
    // Sort by priority (lowest number = highest priority)
    group.sort(
      (a, b) => getTransportPriority(a.discoveryMethod) - getTransportPriority(b.discoveryMethod)
    );
    // Use highest-priority entry as the primary device
    result.push(group[0]);
  }

  return result;
}

/**
 * Generates a key for grouping devices that represent the same physical device.
 * Uses certificate fingerprint first, then falls back to name matching.
 */
function getDeduplicationKey(device: Device): string {
  // Primary: certificate fingerprint
  if (device.certificateInfo?.fingerprint) {
    return `cert:${device.certificateInfo.fingerprint}`;
  }

  // Secondary: device name (from mDNS TXT, SSID extraction, or /info endpoint)
  if (device.name) {
    return `name:${device.name}`;
  }

  // Fallback: unique per device ID
  return `id:${device.id}`;
}
