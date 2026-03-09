/**
 * WiFi Discovery Service
 *
 * Detects when the phone is connected to a BoardingPass WiFi hotspot
 * by checking the SSID pattern and probing the gateway IP.
 *
 * Contract: See specs/006-transient-transports/contracts/discovery-methods.md
 */

import NetInfo from '@react-native-community/netinfo';
import axios from 'axios';
import { Device } from '@/types/device';

const BOARDINGPASS_SSID_PREFIX = 'BoardingPass-';
const DEFAULT_PORT = 8443;
const PROBE_TIMEOUT = 5000;

export type WiFiDeviceCallback = (device: Device | null) => void;

export class WiFiDiscoveryService {
  private callback: WiFiDeviceCallback | null = null;
  private unsubscribe: (() => void) | null = null;
  private isRunning = false;

  /**
   * Start monitoring WiFi state for BoardingPass hotspots.
   */
  public start(): void {
    if (this.isRunning) return;
    this.isRunning = true;

    this.unsubscribe = NetInfo.addEventListener(state => {
      if (state.type === 'wifi' && state.isConnected) {
        this.checkForBoardingPass(state.details);
      }
    });

    // Perform initial check
    this.checkCurrent();
  }

  /**
   * Stop monitoring WiFi state.
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
  public onDeviceFound(callback: WiFiDeviceCallback): () => void {
    this.callback = callback;
    return () => {
      this.callback = null;
    };
  }

  /**
   * Check current network state immediately.
   */
  public async checkCurrent(): Promise<Device | null> {
    try {
      const state = await NetInfo.fetch();
      if (state.type === 'wifi' && state.isConnected && state.details) {
        return this.checkForBoardingPass(state.details);
      }
    } catch {
      // eslint-disable-next-line no-console
      console.log('WiFi discovery: failed to check current state');
    }
    return null;
  }

  public isActive(): boolean {
    return this.isRunning;
  }

  private async checkForBoardingPass(
    details: { ssid?: string | null; ipAddress?: string | null } | null
  ): Promise<Device | null> {
    if (!details) return null;

    const ssid = details.ssid;

    // Check SSID pattern match
    if (ssid && ssid.startsWith(BOARDINGPASS_SSID_PREFIX)) {
      const deviceName = ssid.substring(BOARDINGPASS_SSID_PREFIX.length);
      const gatewayIP = await this.getGatewayIP(details);
      if (gatewayIP) {
        const device = this.createDevice(gatewayIP, deviceName);
        this.callback?.(device);
        return device;
      }
    }

    // Fallback: probe gateway even without SSID access
    const gatewayIP = await this.getGatewayIP(details);
    if (gatewayIP) {
      const reachable = await this.probeGateway(gatewayIP);
      if (reachable) {
        const device = this.createDevice(gatewayIP, 'Unknown Device');
        this.callback?.(device);
        return device;
      }
    }

    return null;
  }

  private async getGatewayIP(
    details: { ipAddress?: string | null } | null
  ): Promise<string | null> {
    if (!details?.ipAddress) return null;

    // Derive gateway from device IP (assumes /24 subnet with .1 gateway)
    const parts = details.ipAddress.split('.');
    if (parts.length === 4) {
      parts[3] = '1';
      return parts.join('.');
    }
    return null;
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

  private createDevice(host: string, name: string): Device {
    return {
      id: `wifi:${host}`,
      name,
      host,
      port: DEFAULT_PORT,
      addresses: [host],
      discoveryMethod: 'wifi',
      status: 'online',
      lastSeen: new Date(),
    };
  }
}

// Singleton
let instance: WiFiDiscoveryService | null = null;

export function getWiFiDiscoveryService(): WiFiDiscoveryService {
  if (!instance) {
    instance = new WiFiDiscoveryService();
  }
  return instance;
}

export function resetWiFiDiscoveryService(): void {
  if (instance) {
    instance.stop();
    instance = null;
  }
}
