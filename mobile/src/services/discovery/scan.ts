/**
 * Subnet Scanner Discovery Service
 *
 * Actively scans the local subnet to find BoardingPass devices when mDNS
 * is unavailable. Determines the subnet to scan from the phone's current
 * network state (WiFi IP, USB tethering subnet, etc.).
 *
 * Replaces the previous fallback IP check and USB gateway probing, which
 * only checked single hardcoded IPs. The scanner probes all hosts in the
 * subnet to find the actual device IP.
 */

import NetInfo, { NetInfoState } from '@react-native-community/netinfo';
import { Platform } from 'react-native';
import { Device } from '@/types/device';
import { DEFAULT_BOARDINGPASS_PORT } from '@/constants/network';
import { probeDevice } from '../reachability/probe';

/** Timeout per host during scanning (shorter than normal probe for speed) */
const SCAN_TIMEOUT = 1500;

/** Number of hosts to probe in parallel per batch */
const BATCH_SIZE = 20;

/** Delay between batches (ms) to avoid overwhelming the network */
const BATCH_DELAY = 100;

/** Delay before auto-retry when no devices found (ms) */
const RETRY_DELAY = 3000;

export type ScanDeviceCallback = (device: Device) => void;
export type ScanProgressCallback = (scanned: number, total: number) => void;

interface SubnetRange {
  network: string; // e.g. "192.168.1"
  startHost: number; // e.g. 1
  endHost: number; // e.g. 254
  selfIP?: string; // phone's own IP to skip
}

export class SubnetScannerService {
  private deviceCallbacks: ScanDeviceCallback[] = [];
  private progressCallbacks: ScanProgressCallback[] = [];
  private running = false;
  private abortController: AbortController | null = null;
  private foundCount = 0;

  /**
   * Start a subnet scan. Determines subnets from current network state.
   */
  public async start(): Promise<void> {
    if (this.running) return;
    this.running = true;
    this.abortController = new AbortController();

    try {
      this.foundCount = 0;
      await this.scanAllSubnets();

      // If nothing found, wait and retry once — covers the race where a
      // USB tethering listener isn't ready yet on the service side.
      if (this.foundCount === 0 && !this.abortController.signal.aborted) {
        await new Promise(resolve => setTimeout(resolve, RETRY_DELAY));
        if (!this.abortController.signal.aborted) {
          // eslint-disable-next-line no-console
          console.log('[Scanner] No devices found, retrying...');
          await this.scanAllSubnets();
        }
      }
    } catch (error) {
      if (!this.abortController?.signal.aborted) {
        // eslint-disable-next-line no-console
        console.log('[Scanner] Scan error:', error);
      }
    } finally {
      this.running = false;
      this.abortController = null;
    }
  }

  /**
   * Stop an in-progress scan.
   */
  public stop(): void {
    if (!this.running) return;
    this.abortController?.abort();
    this.running = false;
  }

  /**
   * Register a callback for discovered devices.
   */
  public onDeviceFound(callback: ScanDeviceCallback): () => void {
    this.deviceCallbacks.push(callback);
    return () => {
      this.deviceCallbacks = this.deviceCallbacks.filter(cb => cb !== callback);
    };
  }

  /**
   * Register a callback for scan progress updates.
   */
  public onProgress(callback: ScanProgressCallback): () => void {
    this.progressCallbacks.push(callback);
    return () => {
      this.progressCallbacks = this.progressCallbacks.filter(cb => cb !== callback);
    };
  }

  public isActive(): boolean {
    return this.running;
  }

  /**
   * Run a single pass over all scannable subnets.
   */
  private async scanAllSubnets(): Promise<void> {
    const subnets = await this.getSubnetsToScan();

    if (subnets.length === 0) {
      // eslint-disable-next-line no-console
      console.log('[Scanner] No scannable subnets found');
      return;
    }

    for (const subnet of subnets) {
      if (this.abortController?.signal.aborted) break;
      const hosts = this.generateHosts(subnet);
      await this.scanHosts(hosts, DEFAULT_BOARDINGPASS_PORT);
    }
  }

  /**
   * Determine which subnets to scan based on current network state.
   *
   * Always includes platform-specific tethering subnets (iOS: 172.20.10.0/28,
   * just 14 hosts) since USB tethering can be active alongside WiFi but
   * NetInfo only reports the primary connection.
   */
  public async getSubnetsToScan(): Promise<SubnetRange[]> {
    let state: NetInfoState;
    try {
      state = await NetInfo.fetch();
    } catch {
      return [];
    }

    if (!state.isConnected) return [];

    // Scan tethering subnets first — they're small (iOS: 14 hosts) and
    // most likely to contain a BoardingPass device on a direct connection
    const subnets: SubnetRange[] = [...this.getTetheringSubnets()];

    // WiFi connected — also scan the /24 around the phone's IP
    if (state.type === 'wifi') {
      const details = state.details as { ipAddress?: string } | null;
      const ip = details?.ipAddress;
      if (ip && !ip.startsWith('169.254.')) {
        const wifiSubnet = this.subnetFromIP(ip);
        const isDuplicate = subnets.some(s => s.network === wifiSubnet.network);
        if (!isDuplicate) {
          subnets.push(wifiSubnet);
        }
      }
    }

    return subnets;
  }

  /**
   * Get tethering subnets based on platform.
   * iOS uses 172.20.10.0/28 (14 hosts), Android uses 192.168.42.0/24 and 192.168.43.0/24.
   */
  private getTetheringSubnets(): SubnetRange[] {
    if (Platform.OS === 'ios') {
      return [{ network: '172.20.10', startHost: 1, endHost: 14 }];
    }
    // Android tethering subnets
    return [
      { network: '192.168.42', startHost: 1, endHost: 254 },
      { network: '192.168.43', startHost: 1, endHost: 254 },
    ];
  }

  /**
   * Derive a /24 subnet range from an IP address.
   */
  private subnetFromIP(ip: string): SubnetRange {
    const parts = ip.split('.');
    const network = parts.slice(0, 3).join('.');
    return { network, startHost: 1, endHost: 254, selfIP: ip };
  }

  /**
   * Generate the list of host IPs to probe, excluding the phone's own IP.
   */
  public generateHosts(subnet: SubnetRange): string[] {
    const hosts: string[] = [];
    for (let i = subnet.startHost; i <= subnet.endHost; i++) {
      const ip = `${subnet.network}.${i}`;
      if (ip !== subnet.selfIP) {
        hosts.push(ip);
      }
    }
    return hosts;
  }

  /**
   * Scan a list of hosts in batches, probing each for a BoardingPass service.
   */
  private async scanHosts(hosts: string[], port: number): Promise<void> {
    const total = hosts.length;
    let scanned = 0;

    for (let i = 0; i < hosts.length; i += BATCH_SIZE) {
      if (this.abortController?.signal.aborted) return;

      const batch = hosts.slice(i, i + BATCH_SIZE);
      const results = await Promise.allSettled(
        batch.map(host => probeDevice(host, port, SCAN_TIMEOUT))
      );

      for (let j = 0; j < results.length; j++) {
        const result = results[j];
        if (result.status === 'fulfilled' && result.value === 'online') {
          const host = batch[j];
          const device: Device = {
            id: `scan:${host}:${port}`,
            name: `Device (${host})`,
            host,
            port,
            addresses: [host],
            discoveryMethod: 'scan',
            status: 'online',
            lastSeen: new Date(),
          };

          this.foundCount++;
          for (const cb of this.deviceCallbacks) {
            cb(device);
          }
        }
      }

      scanned += batch.length;
      for (const cb of this.progressCallbacks) {
        cb(scanned, total);
      }

      // Brief pause between batches to avoid network saturation
      if (i + BATCH_SIZE < hosts.length && !this.abortController?.signal.aborted) {
        await new Promise(resolve => setTimeout(resolve, BATCH_DELAY));
      }
    }
  }
}

// Singleton
let instance: SubnetScannerService | null = null;

export function getSubnetScannerService(): SubnetScannerService {
  if (!instance) {
    instance = new SubnetScannerService();
  }
  return instance;
}

export function resetSubnetScannerService(): void {
  if (instance) {
    instance.stop();
    instance = null;
  }
}
