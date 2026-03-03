/**
 * Complete Service
 *
 * Wrapper for POST /complete endpoint.
 * Signals that provisioning is complete, optionally triggering a device reboot.
 */

import type { components } from '../../types/api';
import type { APIClient } from './client';

export type CompleteRequest = components['schemas']['CompleteRequest'];
export type CompleteResponse = components['schemas']['CompleteResponse'];

/**
 * Signal provisioning completion.
 *
 * @param client - Authenticated API client
 * @param reboot - If true, device will reboot after creating sentinel file.
 *                 If false/omitted, service shuts down gracefully.
 * @returns Completion status with sentinel file path
 */
export async function completeProvisioning(
  client: APIClient,
  reboot = false
): Promise<CompleteResponse> {
  if (!client.getAuthToken()) {
    throw new Error('Authentication required: session token not set');
  }

  const request: CompleteRequest = { reboot };
  return client.post<CompleteResponse>('/complete', request);
}
