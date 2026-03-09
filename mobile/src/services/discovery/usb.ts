/**
 * USB Tethering Discovery Service
 *
 * Detects when the phone is connected via USB tethering by probing
 * well-known tethering gateway IPs.
 *
 * Contract: See specs/006-transient-transports/contracts/discovery-methods.md
 */

import NetInfo from '@react-native-community/netinfo';
import axios from 'axios';
import { Device } from '@/types/device';

// Well-known tethering gateway IPs
const IOS_TETHERING_GATEWAY = '172.20.10.1';
const ANDROID_TETHERING_GATEWAY = '192.168.42.1';
const ANDROID_TETHERING_GATEWAY_ALT = '192.168.43.1';

const DEFAULT_PORT = 8443;
const PROBE_TIMEOUT = 3000;

export type USBDeviceCallback = (device: Device) => void;

export class USBDiscoveryService {
  private callback: USBDeviceCallback | null = null;
  private unsubscribe: (() => void) | null = null;
  private isRunning = false;

  /**
   * Start monitoring for USB tethering connections.
   */
  public start(): void {
    if (this.isRunning) return;
    this.isRunning = true;

    this.unsubscribe = NetInfo.addEventListener(state => {
      // Look for non-WiFi, non-cellular connections (ethernet type covers USB)
      if (state.isConnected && state.type !== 'wifi' && state.type !== 'cellular') {
        this.probeTetheringGateways();
      }
    });

    // Initial check
    this.checkCurrent();
  }

  /**
   * Stop monitoring for USB tethering.
   */
  public stop(): void {
    if (!this.isRunning) return;
    this.isRunning = false;

    if (this.unsubscribe) {
      this.unsubscribe();
      this.unsubscribe = null;
    }
  }

  /**
   * Register callback for discovered devices.
   */
  public onDeviceFound(callback: USBDeviceCallback): () => void {
    this.callback = callback;
    return () => {
      this.callback = null;
    };
  }

  public isActive(): boolean {
    return this.isRunning;
  }

  /**
   * Check current connection state.
   */
  public async checkCurrent(): Promise<void> {
    try {
      const state = await NetInfo.fetch();
      if (state.isConnected && state.type !== 'wifi' && state.type !== 'cellular') {
        await this.probeTetheringGateways();
      }
    } catch {
      // eslint-disable-next-line no-console
      console.log('USB discovery: failed to check current state');
    }
  }

  private async probeTetheringGateways(): Promise<void> {
    const gateways = [
      IOS_TETHERING_GATEWAY,
      ANDROID_TETHERING_GATEWAY,
      ANDROID_TETHERING_GATEWAY_ALT,
    ];

    const results = await Promise.allSettled(gateways.map(ip => this.probeGateway(ip)));

    for (let i = 0; i < results.length; i++) {
      const result = results[i];
      if (result.status === 'fulfilled' && result.value) {
        const device: Device = {
          id: `usb:${gateways[i]}`,
          name: 'USB Device',
          host: gateways[i],
          port: DEFAULT_PORT,
          addresses: [gateways[i]],
          discoveryMethod: 'usb',
          status: 'online',
          lastSeen: new Date(),
        };
        this.callback?.(device);
        return; // Only report first reachable gateway
      }
    }
  }

  private async probeGateway(ip: string): Promise<boolean> {
    try {
      await axios.head(`https://${ip}:${DEFAULT_PORT}/`, {
        timeout: PROBE_TIMEOUT,
        validateStatus: () => true,
      });
      return true;
    } catch {
      return false;
    }
  }
}

// Singleton
let instance: USBDiscoveryService | null = null;

export function getUSBDiscoveryService(): USBDiscoveryService {
  if (!instance) {
    instance = new USBDiscoveryService();
  }
  return instance;
}

export function resetUSBDiscoveryService(): void {
  if (instance) {
    instance.stop();
    instance = null;
  }
}
