/**
 * Configure Service
 *
 * Wrapper for POST /configure endpoint.
 * Sends a file bundle to be atomically provisioned on the device.
 */

import type { components } from '../../types/api';
import type { APIClient } from './client';

export type ConfigBundle = components['schemas']['ConfigBundle'];
export type ConfigFile = components['schemas']['ConfigFile'];

export interface ConfigureResult {
  status: 'success';
  message?: string;
}

/**
 * Send a configuration bundle to the device.
 *
 * Files are written atomically — if any file fails, the entire bundle is rejected.
 * File content must be base64-encoded.
 */
export async function sendConfigBundle(
  client: APIClient,
  files: ConfigFile[]
): Promise<ConfigureResult> {
  if (!client.getAuthToken()) {
    throw new Error('Authentication required: session token not set');
  }

  const bundle: ConfigBundle = { files };
  return client.post<ConfigureResult>('/configure', bundle);
}

/**
 * Helper to create a ConfigFile entry from a path and string content.
 * Encodes content to base64 automatically.
 */
export function createConfigFile(path: string, content: string, mode = 0o644): ConfigFile {
  // btoa is available in React Native's JavaScript engine
  const encoded = btoa(content);
  return { path, content: encoded, mode };
}
