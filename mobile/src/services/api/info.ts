/**
 * Device Info Service
 *
 * Service for retrieving device system information from the BoardingPass API.
 * Handles GET /info endpoint calls with authentication.
 *
 * Related: T081 - Create device info service
 */

import type { components } from '../../types/api';
import type { APIClient } from './client';

// Type aliases from OpenAPI generated types
export type SystemInfo = components['schemas']['SystemInfo'];
export type TPMInfo = components['schemas']['TPMInfo'];
export type BoardInfo = components['schemas']['BoardInfo'];
export type CPUInfo = components['schemas']['CPUInfo'];
export type OSInfo = components['schemas']['OSInfo'];

/**
 * Fetch system information from authenticated device
 *
 * @param client - Configured API client with authentication token
 * @returns Promise resolving to SystemInfo object
 * @throws Error if request fails or response is invalid
 *
 * @example
 * ```typescript
 * const client = createAPIClient('192.168.1.100', 8443);
 * client.setAuthToken(sessionToken);
 *
 * const systemInfo = await getSystemInfo(client);
 * console.log('FIPS Enabled:', systemInfo.os.fips_enabled);
 * console.log('Architecture:', systemInfo.cpu.architecture);
 * ```
 */
export async function getSystemInfo(client: APIClient): Promise<SystemInfo> {
  // Validate that auth token is set
  if (!client.getAuthToken()) {
    throw new Error('Authentication required: session token not set');
  }

  try {
    // GET /info with authentication header (added by client interceptor)
    const data = await client.get<SystemInfo>('/info');

    // Validate response structure
    validateSystemInfo(data);

    return data;
  } catch (error) {
    // Re-throw with additional context
    if (error instanceof Error) {
      throw new Error(`Failed to fetch system information: ${error.message}`);
    }
    throw error;
  }
}

/**
 * Validate SystemInfo response structure
 *
 * Ensures the response from /info endpoint matches expected schema.
 * Throws error if required fields are missing or have invalid types.
 *
 * @param data - Response data to validate
 * @throws Error if validation fails
 */
function validateSystemInfo(data: unknown): asserts data is SystemInfo {
  if (!data || typeof data !== 'object') {
    throw new Error('Invalid response: expected object');
  }

  const info = data as Record<string, unknown>;

  // Validate required top-level fields
  if (!info.tpm || typeof info.tpm !== 'object') {
    throw new Error('Invalid response: missing or invalid tpm field');
  }

  if (!info.board || typeof info.board !== 'object') {
    throw new Error('Invalid response: missing or invalid board field');
  }

  if (!info.cpu || typeof info.cpu !== 'object') {
    throw new Error('Invalid response: missing or invalid cpu field');
  }

  if (!info.os || typeof info.os !== 'object') {
    throw new Error('Invalid response: missing or invalid os field');
  }

  // Validate TPM structure
  const tpm = info.tpm as Record<string, unknown>;
  if (typeof tpm.present !== 'boolean') {
    throw new Error('Invalid response: TPM present field must be boolean');
  }

  // Validate Board structure (all required)
  const board = info.board as Record<string, unknown>;
  if (typeof board.manufacturer !== 'string') {
    throw new Error('Invalid response: Board manufacturer must be string');
  }
  if (typeof board.model !== 'string') {
    throw new Error('Invalid response: Board model must be string');
  }
  if (typeof board.serial !== 'string') {
    throw new Error('Invalid response: Board serial must be string');
  }

  // Validate CPU structure
  const cpu = info.cpu as Record<string, unknown>;
  if (typeof cpu.architecture !== 'string') {
    throw new Error('Invalid response: CPU architecture must be string');
  }
  const validArchitectures = ['x86_64', 'aarch64', 'armv7l'];
  if (!validArchitectures.includes(cpu.architecture)) {
    throw new Error(
      `Invalid response: CPU architecture must be one of ${validArchitectures.join(', ')}`
    );
  }

  // Validate OS structure (all required)
  const os = info.os as Record<string, unknown>;
  if (typeof os.distribution !== 'string') {
    throw new Error('Invalid response: OS distribution must be string');
  }
  if (typeof os.version !== 'string') {
    throw new Error('Invalid response: OS version must be string');
  }
  if (typeof os.fips_enabled !== 'boolean') {
    throw new Error('Invalid response: OS fips_enabled field must be boolean');
  }
}

/**
 * Extract FIPS status from system info
 *
 * Helper function to get FIPS mode boolean for UI display (T094)
 *
 * @param systemInfo - System information object
 * @returns true if FIPS mode is enabled, false otherwise
 */
export function isFIPSEnabled(systemInfo: SystemInfo): boolean {
  return systemInfo.os.fips_enabled === true;
}

/**
 * Check if device has TPM
 *
 * Helper function to determine if device has TPM hardware
 *
 * @param systemInfo - System information object
 * @returns true if TPM is present, false otherwise
 */
export function hasTPM(systemInfo: SystemInfo): boolean {
  return systemInfo.tpm.present === true;
}

/**
 * Get TPM version string
 *
 * Helper function to extract TPM version for display
 *
 * @param systemInfo - System information object
 * @returns TPM version string or null if not present
 */
export function getTPMVersion(systemInfo: SystemInfo): string | null {
  if (!systemInfo.tpm.present) {
    return null;
  }
  return systemInfo.tpm.version || null;
}

/**
 * Get human-readable architecture name
 *
 * Helper function to convert architecture enum to display string (T089)
 *
 * @param architecture - CPU architecture enum value
 * @returns Human-readable architecture name
 */
export function getArchitectureDisplayName(architecture: CPUInfo['architecture']): string {
  const displayNames: Record<CPUInfo['architecture'], string> = {
    x86_64: 'Intel/AMD 64-bit (x86_64)',
    aarch64: 'ARM 64-bit (aarch64)',
    armv7l: 'ARM 32-bit (armv7l)',
  };

  return displayNames[architecture] || architecture;
}
