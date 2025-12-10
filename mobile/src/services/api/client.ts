/**
 * Base API Client
 *
 * Axios-based HTTP client for BoardingPass API communication.
 * Handles HTTPS, timeouts, authentication headers, and error handling.
 */

import axios, { AxiosInstance, AxiosRequestConfig, AxiosError } from 'axios';

const DEFAULT_TIMEOUT = 30000; // 30 seconds (matches spec)
const DEFAULT_PORT = 9443;

export interface APIClientConfig {
  baseURL?: string;
  timeout?: number;
  rejectUnauthorized?: boolean; // Set to false for self-signed certs after user confirms
}

export class APIClient {
  private client: AxiosInstance;
  private authToken: string | null = null;

  constructor(host: string, port: number = DEFAULT_PORT, config: APIClientConfig = {}) {
    const baseURL = config.baseURL || `https://${host}:${port}`;

    this.client = axios.create({
      baseURL,
      timeout: config.timeout || DEFAULT_TIMEOUT,
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      // For self-signed certificates, we'll need to handle this at the native level
      // React Native doesn't support the Node.js rejectUnauthorized option
    });

    // Request interceptor to add auth token
    this.client.interceptors.request.use(
      config => {
        if (this.authToken) {
          config.headers.Authorization = `Bearer ${this.authToken}`;
        }
        return config;
      },
      error => Promise.reject(error)
    );

    // Response interceptor for error handling
    this.client.interceptors.response.use(
      response => response,
      (error: AxiosError) => {
        // Map axios errors to application errors
        if (error.response) {
          // Server responded with error status
          const status = error.response.status;
          const data = error.response.data as any;

          if (status === 401) {
            throw new Error('Authentication required');
          } else if (status === 403) {
            throw new Error('Access forbidden');
          } else if (status === 404) {
            throw new Error('Resource not found');
          } else if (status >= 500) {
            throw new Error(`Server error: ${data?.error || 'Unknown error'}`);
          } else {
            throw new Error(data?.error || `Request failed with status ${status}`);
          }
        } else if (error.request) {
          // Request made but no response received
          if (error.code === 'ECONNABORTED') {
            throw new Error('Request timeout');
          } else if (error.code === 'ERR_NETWORK') {
            throw new Error('Network error - cannot reach device');
          } else {
            throw new Error('No response from device');
          }
        } else {
          // Something else happened
          throw new Error(error.message || 'Unknown error');
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
  async post<T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.post<T>(url, data, config);
    return response.data;
  }

  /**
   * PUT request
   */
  async put<T>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
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
 * Factory function to create API client for a device
 */
export function createAPIClient(host: string, port?: number, config?: APIClientConfig): APIClient {
  return new APIClient(host, port, config);
}
