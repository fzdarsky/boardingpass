/**
 * Network Config Service
 *
 * Service for retrieving network configuration from the BoardingPass API.
 * Handles GET /network endpoint calls with authentication.
 *
 * Related: T082 - Create network config service
 */

import type { components } from '../../types/api';
import type { APIClient } from './client';

// Type aliases from OpenAPI generated types
export type NetworkConfig = components['schemas']['NetworkConfig'];
export type NetworkInterface = components['schemas']['NetworkInterface'];
export type IPAddress = components['schemas']['IPAddress'];
export type LinkState = NetworkInterface['link_state'];
export type AddressFamily = IPAddress['family'];

/**
 * Fetch network configuration from authenticated device
 *
 * @param client - Configured API client with authentication token
 * @returns Promise resolving to NetworkConfig object
 * @throws Error if request fails or response is invalid
 *
 * @example
 * ```typescript
 * const client = createAPIClient('192.168.1.100', 8443);
 * client.setAuthToken(sessionToken);
 *
 * const networkConfig = await getNetworkConfig(client);
 * networkConfig.interfaces.forEach(iface => {
 *   console.log(`${iface.name}: ${iface.link_state}`);
 * });
 * ```
 */
export async function getNetworkConfig(client: APIClient): Promise<NetworkConfig> {
  // Validate that auth token is set
  if (!client.getAuthToken()) {
    throw new Error('Authentication required: session token not set');
  }

  try {
    // GET /network with authentication header (added by client interceptor)
    const data = await client.get<NetworkConfig>('/network');

    // Validate response structure
    validateNetworkConfig(data);

    return data;
  } catch (error) {
    // Re-throw with additional context
    if (error instanceof Error) {
      throw new Error(`Failed to fetch network configuration: ${error.message}`);
    }
    throw error;
  }
}

/**
 * Validate NetworkConfig response structure
 *
 * Ensures the response from /network endpoint matches expected schema.
 * Throws error if required fields are missing or have invalid types.
 *
 * @param data - Response data to validate
 * @throws Error if validation fails
 */
function validateNetworkConfig(data: unknown): asserts data is NetworkConfig {
  if (!data || typeof data !== 'object') {
    throw new Error('Invalid response: expected object');
  }

  const config = data as Record<string, unknown>;

  // Validate interfaces array
  if (!Array.isArray(config.interfaces)) {
    throw new Error('Invalid response: interfaces must be an array');
  }

  // Validate each interface
  config.interfaces.forEach((iface, index) => {
    if (!iface || typeof iface !== 'object') {
      throw new Error(`Invalid response: interface at index ${index} is not an object`);
    }

    const ifaceObj = iface as Record<string, unknown>;

    // Validate required fields
    if (typeof ifaceObj.name !== 'string') {
      throw new Error(`Invalid response: interface ${index} missing valid name`);
    }

    if (typeof ifaceObj.mac_address !== 'string') {
      throw new Error(`Invalid response: interface ${index} missing valid mac_address`);
    }

    if (ifaceObj.link_state !== 'up' && ifaceObj.link_state !== 'down') {
      throw new Error(`Invalid response: interface ${index} has invalid link_state`);
    }

    if (!Array.isArray(ifaceObj.ip_addresses)) {
      throw new Error(`Invalid response: interface ${index} ip_addresses must be an array`);
    }

    // Validate each IP address
    ifaceObj.ip_addresses.forEach((addr, addrIndex) => {
      if (!addr || typeof addr !== 'object') {
        throw new Error(
          `Invalid response: IP address at index ${addrIndex} of interface ${index} is not an object`
        );
      }

      const addrObj = addr as Record<string, unknown>;

      if (!addrObj.ip) {
        throw new Error(
          `Invalid response: IP address at index ${addrIndex} of interface ${index} missing ip field`
        );
      }

      if (typeof addrObj.prefix !== 'number') {
        throw new Error(
          `Invalid response: IP address at index ${addrIndex} of interface ${index} has invalid prefix`
        );
      }

      if (addrObj.family !== 'ipv4' && addrObj.family !== 'ipv6') {
        throw new Error(
          `Invalid response: IP address at index ${addrIndex} of interface ${index} has invalid family`
        );
      }
    });
  });
}

/**
 * Check if interface is online
 *
 * Helper function to determine if interface is up (T095)
 *
 * @param iface - Network interface object
 * @returns true if link_state is 'up', false otherwise
 */
export function isInterfaceUp(iface: NetworkInterface): boolean {
  return iface.link_state === 'up';
}

/**
 * Check if interface is offline
 *
 * Helper function to determine if interface is down (T095)
 *
 * @param iface - Network interface object
 * @returns true if link_state is 'down', false otherwise
 */
export function isInterfaceDown(iface: NetworkInterface): boolean {
  return iface.link_state === 'down';
}

/**
 * Get interface status color
 *
 * Helper function to get color for status indicator (T095)
 *
 * @param iface - Network interface object
 * @returns Color code for status (green for up, red for down)
 */
export function getInterfaceStatusColor(iface: NetworkInterface): string {
  return iface.link_state === 'up' ? '#4CAF50' : '#F44336'; // Green : Red
}

/**
 * Format MAC address for display
 *
 * Helper function to format MAC address for readability (T089)
 * Converts to uppercase for consistency
 *
 * @param macAddress - MAC address string
 * @returns Formatted MAC address (uppercase with colons)
 */
export function formatMACAddress(macAddress: string): string {
  return macAddress.toUpperCase();
}

/**
 * Format IP address for display
 *
 * Helper function to format IP address with CIDR prefix (T089)
 *
 * @param ipAddress - IP address object
 * @returns Formatted IP address with CIDR notation (e.g., "192.168.1.100/24")
 */
export function formatIPAddress(ipAddress: IPAddress): string {
  return `${ipAddress.ip}/${ipAddress.prefix}`;
}

/**
 * Get IPv4 addresses from interface
 *
 * Helper function to filter only IPv4 addresses
 *
 * @param iface - Network interface object
 * @returns Array of IPv4 addresses
 */
export function getIPv4Addresses(iface: NetworkInterface): IPAddress[] {
  return iface.ip_addresses.filter(addr => addr.family === 'ipv4');
}

/**
 * Get IPv6 addresses from interface
 *
 * Helper function to filter only IPv6 addresses
 *
 * @param iface - Network interface object
 * @returns Array of IPv6 addresses
 */
export function getIPv6Addresses(iface: NetworkInterface): IPAddress[] {
  return iface.ip_addresses.filter(addr => addr.family === 'ipv6');
}

/**
 * Check if interface has any IP addresses
 *
 * Helper function to determine if interface is configured
 *
 * @param iface - Network interface object
 * @returns true if interface has at least one IP address, false otherwise
 */
export function hasIPAddresses(iface: NetworkInterface): boolean {
  return iface.ip_addresses.length > 0;
}

/**
 * Get interface type hint from name
 *
 * Helper function to guess interface type from name pattern (T089)
 *
 * @param name - Interface name
 * @returns Human-readable interface type hint
 */
export function getInterfaceTypeHint(name: string): string {
  if (name === 'lo') {
    return 'Loopback';
  }
  if (name.startsWith('eth') || name.startsWith('en')) {
    return 'Ethernet';
  }
  if (name.startsWith('wlan') || name.startsWith('wl')) {
    return 'Wireless';
  }
  if (name.startsWith('br')) {
    return 'Bridge';
  }
  if (name.startsWith('docker') || name.startsWith('veth')) {
    return 'Virtual';
  }
  return 'Unknown';
}

/**
 * Sort interfaces by priority
 *
 * Helper function to sort interfaces for display (loopback last, physical first)
 *
 * @param interfaces - Array of network interfaces
 * @returns Sorted array (physical interfaces first, loopback last)
 */
export function sortInterfacesByPriority(interfaces: NetworkInterface[]): NetworkInterface[] {
  return [...interfaces].sort((a, b) => {
    // Loopback goes last
    if (a.name === 'lo') return 1;
    if (b.name === 'lo') return -1;

    // Up interfaces before down interfaces
    if (a.link_state === 'up' && b.link_state === 'down') return -1;
    if (a.link_state === 'down' && b.link_state === 'up') return 1;

    // Physical interfaces (eth, wlan) before virtual interfaces
    const aIsPhysical = a.name.startsWith('eth') || a.name.startsWith('wlan');
    const bIsPhysical = b.name.startsWith('eth') || b.name.startsWith('wlan');
    if (aIsPhysical && !bIsPhysical) return -1;
    if (!aIsPhysical && bIsPhysical) return 1;

    // Alphabetical by name
    return a.name.localeCompare(b.name);
  });
}
