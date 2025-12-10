/**
 * Fallback IP Detection Service
 *
 * Checks well-known IP address (default: 192.168.1.100:9443) when mDNS is unavailable.
 * Uses HTTPS HEAD request to verify device is reachable.
 *
 * Contract: See mobile/tests/integration/discovery.test.ts
 */

import axios, { AxiosError } from 'axios';
import { Device } from '@/types/device';

export interface FallbackConfig {
  ip: string;
  port: number;
  timeout?: number;
}

export class FallbackIPService {
  private config: FallbackConfig;

  constructor(config?: Partial<FallbackConfig>) {
    // Use environment variables or defaults
    const defaultIP = process.env.EXPO_PUBLIC_FALLBACK_IP || '192.168.1.100';
    const defaultPort = parseInt(process.env.EXPO_PUBLIC_FALLBACK_PORT || '9443', 10);

    this.config = {
      ip: config?.ip || defaultIP,
      port: config?.port || defaultPort,
      timeout: config?.timeout || 5000, // 5 second timeout
    };
  }

  /**
   * Check if fallback device is reachable
   * Returns Device if reachable, null otherwise
   */
  public async check(): Promise<Device | null> {
    const url = `https://${this.config.ip}:${this.config.port}/`;

    try {
      // Use HEAD request to check if device is reachable
      // Accept any status code (including self-signed cert errors)
      // We just want to know if something is listening
      await axios.head(url, {
        timeout: this.config.timeout,
        validateStatus: () => true, // Accept any status
      });

      // Device is reachable, create Device entity
      const device = this.createFallbackDevice();
      // eslint-disable-next-line no-console
      console.log('Fallback device found:', device.id);
      return device;
    } catch (error) {
      // Device not reachable
      if (axios.isAxiosError(error)) {
        const axiosError = error as AxiosError;
        // eslint-disable-next-line no-console
        console.log(`Fallback device not reachable: ${axiosError.code || axiosError.message}`);
      } else {
        // eslint-disable-next-line no-console
        console.log('Fallback device check failed:', error);
      }
      return null;
    }
  }

  /**
   * Create Device entity for fallback device
   */
  private createFallbackDevice(): Device {
    const id = `fallback:${this.config.ip}`;

    return {
      id,
      name: 'BoardingPass Device', // Generic name for fallback device
      host: this.config.ip,
      port: this.config.port,
      addresses: [this.config.ip],
      discoveryMethod: 'fallback',
      status: 'online',
      lastSeen: new Date(),
    };
  }

  /**
   * Get current fallback configuration
   */
  public getConfig(): FallbackConfig {
    return { ...this.config };
  }

  /**
   * Update fallback configuration
   */
  public updateConfig(config: Partial<FallbackConfig>): void {
    this.config = {
      ...this.config,
      ...config,
    };
  }
}

// Singleton instance
let instance: FallbackIPService | null = null;

/**
 * Get singleton instance of fallback IP service
 */
export function getFallbackIPService(): FallbackIPService {
  if (!instance) {
    instance = new FallbackIPService();
  }
  return instance;
}

/**
 * Reset singleton instance (for testing)
 */
export function resetFallbackIPService(): void {
  instance = null;
}
