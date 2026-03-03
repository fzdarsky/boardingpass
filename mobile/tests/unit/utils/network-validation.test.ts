/**
 * Network Validation Utilities Tests
 *
 * Tests for RFC 1123 hostname validation, IPv4/IPv6 validation,
 * subnet mask, gateway-in-subnet, port, HTTPS URL, and NTP server.
 */

import {
  validateHostname,
  validateIPv4,
  validateIPv6,
  validateSubnetMask,
  validateGatewayInSubnet,
  validatePort,
  validateHttpsUrl,
  validateNtpServer,
} from '../../../src/utils/network-validation';

describe('validateHostname', () => {
  it('accepts valid single-label hostnames', () => {
    expect(validateHostname('myhost')).toBeNull();
    expect(validateHostname('host1')).toBeNull();
    expect(validateHostname('my-host')).toBeNull();
    expect(validateHostname('A')).toBeNull();
  });

  it('accepts valid multi-label hostnames', () => {
    expect(validateHostname('host.example.com')).toBeNull();
    expect(validateHostname('my-device.local')).toBeNull();
  });

  it('rejects empty hostname', () => {
    expect(validateHostname('')).toBe('Hostname is required');
  });

  it('rejects hostname longer than 253 characters', () => {
    const long = 'a'.repeat(254);
    expect(validateHostname(long)).toBe('Hostname must be at most 253 characters');
  });

  it('rejects labels longer than 63 characters', () => {
    const long = 'a'.repeat(64);
    expect(validateHostname(long)).toBe('Each hostname label must be at most 63 characters');
  });

  it('rejects leading hyphens', () => {
    expect(validateHostname('-myhost')).not.toBeNull();
  });

  it('rejects trailing hyphens', () => {
    expect(validateHostname('myhost-')).not.toBeNull();
  });

  it('rejects underscores', () => {
    expect(validateHostname('my_host')).not.toBeNull();
  });

  it('rejects empty labels (double dots)', () => {
    expect(validateHostname('host..com')).not.toBeNull();
  });
});

describe('validateIPv4', () => {
  it('accepts valid IPv4 addresses', () => {
    expect(validateIPv4('192.168.1.1')).toBeNull();
    expect(validateIPv4('0.0.0.0')).toBeNull();
    expect(validateIPv4('255.255.255.255')).toBeNull();
    expect(validateIPv4('10.0.0.1')).toBeNull();
  });

  it('rejects empty string', () => {
    expect(validateIPv4('')).toBe('IPv4 address is required');
  });

  it('rejects wrong number of octets', () => {
    expect(validateIPv4('192.168.1')).not.toBeNull();
    expect(validateIPv4('192.168.1.1.1')).not.toBeNull();
  });

  it('rejects out-of-range octets', () => {
    expect(validateIPv4('256.0.0.1')).not.toBeNull();
    expect(validateIPv4('192.168.1.-1')).not.toBeNull();
  });

  it('rejects leading zeros', () => {
    expect(validateIPv4('192.168.01.1')).not.toBeNull();
  });

  it('rejects non-numeric octets', () => {
    expect(validateIPv4('abc.def.ghi.jkl')).not.toBeNull();
  });
});

describe('validateIPv6', () => {
  it('accepts valid full IPv6 addresses', () => {
    expect(validateIPv6('2001:0db8:0000:0000:0000:0000:0000:0001')).toBeNull();
    expect(validateIPv6('fe80:0000:0000:0000:0000:0000:0000:0001')).toBeNull();
  });

  it('accepts abbreviated IPv6 addresses', () => {
    expect(validateIPv6('2001:db8::1')).toBeNull();
    expect(validateIPv6('::1')).toBeNull();
    expect(validateIPv6('fe80::')).toBeNull();
    expect(validateIPv6('::')).toBeNull();
  });

  it('accepts IPv6 with prefix when allowed', () => {
    expect(validateIPv6('2001:db8::1/64', true)).toBeNull();
    expect(validateIPv6('::1/128', true)).toBeNull();
  });

  it('rejects IPv6 with prefix when not allowed', () => {
    expect(validateIPv6('2001:db8::1/64')).not.toBeNull();
  });

  it('rejects invalid prefix lengths', () => {
    expect(validateIPv6('2001:db8::1/129', true)).not.toBeNull();
    expect(validateIPv6('2001:db8::1/-1', true)).not.toBeNull();
  });

  it('rejects empty string', () => {
    expect(validateIPv6('')).toBe('IPv6 address is required');
  });

  it('rejects multiple :: occurrences', () => {
    expect(validateIPv6('2001::db8::1')).not.toBeNull();
  });

  it('rejects invalid groups', () => {
    expect(validateIPv6('2001:db8:xxxx::1')).not.toBeNull();
    expect(validateIPv6('2001:db8:12345::1')).not.toBeNull();
  });
});

describe('validateSubnetMask', () => {
  it('accepts valid dotted-decimal masks', () => {
    expect(validateSubnetMask('255.255.255.0')).toBeNull();
    expect(validateSubnetMask('255.255.0.0')).toBeNull();
    expect(validateSubnetMask('255.0.0.0')).toBeNull();
    expect(validateSubnetMask('255.255.255.255')).toBeNull();
    expect(validateSubnetMask('0.0.0.0')).toBeNull();
  });

  it('accepts CIDR prefix notation', () => {
    expect(validateSubnetMask('24')).toBeNull();
    expect(validateSubnetMask('16')).toBeNull();
    expect(validateSubnetMask('0')).toBeNull();
    expect(validateSubnetMask('32')).toBeNull();
  });

  it('rejects empty string', () => {
    expect(validateSubnetMask('')).toBe('Subnet mask is required');
  });

  it('rejects non-contiguous masks', () => {
    expect(validateSubnetMask('255.0.255.0')).not.toBeNull();
  });

  it('rejects invalid octets', () => {
    expect(validateSubnetMask('256.255.255.0')).not.toBeNull();
  });
});

describe('validateGatewayInSubnet', () => {
  it('accepts gateway in same subnet', () => {
    expect(validateGatewayInSubnet('192.168.1.1', '192.168.1.100', '255.255.255.0')).toBeNull();
    expect(validateGatewayInSubnet('10.0.0.1', '10.0.0.100', '255.255.0.0')).toBeNull();
  });

  it('rejects gateway in different subnet', () => {
    expect(validateGatewayInSubnet('192.168.2.1', '192.168.1.100', '255.255.255.0')).not.toBeNull();
    expect(validateGatewayInSubnet('10.1.0.1', '10.0.0.100', '255.255.255.0')).not.toBeNull();
  });

  it('returns error for invalid gateway IP', () => {
    expect(validateGatewayInSubnet('invalid', '192.168.1.100', '255.255.255.0')).not.toBeNull();
  });

  it('skips validation when address is invalid', () => {
    expect(validateGatewayInSubnet('192.168.1.1', 'invalid', '255.255.255.0')).toBeNull();
  });
});

describe('validatePort', () => {
  it('accepts valid ports', () => {
    expect(validatePort(1)).toBeNull();
    expect(validatePort(80)).toBeNull();
    expect(validatePort(443)).toBeNull();
    expect(validatePort(8443)).toBeNull();
    expect(validatePort(65535)).toBeNull();
  });

  it('accepts string ports', () => {
    expect(validatePort('8080')).toBeNull();
    expect(validatePort('443')).toBeNull();
  });

  it('rejects zero', () => {
    expect(validatePort(0)).not.toBeNull();
  });

  it('rejects negative numbers', () => {
    expect(validatePort(-1)).not.toBeNull();
  });

  it('rejects out-of-range', () => {
    expect(validatePort(65536)).not.toBeNull();
  });

  it('rejects non-numeric strings', () => {
    expect(validatePort('abc')).not.toBeNull();
  });
});

describe('validateHttpsUrl', () => {
  it('accepts valid HTTPS URLs', () => {
    expect(validateHttpsUrl('https://example.com')).toBeNull();
    expect(validateHttpsUrl('https://api.example.com/v1')).toBeNull();
    expect(validateHttpsUrl('https://cert-api.access.redhat.com')).toBeNull();
  });

  it('rejects empty string', () => {
    expect(validateHttpsUrl('')).toBe('URL is required');
  });

  it('rejects HTTP URLs', () => {
    expect(validateHttpsUrl('http://example.com')).toBe('URL must use HTTPS');
  });

  it('rejects invalid URLs', () => {
    expect(validateHttpsUrl('not-a-url')).toBe('Invalid URL format');
  });
});

describe('validateNtpServer', () => {
  it('accepts valid hostnames', () => {
    expect(validateNtpServer('pool.ntp.org')).toBeNull();
    expect(validateNtpServer('time.google.com')).toBeNull();
  });

  it('accepts valid IPv4 addresses', () => {
    expect(validateNtpServer('192.168.1.1')).toBeNull();
    expect(validateNtpServer('10.0.0.1')).toBeNull();
  });

  it('rejects empty string', () => {
    expect(validateNtpServer('')).not.toBeNull();
  });

  it('rejects invalid values', () => {
    expect(validateNtpServer('not valid!')).not.toBeNull();
  });
});
