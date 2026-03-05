/**
 * NetworkManager Connection File Builder
 *
 * Generates NetworkManager keyfile-format connection profiles for
 * Ethernet, WiFi, and VLAN interfaces with IPv4/IPv6 configuration.
 *
 * Reference: https://networkmanager.dev/docs/api/latest/nm-settings-keyfile.html
 */

import type { InterfaceConfig, AddressingConfig, IPv4Config, IPv6Config } from '../types/wizard';

const CONNECTION_NAME = 'boardingpass-enrollment';

export interface NmConnectionOptions {
  interfaceConfig: InterfaceConfig;
  addressing: AddressingConfig;
  connectionName?: string;
}

/**
 * Generate a NetworkManager connection file in keyfile format.
 */
export function generateNmConnection(options: NmConnectionOptions): string {
  const { interfaceConfig, addressing, connectionName = CONNECTION_NAME } = options;

  const sections: string[] = [];

  // [connection] section
  sections.push(buildConnectionSection(connectionName, interfaceConfig));

  // Interface-specific section
  if (interfaceConfig.interfaceType === 'wifi' && interfaceConfig.wifi) {
    sections.push(buildWifiSection(interfaceConfig.wifi.ssid));
    if (interfaceConfig.wifi.security !== 'open' && interfaceConfig.wifi.password) {
      sections.push(
        buildWifiSecuritySection(interfaceConfig.wifi.security, interfaceConfig.wifi.password)
      );
    }
  } else if (interfaceConfig.vlanId !== null) {
    sections.push(buildVlanSection(interfaceConfig.interfaceName, interfaceConfig.vlanId));
  } else {
    sections.push(buildEthernetSection());
  }

  // [ipv4] section
  sections.push(buildIPv4Section(addressing.ipv4));

  // [ipv6] section
  sections.push(buildIPv6Section(addressing.ipv6));

  return sections.join('\n\n') + '\n';
}

/**
 * Get the connection filename for use in /configure bundles.
 */
export function getConnectionFilename(connectionName = CONNECTION_NAME): string {
  return `${connectionName}.nmconnection`;
}

/**
 * Get the full /etc path for the connection file.
 */
export function getConnectionPath(connectionName = CONNECTION_NAME): string {
  return `NetworkManager/system-connections/${getConnectionFilename(connectionName)}`;
}

// ── Section Builders ──

function buildConnectionSection(name: string, iface: InterfaceConfig): string {
  const lines = ['[connection]', `id=${name}`, `uuid=${generateDeterministicUUID(name)}`];

  if (iface.interfaceType === 'wifi') {
    lines.push('type=wifi');
  } else if (iface.vlanId !== null) {
    lines.push('type=vlan');
  } else {
    lines.push('type=ethernet');
  }

  // Bind to specific interface (except VLAN which uses parent)
  if (iface.vlanId === null) {
    lines.push(`interface-name=${iface.interfaceName}`);
  }

  lines.push('autoconnect=true');

  return lines.join('\n');
}

function buildEthernetSection(): string {
  return '[ethernet]';
}

function buildWifiSection(ssid: string): string {
  return ['[wifi]', `ssid=${ssid}`, 'mode=infrastructure'].join('\n');
}

function buildWifiSecuritySection(security: string, password: string): string {
  const lines = ['[wifi-security]'];

  if (security === 'wpa3' || security === 'SAE') {
    lines.push('key-mgmt=sae');
  } else {
    // WPA2 / WPA-PSK default
    lines.push('key-mgmt=wpa-psk');
  }

  lines.push(`psk=${password}`);

  return lines.join('\n');
}

function buildVlanSection(parentInterface: string, vlanId: number): string {
  return ['[vlan]', `parent=${parentInterface}`, `id=${vlanId}`].join('\n');
}

function buildIPv4Section(ipv4: IPv4Config): string {
  const lines = ['[ipv4]'];

  if (ipv4.method === 'static') {
    lines.push('method=manual');

    if (ipv4.address && ipv4.subnetMask) {
      const prefix = toCIDRPrefix(ipv4.subnetMask);
      lines.push(`address1=${ipv4.address}/${prefix}`);
    }

    if (ipv4.gateway) {
      lines.push(`gateway=${ipv4.gateway}`);
    }
  } else {
    lines.push('method=auto');
  }

  if (!ipv4.dnsAuto) {
    const dnsServers: string[] = [];
    if (ipv4.dnsPrimary) dnsServers.push(ipv4.dnsPrimary);
    if (ipv4.dnsSecondary) dnsServers.push(ipv4.dnsSecondary);
    if (dnsServers.length > 0) {
      lines.push(`dns=${dnsServers.join(';')};`);
      // Ignore auto DNS when manual DNS is set
      if (ipv4.method === 'dhcp') {
        lines.push('ignore-auto-dns=true');
      }
    }
  }

  return lines.join('\n');
}

function buildIPv6Section(ipv6: IPv6Config): string {
  const lines = ['[ipv6]'];

  if (ipv6.method === 'disabled') {
    lines.push('method=disabled');
    return lines.join('\n');
  }

  if (ipv6.method === 'static') {
    lines.push('method=manual');

    if (ipv6.address) {
      // IPv6 address should already include prefix (e.g., "2001:db8::1/64")
      lines.push(`address1=${ipv6.address}`);
    }

    if (ipv6.gateway) {
      lines.push(`gateway=${ipv6.gateway}`);
    }
  } else {
    lines.push('method=auto');
  }

  if (!ipv6.dnsAuto) {
    const dnsServers: string[] = [];
    if (ipv6.dnsPrimary) dnsServers.push(ipv6.dnsPrimary);
    if (ipv6.dnsSecondary) dnsServers.push(ipv6.dnsSecondary);
    if (dnsServers.length > 0) {
      lines.push(`dns=${dnsServers.join(';')};`);
      if (ipv6.method === 'dhcp') {
        lines.push('ignore-auto-dns=true');
      }
    }
  }

  return lines.join('\n');
}

// ── Helpers ──

/**
 * Convert a dotted-decimal subnet mask or CIDR string to prefix length.
 */
function toCIDRPrefix(mask: string): number {
  const cidr = parseInt(mask, 10);
  if (String(cidr) === mask && cidr >= 0 && cidr <= 32) {
    return cidr;
  }

  const parts = mask.split('.').map(Number);
  let bits = 0;
  for (const octet of parts) {
    bits += octet.toString(2).split('1').length - 1;
  }
  return bits;
}

/**
 * Generate a deterministic UUID v4-like string from a connection name.
 * Not cryptographically random — just a stable identifier for NM.
 */
function generateDeterministicUUID(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    const char = name.charCodeAt(i);
    // eslint-disable-next-line no-bitwise
    hash = ((hash << 5) - hash + char) | 0;
  }

  const hex = Math.abs(hash).toString(16).padStart(8, '0');
  // Pad to fill UUID format
  const padded = (hex + hex + hex + hex).slice(0, 32);
  return `${padded.slice(0, 8)}-${padded.slice(8, 12)}-4${padded.slice(13, 16)}-a${padded.slice(17, 20)}-${padded.slice(20, 32)}`;
}
