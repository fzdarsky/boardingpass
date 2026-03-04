/**
 * NetworkManager Connection File Builder Tests
 *
 * Tests for generating NM keyfile-format connection profiles
 * for Ethernet, WiFi, VLAN, and various IPv4/IPv6 configurations.
 */

import { generateNmConnection, getConnectionPath } from '../../../src/utils/nm-connection';
import type { InterfaceConfig, AddressingConfig } from '../../../src/types/wizard';

const defaultAddressing: AddressingConfig = {
  ipv4: {
    method: 'dhcp',
    address: null,
    subnetMask: null,
    gateway: null,
    dnsAuto: true,
    dnsPrimary: null,
    dnsSecondary: null,
  },
  ipv6: {
    method: 'disabled',
    address: null,
    gateway: null,
    dnsAuto: true,
    dnsPrimary: null,
    dnsSecondary: null,
  },
};

describe('generateNmConnection', () => {
  describe('Ethernet connections', () => {
    const ethernetInterface: InterfaceConfig = {
      interfaceName: 'eth0',
      interfaceType: 'ethernet',
      vlanId: null,
      wifi: null,
    };

    it('generates basic Ethernet DHCP connection', () => {
      const result = generateNmConnection({
        interfaceConfig: ethernetInterface,
        addressing: defaultAddressing,
      });

      expect(result).toContain('[connection]');
      expect(result).toContain('type=ethernet');
      expect(result).toContain('interface-name=eth0');
      expect(result).toContain('autoconnect=true');
      expect(result).toContain('[ethernet]');
      expect(result).toContain('[ipv4]');
      expect(result).toContain('method=auto');
      expect(result).toContain('[ipv6]');
      expect(result).toContain('method=disabled');
    });

    it('generates Ethernet with static IPv4', () => {
      const addressing: AddressingConfig = {
        ...defaultAddressing,
        ipv4: {
          method: 'static',
          address: '192.168.1.100',
          subnetMask: '255.255.255.0',
          gateway: '192.168.1.1',
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
      };

      const result = generateNmConnection({
        interfaceConfig: ethernetInterface,
        addressing,
      });

      expect(result).toContain('method=manual');
      expect(result).toContain('address1=192.168.1.100/24');
      expect(result).toContain('gateway=192.168.1.1');
    });

    it('generates Ethernet with static IPv4 and manual DNS', () => {
      const addressing: AddressingConfig = {
        ...defaultAddressing,
        ipv4: {
          method: 'static',
          address: '192.168.1.100',
          subnetMask: '24',
          gateway: '192.168.1.1',
          dnsAuto: false,
          dnsPrimary: '8.8.8.8',
          dnsSecondary: '8.8.4.4',
        },
      };

      const result = generateNmConnection({
        interfaceConfig: ethernetInterface,
        addressing,
      });

      expect(result).toContain('dns=8.8.8.8;8.8.4.4;');
    });

    it('generates Ethernet with DHCP and manual DNS override', () => {
      const addressing: AddressingConfig = {
        ...defaultAddressing,
        ipv4: {
          method: 'dhcp',
          address: null,
          subnetMask: null,
          gateway: null,
          dnsAuto: false,
          dnsPrimary: '1.1.1.1',
          dnsSecondary: null,
        },
      };

      const result = generateNmConnection({
        interfaceConfig: ethernetInterface,
        addressing,
      });

      expect(result).toContain('method=auto');
      expect(result).toContain('dns=1.1.1.1;');
      expect(result).toContain('ignore-auto-dns=true');
    });
  });

  describe('IPv6 configurations', () => {
    const ethernetInterface: InterfaceConfig = {
      interfaceName: 'eth0',
      interfaceType: 'ethernet',
      vlanId: null,
      wifi: null,
    };

    it('generates IPv6 DHCP', () => {
      const addressing: AddressingConfig = {
        ...defaultAddressing,
        ipv6: {
          method: 'dhcp',
          address: null,
          gateway: null,
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
      };

      const result = generateNmConnection({
        interfaceConfig: ethernetInterface,
        addressing,
      });

      const ipv6Section = result.split('[ipv6]')[1];
      expect(ipv6Section).toContain('method=auto');
    });

    it('generates IPv6 static', () => {
      const addressing: AddressingConfig = {
        ...defaultAddressing,
        ipv6: {
          method: 'static',
          address: '2001:db8::1/64',
          gateway: '2001:db8::ffff',
          dnsAuto: false,
          dnsPrimary: '2001:4860:4860::8888',
          dnsSecondary: null,
        },
      };

      const result = generateNmConnection({
        interfaceConfig: ethernetInterface,
        addressing,
      });

      const ipv6Section = result.split('[ipv6]')[1];
      expect(ipv6Section).toContain('method=manual');
      expect(ipv6Section).toContain('address1=2001:db8::1/64');
      expect(ipv6Section).toContain('gateway=2001:db8::ffff');
      expect(ipv6Section).toContain('dns=2001:4860:4860::8888;');
    });

    it('generates IPv6 disabled', () => {
      const result = generateNmConnection({
        interfaceConfig: ethernetInterface,
        addressing: defaultAddressing,
      });

      const ipv6Section = result.split('[ipv6]')[1];
      expect(ipv6Section).toContain('method=disabled');
    });
  });

  describe('WiFi connections', () => {
    it('generates WiFi WPA2 connection', () => {
      const wifiInterface: InterfaceConfig = {
        interfaceName: 'wlan0',
        interfaceType: 'wifi',
        vlanId: null,
        wifi: {
          ssid: 'MyNetwork',
          bssid: '00:11:22:33:44:55',
          security: 'wpa2',
          password: 'secret123',
        },
      };

      const result = generateNmConnection({
        interfaceConfig: wifiInterface,
        addressing: defaultAddressing,
      });

      expect(result).toContain('type=wifi');
      expect(result).toContain('[wifi]');
      expect(result).toContain('ssid=MyNetwork');
      expect(result).toContain('mode=infrastructure');
      expect(result).toContain('[wifi-security]');
      expect(result).toContain('key-mgmt=wpa-psk');
      expect(result).toContain('psk=secret123');
    });

    it('generates WiFi WPA3 connection', () => {
      const wifiInterface: InterfaceConfig = {
        interfaceName: 'wlan0',
        interfaceType: 'wifi',
        vlanId: null,
        wifi: {
          ssid: 'SecureNet',
          bssid: '00:11:22:33:44:55',
          security: 'wpa3',
          password: 'strongpass',
        },
      };

      const result = generateNmConnection({
        interfaceConfig: wifiInterface,
        addressing: defaultAddressing,
      });

      expect(result).toContain('key-mgmt=sae');
      expect(result).toContain('psk=strongpass');
    });

    it('generates WiFi with SAE security string', () => {
      const wifiInterface: InterfaceConfig = {
        interfaceName: 'wlan0',
        interfaceType: 'wifi',
        vlanId: null,
        wifi: {
          ssid: 'SAENetwork',
          bssid: 'AA:BB:CC:DD:EE:FF',
          security: 'SAE',
          password: 'saepass',
        },
      };

      const result = generateNmConnection({
        interfaceConfig: wifiInterface,
        addressing: defaultAddressing,
      });

      expect(result).toContain('key-mgmt=sae');
      expect(result).toContain('psk=saepass');
    });

    it('generates WiFi with static IPv4 addressing', () => {
      const wifiInterface: InterfaceConfig = {
        interfaceName: 'wlan0',
        interfaceType: 'wifi',
        vlanId: null,
        wifi: {
          ssid: 'StaticNet',
          bssid: '00:11:22:33:44:55',
          security: 'wpa2',
          password: 'mypass',
        },
      };

      const staticAddressing: AddressingConfig = {
        ipv4: {
          method: 'static',
          address: '10.0.0.50',
          subnetMask: '255.255.255.0',
          gateway: '10.0.0.1',
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
        ipv6: {
          method: 'disabled',
          address: null,
          gateway: null,
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
      };

      const result = generateNmConnection({
        interfaceConfig: wifiInterface,
        addressing: staticAddressing,
      });

      // WiFi sections present
      expect(result).toContain('[wifi]');
      expect(result).toContain('ssid=StaticNet');
      expect(result).toContain('[wifi-security]');
      // Static IPv4 sections present
      expect(result).toContain('method=manual');
      expect(result).toContain('address1=10.0.0.50/24');
      expect(result).toContain('gateway=10.0.0.1');
    });

    it('binds WiFi connection to interface name', () => {
      const wifiInterface: InterfaceConfig = {
        interfaceName: 'wlp3s0',
        interfaceType: 'wifi',
        vlanId: null,
        wifi: {
          ssid: 'Test',
          bssid: '00:11:22:33:44:55',
          security: 'open',
          password: null,
        },
      };

      const result = generateNmConnection({
        interfaceConfig: wifiInterface,
        addressing: defaultAddressing,
      });

      expect(result).toContain('interface-name=wlp3s0');
    });

    it('does not include ethernet section for WiFi connections', () => {
      const wifiInterface: InterfaceConfig = {
        interfaceName: 'wlan0',
        interfaceType: 'wifi',
        vlanId: null,
        wifi: {
          ssid: 'Test',
          bssid: '00:11:22:33:44:55',
          security: 'wpa2',
          password: 'pass',
        },
      };

      const result = generateNmConnection({
        interfaceConfig: wifiInterface,
        addressing: defaultAddressing,
      });

      expect(result).not.toContain('[ethernet]');
      expect(result).not.toContain('[vlan]');
    });

    it('generates open WiFi connection without security section', () => {
      const wifiInterface: InterfaceConfig = {
        interfaceName: 'wlan0',
        interfaceType: 'wifi',
        vlanId: null,
        wifi: {
          ssid: 'OpenNet',
          bssid: '00:11:22:33:44:55',
          security: 'open',
          password: null,
        },
      };

      const result = generateNmConnection({
        interfaceConfig: wifiInterface,
        addressing: defaultAddressing,
      });

      expect(result).toContain('ssid=OpenNet');
      expect(result).not.toContain('[wifi-security]');
    });
  });

  describe('VLAN connections', () => {
    it('generates VLAN connection', () => {
      const vlanInterface: InterfaceConfig = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: 100,
        wifi: null,
      };

      const result = generateNmConnection({
        interfaceConfig: vlanInterface,
        addressing: defaultAddressing,
      });

      expect(result).toContain('type=vlan');
      expect(result).toContain('[vlan]');
      expect(result).toContain('parent=eth0');
      expect(result).toContain('id=100');
      // VLAN should not have interface-name
      expect(result).not.toContain('interface-name=');
    });
  });

  describe('connection naming', () => {
    it('uses custom connection name', () => {
      const iface: InterfaceConfig = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };

      const result = generateNmConnection({
        interfaceConfig: iface,
        addressing: defaultAddressing,
        connectionName: 'my-custom-connection',
      });

      expect(result).toContain('id=my-custom-connection');
    });

    it('uses default connection name', () => {
      const iface: InterfaceConfig = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };

      const result = generateNmConnection({
        interfaceConfig: iface,
        addressing: defaultAddressing,
      });

      expect(result).toContain('id=boardingpass-enrollment');
    });
  });
});

describe('getConnectionPath', () => {
  it('returns correct path with default name', () => {
    expect(getConnectionPath()).toBe(
      'NetworkManager/system-connections/boardingpass-enrollment.nmconnection'
    );
  });

  it('returns correct path with custom name', () => {
    expect(getConnectionPath('custom')).toBe(
      'NetworkManager/system-connections/custom.nmconnection'
    );
  });
});
