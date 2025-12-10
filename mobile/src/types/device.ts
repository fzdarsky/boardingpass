/**
 * Device Entity Types
 *
 * Represents a headless Linux device running BoardingPass service,
 * either discovered via mDNS or accessed via well-known IP address.
 */

import { CertificateInfo } from './certificate';

export type DiscoveryMethod = 'mdns' | 'fallback' | 'manual';

export type DeviceStatus = 'online' | 'offline' | 'authenticating' | 'authenticated' | 'error';

export interface Device {
  // Identity
  id: string;
  name: string;

  // Network
  host: string;
  port: number;
  addresses: string[];

  // Discovery metadata
  discoveryMethod: DiscoveryMethod;
  txt?: Record<string, string>;

  // Status
  status: DeviceStatus;
  lastSeen: Date;

  // Relationships (loaded separately)
  certificateInfo?: CertificateInfo;
  systemInfo?: DeviceInformation;
  networkConfig?: NetworkConfiguration;
}

/**
 * DeviceInformation - System details from /info endpoint
 *
 * Matches SystemInfo schema from BoardingPass API
 */
export interface DeviceInformation {
  // TPM information
  tpm: {
    present: boolean;
    version?: string;
    manufacturer?: string;
  };

  // Board/hardware information
  board: {
    manufacturer?: string;
    productName?: string;
    serialNumber?: string;
    uuid?: string;
  };

  // CPU information
  cpu: {
    model?: string;
    architecture?: string;
    cores?: number;
  };

  // Operating system information
  os: {
    distribution?: string;
    version?: string;
    kernel?: string;
    hostname?: string;
  };

  // FIPS mode status
  fips: {
    enabled: boolean;
    validated?: boolean;
  };

  // Metadata
  retrievedAt: Date;
}

/**
 * NetworkConfiguration - Network details from /network endpoint
 */
export interface NetworkConfiguration {
  interfaces: NetworkInterface[];
  retrievedAt: Date;
}

export type LinkState = 'up' | 'down' | 'unknown';

export type AddressFamily = 'ipv4' | 'ipv6';

export type AddressScope = 'host' | 'link' | 'global' | 'site';

export interface NetworkInterface {
  // Interface identity
  name: string;
  index: number;

  // Hardware
  macAddress?: string;

  // Status
  linkState: LinkState;
  operationalState?: string;

  // IP addresses
  addresses: IPAddress[];

  // Interface type metadata
  type?: string;
}

export interface IPAddress {
  address: string;
  family: AddressFamily;
  prefixLength: number;
  scope?: AddressScope;
}

/**
 * Validation helpers
 */
export const isValidPort = (port: number): boolean => {
  return Number.isInteger(port) && port >= 1 && port <= 65535;
};

export const isValidDeviceName = (name: string): boolean => {
  return name.length > 0 && name.length <= 255;
};

export const isDeviceOnline = (device: Device): boolean => {
  return device.status === 'online' || device.status === 'authenticated';
};

export const isDeviceAuthenticated = (device: Device): boolean => {
  return device.status === 'authenticated';
};
