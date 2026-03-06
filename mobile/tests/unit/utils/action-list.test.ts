import { buildActionList } from '../../../src/utils/action-list';
import { createInitialWizardState } from '../../../src/types/wizard';
import type { WizardState } from '../../../src/types/wizard';

function makeState(overrides: Partial<WizardState> = {}): WizardState {
  return { ...createInitialWizardState(), ...overrides };
}

describe('buildActionList', () => {
  it('generates correct actions for all-defaults (DHCP, no enrollment)', () => {
    const state = makeState({
      hostname: { hostname: 'device1', original: 'device1' },
      networkInterface: {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      },
      addressing: {
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
      },
      services: { ntp: { mode: 'automatic', servers: [] }, proxy: null },
      enrollment: { insights: null, flightControl: null },
    });

    const actions = buildActionList(state, 'immediate');

    expect(actions.map(a => a.id)).toEqual([
      'hostname',
      'interface',
      'ipv4-config',
      'connectivity-check',
      'dns-check',
      'ntp-config',
      'clock-sync',
    ]);

    // Hostname unchanged → info only
    expect(actions[0].description).toContain('Keep hostname');
    expect(actions[0].infoOnly).toBe(true);

    // DHCP
    expect(actions[2].description).toBe('Assign IPv4 address via DHCP');

    // Automatic NTP
    expect(actions[5].description).toBe('Use automatic time servers');

    // Clock sync in immediate mode → not info only
    expect(actions[6].infoOnly).toBe(false);
  });

  it('generates correct actions for static IPv4 with manual DNS', () => {
    const state = makeState({
      hostname: { hostname: 'myhost', original: 'device1' },
      networkInterface: {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      },
      addressing: {
        ipv4: {
          method: 'static',
          address: '192.168.1.100',
          subnetMask: '255.255.255.0',
          gateway: '192.168.1.1',
          dnsAuto: false,
          dnsPrimary: '8.8.8.8',
          dnsSecondary: '8.8.4.4',
        },
        ipv6: {
          method: 'disabled',
          address: null,
          gateway: null,
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
      },
    });

    const actions = buildActionList(state, 'immediate');

    // Hostname changed
    const hostnameAction = actions.find(a => a.id === 'hostname')!;
    expect(hostnameAction.description).toBe('Set hostname to "myhost"');
    expect(hostnameAction.infoOnly).toBe(false);

    // Static IPv4
    const ipv4Action = actions.find(a => a.id === 'ipv4-config')!;
    expect(ipv4Action.description).toContain('192.168.1.100');
    expect(ipv4Action.description).toContain('gateway 192.168.1.1');

    // Manual DNS
    const dnsAction = actions.find(a => a.id === 'dns-config')!;
    expect(dnsAction.description).toContain('8.8.8.8');
    expect(dnsAction.description).toContain('8.8.4.4');
  });

  it('includes WiFi action for WiFi interface', () => {
    const state = makeState({
      networkInterface: {
        interfaceName: 'wlan0',
        interfaceType: 'wifi',
        vlanId: null,
        wifi: {
          ssid: 'MyNetwork',
          bssid: '00:11:22:33:44:55',
          security: 'WPA2',
          password: 'secret',
        },
      },
    });

    const actions = buildActionList(state, 'immediate');

    const wifiAction = actions.find(a => a.id === 'wifi-connect')!;
    expect(wifiAction).toBeDefined();
    expect(wifiAction.description).toContain('MyNetwork');
  });

  it('includes IPv6 DHCP action when IPv6 enabled', () => {
    const state = makeState({
      addressing: {
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
          method: 'dhcp',
          address: null,
          gateway: null,
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
      },
    });

    const actions = buildActionList(state, 'immediate');
    const ipv6Action = actions.find(a => a.id === 'ipv6-config')!;
    expect(ipv6Action).toBeDefined();
    expect(ipv6Action.description).toBe('Assign IPv6 address via DHCP');
  });

  it('includes static IPv6 action', () => {
    const state = makeState({
      addressing: {
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
          method: 'static',
          address: 'fd00::1/64',
          gateway: 'fd00::ffff',
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
      },
    });

    const actions = buildActionList(state, 'immediate');
    const ipv6Action = actions.find(a => a.id === 'ipv6-config')!;
    expect(ipv6Action.description).toContain('fd00::1/64');
  });

  it('omits IPv6 action when disabled', () => {
    const state = makeState({
      addressing: {
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
      },
    });

    const actions = buildActionList(state, 'immediate');
    expect(actions.find(a => a.id === 'ipv6-config')).toBeUndefined();
  });

  it('includes Insights enrollment action', () => {
    const state = makeState({
      enrollment: {
        insights: {
          endpoint: 'https://cert-api.access.redhat.com',
          orgId: '123',
          activationKey: 'key1',
        },
        flightControl: null,
      },
    });

    const actions = buildActionList(state, 'immediate');
    expect(actions.find(a => a.id === 'enroll-insights')).toBeDefined();
    expect(actions.find(a => a.id === 'enroll-flightctl')).toBeUndefined();
  });

  it('includes Flight Control enrollment action', () => {
    const state = makeState({
      enrollment: {
        insights: null,
        flightControl: { endpoint: 'https://fc.example.com', username: 'user', password: 'pass' },
      },
    });

    const actions = buildActionList(state, 'immediate');
    expect(actions.find(a => a.id === 'enroll-flightctl')).toBeDefined();
    expect(actions.find(a => a.id === 'enroll-insights')).toBeUndefined();
  });

  it('includes both enrollment actions when both configured', () => {
    const state = makeState({
      enrollment: {
        insights: {
          endpoint: 'https://cert-api.access.redhat.com',
          orgId: '123',
          activationKey: 'key1',
        },
        flightControl: { endpoint: 'https://fc.example.com', username: 'user', password: 'pass' },
      },
    });

    const actions = buildActionList(state, 'immediate');
    expect(actions.find(a => a.id === 'enroll-insights')).toBeDefined();
    expect(actions.find(a => a.id === 'enroll-flightctl')).toBeDefined();
  });

  it('includes manual NTP servers in description', () => {
    const state = makeState({
      services: {
        ntp: { mode: 'manual', servers: ['ntp1.example.com', 'ntp2.example.com'] },
        proxy: null,
      },
    });

    const actions = buildActionList(state, 'immediate');
    const ntpAction = actions.find(a => a.id === 'ntp-config')!;
    expect(ntpAction.description).toContain('ntp1.example.com');
    expect(ntpAction.description).toContain('ntp2.example.com');
  });

  describe('immediate vs deferred mode', () => {
    it('includes check actions in immediate mode', () => {
      const state = makeState();
      const actions = buildActionList(state, 'immediate');
      expect(actions.find(a => a.id === 'connectivity-check')).toBeDefined();
      expect(actions.find(a => a.id === 'dns-check')).toBeDefined();
    });

    it('omits check actions in deferred mode', () => {
      const state = makeState();
      const actions = buildActionList(state, 'deferred');
      expect(actions.find(a => a.id === 'connectivity-check')).toBeUndefined();
      expect(actions.find(a => a.id === 'dns-check')).toBeUndefined();
    });

    it('clock sync is executable in immediate mode', () => {
      const state = makeState();
      const actions = buildActionList(state, 'immediate');
      const clockAction = actions.find(a => a.id === 'clock-sync')!;
      expect(clockAction.infoOnly).toBe(false);
      expect(clockAction.description).toBe('Wait for clock to be synchronised');
    });

    it('clock sync is info-only in deferred mode', () => {
      const state = makeState();
      const actions = buildActionList(state, 'deferred');
      const clockAction = actions.find(a => a.id === 'clock-sync')!;
      expect(clockAction.infoOnly).toBe(true);
      expect(clockAction.description).toContain('after reboot');
    });
  });
});
