/**
 * Native HTTPS Adapter for Axios
 *
 * Routes HTTPS requests through our native iOS module's URLSession with
 * TOFU certificate pinning, bypassing React Native's RCTHTTPRequestHandler.
 *
 * Why: RCTHTTPRequestHandler's NSURLSession does not call dynamically injected
 * TLS challenge handlers (despite class_addMethod succeeding). Our native
 * module creates its own URLSession(delegate:) which correctly dispatches
 * challenges to our TOFU delegate — proven by diagnoseTLS succeeding while
 * RCT's session fails with ERR_NETWORK.
 */

import { AxiosError, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import { nativeFetch } from '../../../modules/certificate-pinning';

/**
 * Build full URL from baseURL and relative path.
 */
function buildURL(baseURL: string | undefined, url: string | undefined): string {
  if (!url) return baseURL || '';
  if (/^https?:\/\//i.test(url)) return url;
  const base = (baseURL || '').replace(/\/$/, '');
  const path = url.replace(/^\//, '');
  return `${base}/${path}`;
}

/**
 * Axios adapter that routes requests through the native certificate-pinning
 * module's URLSession. Handles TOFU certificate pinning for self-signed
 * certificates transparently.
 */
export async function nativeAdapter(config: InternalAxiosRequestConfig): Promise<AxiosResponse> {
  const url = buildURL(config.baseURL, config.url);
  const method = (config.method || 'GET').toUpperCase();

  // Flatten AxiosHeaders to plain Record<string, string>.
  // AxiosHeaders.toJSON() may omit headers on some JS engines (Hermes),
  // so we also try direct .get() access as a fallback.
  const headers: Record<string, string> = {};
  if (config.headers) {
    const raw =
      typeof config.headers.toJSON === 'function' ? config.headers.toJSON() : config.headers;
    for (const [key, value] of Object.entries(raw as Record<string, unknown>)) {
      if (value != null && typeof value !== 'object') {
        headers[key] = String(value);
      }
    }

    // Fallback: ensure Authorization header is included even if toJSON() missed it
    if (!('Authorization' in headers) && typeof config.headers.get === 'function') {
      const auth = config.headers.get('Authorization');
      if (auth) {
        headers.Authorization = String(auth);
      }
    }
  }

  if (__DEV__) {
    // eslint-disable-next-line no-console
    console.log('[nativeAdapter]', method, url, {
      headerKeys: Object.keys(headers),
      hasAuth: 'Authorization' in headers,
    });
  }

  // Body (already serialized by axios transformRequest for JSON payloads)
  const body = config.data != null ? String(config.data) : '';

  const timeoutMs = config.timeout || 30000;

  let result;
  try {
    result = await nativeFetch(url, method, headers, body, timeoutMs);
  } catch (e: unknown) {
    const message = e instanceof Error ? e.message : 'Native fetch failed';
    throw new AxiosError(message, 'ERR_NETWORK', config, true);
  }

  // Network/TLS error — no HTTP response received
  if (result.error) {
    // 4th arg = request (truthy signals "request was sent but no response")
    throw new AxiosError(result.error, result.code || 'ERR_NETWORK', config, {});
  }

  // Build AxiosResponse — data is left as a string so axios's default
  // transformResponse (JSON.parse) processes it before interceptors run
  const response: AxiosResponse = {
    data: result.body || '',
    status: result.status,
    statusText: '',
    headers: result.headers || {},
    config,
  };

  // Validate HTTP status (default: 2xx is success)
  const validateStatus = config.validateStatus || ((s: number) => s >= 200 && s < 300);
  if (!validateStatus(result.status)) {
    throw new AxiosError(
      `Request failed with status code ${result.status}`,
      result.status >= 500 ? 'ERR_BAD_RESPONSE' : 'ERR_BAD_REQUEST',
      config,
      true,
      response
    );
  }

  return response;
}
