/**
 * Base API Client
 *
 * Axios-based HTTP client for BoardingPass API communication.
 * Handles HTTPS, timeouts, authentication headers, and error handling.
 */

import axios, { AxiosInstance, AxiosRequestConfig, AxiosError } from 'axios';
import { Platform } from 'react-native';
import { nativeAdapter } from './nativeAdapter';

const DEFAULT_TIMEOUT = 30000; // 30 seconds (matches spec)
const DEFAULT_PORT = 8443;

export interface APIClientConfig {
  baseURL?: string;
  timeout?: number;
  rejectUnauthorized?: boolean; // Set to false for self-signed certs after user confirms
}

/**
 * Custom error class that preserves network error details through the error chain.
 * Carries the original error code (ERR_NETWORK, ECONNREFUSED, ECONNABORTED, etc.)
 * and the underlying cause message so downstream error handling can distinguish
 * between routing failures, TLS rejections, timeouts, and HTTP errors.
 */
export class APIError extends Error {
  code: string;
  status?: number;

  constructor(message: string, code: string, status?: number) {
    super(message);
    this.name = 'APIError';
    this.code = code;
    this.status = status;
  }
}

export class APIClient {
  private client: AxiosInstance;
  private authToken: string | null = null;

  constructor(host: string, port: number = DEFAULT_PORT, config: APIClientConfig = {}) {
    const baseURL = config.baseURL || `https://${host}:${port}`;

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[APIClient] Creating client', {
        platform: Platform.OS,
        useNativeAdapter: Platform.OS === 'ios',
        nativeAdapterDefined: typeof nativeAdapter === 'function',
      });
    }

    this.client = axios.create({
      baseURL,
      timeout: config.timeout || DEFAULT_TIMEOUT,
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      // On iOS, route HTTPS requests through our native module's URLSession
      // which has a proper TOFU delegate for self-signed certificates.
      // RCTHTTPRequestHandler's session ignores dynamically injected TLS
      // challenge handlers, so we bypass it entirely.
      ...(Platform.OS === 'ios' ? { adapter: nativeAdapter } : {}),
    });

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[APIClient] Client created', {
        adapterSet: typeof this.client.defaults.adapter,
        baseURL,
      });
    }

    // Request interceptor to add auth token
    this.client.interceptors.request.use(
      requestConfig => {
        if (this.authToken) {
          requestConfig.headers.Authorization = `Bearer ${this.authToken}`;
        }
        return requestConfig;
      },
      error => Promise.reject(error)
    );

    // Response interceptor for error handling
    this.client.interceptors.response.use(
      response => response,
      (error: AxiosError) => {
        if (error.response) {
          // Server responded with error status — we reached the server
          const status = error.response.status;
          const data = error.response.data as { error?: string; message?: string };
          // Prefer the descriptive 'message' field, fall back to 'error' code
          const serverMsg = data?.message || data?.error || `HTTP ${status}`;
          throw new APIError(serverMsg, `HTTP_${status}`, status);
        } else if (error.request) {
          // Request made but no response — network/TLS level failure
          const code = error.code || 'UNKNOWN';
          const cause = error.message || 'Unknown cause';

          if (__DEV__) {
            console.warn(`[APIClient] Request failed: code=${code}, cause=${cause}`);
          }

          // Preserve the original error code and cause message so downstream
          // error handling can distinguish failure modes
          throw new APIError(`${describeNetworkError(code)}: ${cause}`, code);
        } else {
          throw new APIError(error.message || 'Request setup failed', 'ERR_SETUP');
        }
      }
    );
  }

  /**
   * Set authentication token for subsequent requests
   */
  setAuthToken(token: string | null): void {
    this.authToken = token;
  }

  /**
   * Get current authentication token
   */
  getAuthToken(): string | null {
    return this.authToken;
  }

  /**
   * Clear authentication token
   */
  clearAuthToken(): void {
    this.authToken = null;
  }

  /**
   * GET request
   */
  async get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.get<T>(url, config);
    return response.data;
  }

  /**
   * POST request
   */
  async post<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.post<T>(url, data, config);
    return response.data;
  }

  /**
   * PUT request
   */
  async put<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.put<T>(url, data, config);
    return response.data;
  }

  /**
   * DELETE request
   */
  async delete<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.delete<T>(url, config);
    return response.data;
  }

  /**
   * Get base URL
   */
  getBaseURL(): string {
    return this.client.defaults.baseURL || '';
  }

  /**
   * Update base URL (for switching between devices)
   */
  setBaseURL(host: string, port: number = DEFAULT_PORT): void {
    this.client.defaults.baseURL = `https://${host}:${port}`;
  }
}

/**
 * Map axios error codes to human-readable descriptions.
 * The original cause message is appended separately for full context.
 */
function describeNetworkError(code: string): string {
  switch (code) {
    case 'ECONNABORTED':
      return 'Request timed out';
    case 'ECONNREFUSED':
      return 'Connection refused (device may not be running)';
    case 'ECONNRESET':
      return 'Connection reset by device';
    case 'ENOTFOUND':
      return 'Device not found (DNS resolution failed)';
    case 'EHOSTUNREACH':
      return 'Device unreachable (no route to host)';
    case 'ENETUNREACH':
      return 'Network unreachable';
    case 'ETIMEDOUT':
      return 'Connection timed out';
    case 'ERR_NETWORK':
      return 'Network error (possible TLS rejection or connectivity issue)';
    default:
      return `Network error (${code})`;
  }
}

/**
 * Factory function to create API client for a device
 */
export function createAPIClient(host: string, port?: number, config?: APIClientConfig): APIClient {
  return new APIClient(host, port, config);
}
