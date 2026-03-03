/**
 * Command Service
 *
 * Wrapper for POST /command endpoint.
 * Executes allow-listed commands on the device with optional parameters.
 */

import type { components } from '../../types/api';
import type { APIClient } from './client';

export type CommandRequest = components['schemas']['CommandRequest'];
export type CommandResponse = components['schemas']['CommandResponse'];

/**
 * Execute an allow-listed command on the device.
 *
 * @param client - Authenticated API client
 * @param id - Command identifier from the allow-list
 * @param params - Optional positional parameters (appended after --)
 * @returns Command execution result with exit_code, stdout, stderr
 */
export async function executeCommand(
  client: APIClient,
  id: string,
  params?: string[]
): Promise<CommandResponse> {
  if (!client.getAuthToken()) {
    throw new Error('Authentication required: session token not set');
  }

  const request: CommandRequest = { id };
  if (params && params.length > 0) {
    request.params = params;
  }

  return client.post<CommandResponse>('/command', request);
}
