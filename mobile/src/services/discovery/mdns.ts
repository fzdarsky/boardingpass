/**
 * mDNS Service Discovery
 *
 * Discovers BoardingPass devices on local network via mDNS/Bonjour.
 * Service type: _boardingpass._tcp
 * Domain: local.
 *
 * Contract: See mobile/tests/contract/mdns.test.ts
 */

import Zeroconf from 'react-native-zeroconf';
import { Device } from '@/types/device';

export interface MDNSService {
  name: string;
  host: string;
  port: number;
  addresses: string[];
  txt?: Record<string, string>;
}

export type MDNSEventListener = (device: Device) => void;

export class MDNSDiscoveryService {
  private zeroconf: Zeroconf;
  private serviceType: string;
  private domain: string;
  private isScanning: boolean = false;

  constructor(serviceType: string = '_boardingpass._tcp', domain: string = 'local.') {
    this.zeroconf = new Zeroconf();
    this.serviceType = serviceType;
    this.domain = domain;
  }

  /**
   * Start scanning for BoardingPass devices
   */
  public start(): void {
    if (this.isScanning) {
      console.warn('mDNS scan already in progress');
      return;
    }

    try {
      this.zeroconf.scan(this.serviceType, 'tcp', this.domain);
      this.isScanning = true;
      // eslint-disable-next-line no-console
      console.log(`Started mDNS scan for ${this.serviceType}`);
    } catch (error) {
      console.error('Failed to start mDNS scan:', error);
      throw new Error('mDNS not available');
    }
  }

  /**
   * Stop scanning for devices
   */
  public stop(): void {
    if (!this.isScanning) {
      return;
    }

    try {
      this.zeroconf.stop();
      this.isScanning = false;
      // eslint-disable-next-line no-console
      console.log('Stopped mDNS scan');
    } catch (error) {
      console.error('Failed to stop mDNS scan:', error);
    }
  }

  /**
   * Register listener for device found events
   */
  public onDeviceFound(callback: MDNSEventListener): () => void {
    const listener = (service: MDNSService) => {
      const device = this.mapServiceToDevice(service);
      callback(device);
    };

    this.zeroconf.on('found', listener);

    // Return cleanup function
    return () => {
      this.zeroconf.removeListener('found', listener);
    };
  }

  /**
   * Register listener for device removed events
   */
  public onDeviceRemoved(callback: (deviceId: string) => void): () => void {
    const listener = (service: MDNSService) => {
      const deviceId = this.generateDeviceId(service.name, service.host);
      callback(deviceId);
    };

    this.zeroconf.on('remove', listener);

    // Return cleanup function
    return () => {
      this.zeroconf.removeListener('remove', listener);
    };
  }

  /**
   * Register listener for resolution events (when service is resolved with full details)
   */
  public onDeviceResolved(callback: MDNSEventListener): () => void {
    const listener = (service: MDNSService) => {
      const device = this.mapServiceToDevice(service);
      callback(device);
    };

    this.zeroconf.on('resolved', listener);

    // Return cleanup function
    return () => {
      this.zeroconf.removeListener('resolved', listener);
    };
  }

  /**
   * Register listener for errors
   */
  public onError(callback: (error: Error) => void): () => void {
    const listener = (err: unknown) => {
      const error = err instanceof Error ? err : new Error(String(err));
      callback(error);
    };

    this.zeroconf.on('error', listener);

    // Return cleanup function
    return () => {
      this.zeroconf.removeListener('error', listener);
    };
  }

  /**
   * Map mDNS service to Device entity
   */
  private mapServiceToDevice(service: MDNSService): Device {
    const id = this.generateDeviceId(service.name, service.host);

    return {
      id,
      name: service.name,
      host: service.host,
      port: service.port || 9443, // Default to 9443 if not specified
      addresses: service.addresses || [service.host],
      discoveryMethod: 'mdns',
      txt: service.txt,
      status: 'online',
      lastSeen: new Date(),
    };
  }

  /**
   * Generate unique device ID from name and host
   * Format: name:host
   */
  private generateDeviceId(name: string, host: string): string {
    return `${name}:${host}`;
  }

  /**
   * Check if scanning is in progress
   */
  public isActive(): boolean {
    return this.isScanning;
  }
}

// Singleton instance
let instance: MDNSDiscoveryService | null = null;

/**
 * Get singleton instance of mDNS discovery service
 */
export function getMDNSDiscoveryService(): MDNSDiscoveryService {
  if (!instance) {
    // Use environment variables or defaults
    const serviceType = process.env.EXPO_PUBLIC_MDNS_SERVICE || '_boardingpass._tcp';
    instance = new MDNSDiscoveryService(serviceType);
  }
  return instance;
}

/**
 * Reset singleton instance (for testing)
 */
export function resetMDNSDiscoveryService(): void {
  if (instance) {
    instance.stop();
    instance = null;
  }
}
