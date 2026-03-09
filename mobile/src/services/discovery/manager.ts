/**
 * Discovery Manager
 *
 * Coordinates all device discovery methods (mDNS, fallback, WiFi, BLE, USB)
 * and aggregates discovered devices with de-duplication.
 */

import { Device } from '@/types/device';
import { getMDNSDiscoveryService } from './mdns';
import { getFallbackIPService } from './fallback';
import { getWiFiDiscoveryService } from './wifi';
import { getBLEDiscoveryService } from './bluetooth';
import { getUSBDiscoveryService } from './usb';
import { deduplicateDevices } from './preference';

export type DeviceFoundCallback = (device: Device) => void;

export class DiscoveryManager {
  private callbacks: DeviceFoundCallback[] = [];
  private isRunning = false;
  private discoveredDevices: Device[] = [];
  private cleanupFns: (() => void)[] = [];

  /**
   * Register a callback for when a device is discovered.
   */
  public onDeviceFound(callback: DeviceFoundCallback): () => void {
    this.callbacks.push(callback);
    return () => {
      this.callbacks = this.callbacks.filter(cb => cb !== callback);
    };
  }

  /**
   * Start all discovery methods.
   */
  public async startAll(): Promise<void> {
    if (this.isRunning) return;
    this.isRunning = true;

    // Start mDNS discovery
    try {
      const mdns = getMDNSDiscoveryService();
      const cleanup = mdns.onDeviceResolved((device: Device) => {
        this.handleDeviceFound(device);
      });
      this.cleanupFns.push(cleanup);
      mdns.start();
    } catch {
      // eslint-disable-next-line no-console
      console.log('mDNS discovery not available, skipping');
    }

    // Start WiFi discovery
    try {
      const wifi = getWiFiDiscoveryService();
      const cleanup = wifi.onDeviceFound(device => {
        if (device) {
          this.handleDeviceFound(device);
        }
      });
      this.cleanupFns.push(cleanup);
      wifi.start();
    } catch {
      // eslint-disable-next-line no-console
      console.log('WiFi discovery not available, skipping');
    }

    // Start BLE discovery
    try {
      const ble = getBLEDiscoveryService();
      const cleanup = ble.onDeviceFound(device => {
        this.handleDeviceFound(device);
      });
      this.cleanupFns.push(cleanup);
      await ble.start();
    } catch {
      // eslint-disable-next-line no-console
      console.log('BLE discovery not available, skipping');
    }

    // Start USB tethering discovery
    try {
      const usb = getUSBDiscoveryService();
      const cleanup = usb.onDeviceFound(device => {
        this.handleDeviceFound(device);
      });
      this.cleanupFns.push(cleanup);
      usb.start();
    } catch {
      // eslint-disable-next-line no-console
      console.log('USB discovery not available, skipping');
    }

    // Start fallback IP check
    try {
      const fallback = getFallbackIPService();
      const device = await fallback.check();
      if (device) {
        this.handleDeviceFound(device);
      }
    } catch {
      // eslint-disable-next-line no-console
      console.log('Fallback discovery failed, skipping');
    }
  }

  /**
   * Stop all discovery methods.
   */
  public stopAll(): void {
    if (!this.isRunning) return;
    this.isRunning = false;

    for (const cleanup of this.cleanupFns) {
      cleanup();
    }
    this.cleanupFns = [];

    try {
      getMDNSDiscoveryService().stop();
    } catch {
      // ignore
    }

    try {
      getWiFiDiscoveryService().stop();
    } catch {
      // ignore
    }

    try {
      getBLEDiscoveryService().stop();
    } catch {
      // ignore
    }

    try {
      getUSBDiscoveryService().stop();
    } catch {
      // ignore
    }

    this.discoveredDevices = [];
  }

  /**
   * Get all currently discovered devices (de-duplicated).
   */
  public getDevices(): Device[] {
    return deduplicateDevices(this.discoveredDevices);
  }

  /**
   * Check if discovery is running.
   */
  public isActive(): boolean {
    return this.isRunning;
  }

  private handleDeviceFound(device: Device): void {
    this.discoveredDevices.push(device);

    const deduplicated = deduplicateDevices(this.discoveredDevices);
    this.discoveredDevices = deduplicated;

    for (const cb of this.callbacks) {
      cb(device);
    }
  }
}

// Singleton instance
let instance: DiscoveryManager | null = null;

/**
 * Get singleton instance of discovery manager.
 */
export function getDiscoveryManager(): DiscoveryManager {
  if (!instance) {
    instance = new DiscoveryManager();
  }
  return instance;
}

/**
 * Reset singleton instance (for testing).
 */
export function resetDiscoveryManager(): void {
  if (instance) {
    instance.stopAll();
    instance = null;
  }
}
