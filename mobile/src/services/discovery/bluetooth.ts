/**
 * BLE Discovery Service
 *
 * Scans for BoardingPass BLE advertisements and reads device connection
 * info from GATT characteristics.
 *
 * Contract: See specs/006-transient-transports/contracts/discovery-methods.md
 */

import { BleManager, Device as BleDevice, State as BleState } from 'react-native-ble-plx';
import { Device } from '@/types/device';

// Placeholder UUID — must match the BLE service advertised by the Linux device
const BOARDINGPASS_SERVICE_UUID = 'BBBBBBBB-BBBB-BBBB-BBBB-BBBBBBBBBBBB';
const CHAR_DEVICE_NAME = '00000001-BBBB-BBBB-BBBB-BBBBBBBBBBBB';
const CHAR_IP_ADDRESS = '00000002-BBBB-BBBB-BBBB-BBBBBBBBBBBB';
const CHAR_PORT = '00000003-BBBB-BBBB-BBBB-BBBBBBBBBBBB';
const CHAR_CERT_FINGERPRINT = '00000004-BBBB-BBBB-BBBB-BBBBBBBBBBBB';

const SCAN_TIMEOUT = 10000;

export type BLEDeviceCallback = (device: Device) => void;

export class BLEDiscoveryService {
  private manager: BleManager;
  private callback: BLEDeviceCallback | null = null;
  private isRunning = false;
  private scanTimeout: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    this.manager = new BleManager();
  }

  /**
   * Start scanning for BoardingPass BLE advertisements.
   */
  public async start(): Promise<void> {
    if (this.isRunning) return;

    const state = await this.manager.state();
    if (state !== BleState.PoweredOn) {
      // eslint-disable-next-line no-console
      console.log('BLE not powered on, waiting...');
      await new Promise<void>(resolve => {
        const sub = this.manager.onStateChange(newState => {
          if (newState === BleState.PoweredOn) {
            sub.remove();
            resolve();
          }
        }, true);
      });
    }

    this.isRunning = true;

    this.manager.startDeviceScan(
      [BOARDINGPASS_SERVICE_UUID],
      { allowDuplicates: false },
      (error, bleDevice) => {
        if (error) {
          // eslint-disable-next-line no-console
          console.log('BLE scan error:', error.message);
          return;
        }
        if (bleDevice) {
          this.handleDiscoveredDevice(bleDevice);
        }
      }
    );

    // Auto-stop after timeout
    this.scanTimeout = setTimeout(() => {
      this.stop();
    }, SCAN_TIMEOUT);
  }

  /**
   * Stop BLE scanning.
   */
  public stop(): void {
    if (!this.isRunning) return;
    this.isRunning = false;

    if (this.scanTimeout) {
      clearTimeout(this.scanTimeout);
      this.scanTimeout = null;
    }

    this.manager.stopDeviceScan();
  }

  /**
   * Register callback for discovered devices.
   */
  public onDeviceFound(callback: BLEDeviceCallback): () => void {
    this.callback = callback;
    return () => {
      this.callback = null;
    };
  }

  public isActive(): boolean {
    return this.isRunning;
  }

  private async handleDiscoveredDevice(bleDevice: BleDevice): Promise<void> {
    try {
      const connected = await bleDevice.connect();
      const discovered = await connected.discoverAllServicesAndCharacteristics();

      const [nameChar, ipChar, portChar, fingerprintChar] = await Promise.all([
        discovered.readCharacteristicForService(BOARDINGPASS_SERVICE_UUID, CHAR_DEVICE_NAME),
        discovered.readCharacteristicForService(BOARDINGPASS_SERVICE_UUID, CHAR_IP_ADDRESS),
        discovered.readCharacteristicForService(BOARDINGPASS_SERVICE_UUID, CHAR_PORT),
        discovered.readCharacteristicForService(BOARDINGPASS_SERVICE_UUID, CHAR_CERT_FINGERPRINT),
      ]);

      const name = nameChar.value ? atob(nameChar.value) : bleDevice.name || 'Unknown';
      const host = ipChar.value ? atob(ipChar.value) : '';
      const port = portChar.value ? parseInt(atob(portChar.value), 10) : 8443;
      const fingerprint = fingerprintChar.value ? atob(fingerprintChar.value) : undefined;

      await discovered.cancelConnection();

      if (!host) return;

      const device: Device = {
        id: `bluetooth:${bleDevice.id}`,
        name,
        host,
        port,
        addresses: [host],
        discoveryMethod: 'bluetooth',
        status: 'online',
        lastSeen: new Date(),
        txt: fingerprint ? { certFingerprint: fingerprint } : undefined,
      };

      this.callback?.(device);
    } catch {
      // eslint-disable-next-line no-console
      console.log('BLE: failed to read device info from', bleDevice.id);
    }
  }
}

// Singleton
let instance: BLEDiscoveryService | null = null;

export function getBLEDiscoveryService(): BLEDiscoveryService {
  if (!instance) {
    instance = new BLEDiscoveryService();
  }
  return instance;
}

export function resetBLEDiscoveryService(): void {
  if (instance) {
    instance.stop();
    instance = null;
  }
}
