/**
 * Validation Utilities Tests
 */

import {
  isValidIPv4,
  isValidIPv6,
  isValidIPAddress,
  isValidHostname,
  isValidHost,
  isValidPort,
  isValidConnectionCode,
  isValidFingerprint,
  isValidMACAddress,
  isValidUUID,
  isValidDeviceName,
  sanitizeInput,
  validateAndSanitizeConnectionCode,
} from '../../../src/utils/validation';

describe('IP Address Validation', () => {
  describe('isValidIPv4', () => {
    it('should validate correct IPv4 addresses', () => {
      expect(isValidIPv4('192.168.1.1')).toBe(true);
      expect(isValidIPv4('10.0.0.1')).toBe(true);
      expect(isValidIPv4('172.16.0.1')).toBe(true);
      expect(isValidIPv4('255.255.255.255')).toBe(true);
      expect(isValidIPv4('0.0.0.0')).toBe(true);
    });

    it('should reject invalid IPv4 addresses', () => {
      expect(isValidIPv4('256.1.1.1')).toBe(false);
      expect(isValidIPv4('192.168.1')).toBe(false);
      expect(isValidIPv4('192.168.1.1.1')).toBe(false);
      expect(isValidIPv4('192.168.-1.1')).toBe(false);
      expect(isValidIPv4('not.an.ip.address')).toBe(false);
      expect(isValidIPv4('')).toBe(false);
    });
  });

  describe('isValidIPv6', () => {
    it('should validate correct IPv6 addresses', () => {
      expect(isValidIPv6('2001:0db8:85a3:0000:0000:8a2e:0370:7334')).toBe(true);
      expect(isValidIPv6('2001:db8:85a3::8a2e:370:7334')).toBe(true);
      expect(isValidIPv6('::1')).toBe(true);
      expect(isValidIPv6('fe80::1')).toBe(true);
    });

    it('should reject invalid IPv6 addresses', () => {
      expect(isValidIPv6('192.168.1.1')).toBe(false);
      expect(isValidIPv6('gggg::1')).toBe(false);
      expect(isValidIPv6('')).toBe(false);
    });
  });

  describe('isValidIPAddress', () => {
    it('should validate both IPv4 and IPv6', () => {
      expect(isValidIPAddress('192.168.1.1')).toBe(true);
      expect(isValidIPAddress('::1')).toBe(true);
      expect(isValidIPAddress('invalid')).toBe(false);
    });
  });
});

describe('Hostname Validation', () => {
  describe('isValidHostname', () => {
    it('should validate correct hostnames', () => {
      expect(isValidHostname('example.com')).toBe(true);
      expect(isValidHostname('sub.example.com')).toBe(true);
      expect(isValidHostname('device')).toBe(true);
      expect(isValidHostname('device-1')).toBe(true);
      expect(isValidHostname('my-device.local')).toBe(true);
    });

    it('should reject invalid hostnames', () => {
      expect(isValidHostname('-example.com')).toBe(false);
      expect(isValidHostname('example-.com')).toBe(false);
      expect(isValidHostname('example..com')).toBe(false);
      expect(isValidHostname('')).toBe(false);
      expect(isValidHostname('a'.repeat(64))).toBe(false); // Label too long
    });
  });

  describe('isValidHost', () => {
    it('should validate both IP addresses and hostnames', () => {
      expect(isValidHost('192.168.1.1')).toBe(true);
      expect(isValidHost('example.com')).toBe(true);
      expect(isValidHost('::1')).toBe(true);
      expect(isValidHost('invalid!')).toBe(false);
    });
  });
});

describe('Port Validation', () => {
  describe('isValidPort', () => {
    it('should validate correct ports', () => {
      expect(isValidPort(1)).toBe(true);
      expect(isValidPort(9443)).toBe(true);
      expect(isValidPort(65535)).toBe(true);
    });

    it('should reject invalid ports', () => {
      expect(isValidPort(0)).toBe(false);
      expect(isValidPort(65536)).toBe(false);
      expect(isValidPort(-1)).toBe(false);
      expect(isValidPort(1.5)).toBe(false);
    });
  });
});

describe('Connection Code Validation', () => {
  describe('isValidConnectionCode', () => {
    it('should validate correct connection codes', () => {
      expect(isValidConnectionCode('YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwCg==')).toBe(
        true
      );
      expect(isValidConnectionCode('A'.repeat(32))).toBe(true);
    });

    it('should reject invalid connection codes', () => {
      expect(isValidConnectionCode('')).toBe(false);
      expect(isValidConnectionCode('short')).toBe(false);
      expect(isValidConnectionCode('invalid!chars@here#')).toBe(false);
    });
  });
});

describe('Certificate Fingerprint Validation', () => {
  describe('isValidFingerprint', () => {
    it('should validate correct SHA-256 fingerprints', () => {
      expect(isValidFingerprint('a'.repeat(64))).toBe(true);
      expect(isValidFingerprint('0123456789abcdef'.repeat(4))).toBe(true);
    });

    it('should reject invalid fingerprints', () => {
      expect(isValidFingerprint('a'.repeat(63))).toBe(false);
      expect(isValidFingerprint('a'.repeat(65))).toBe(false);
      expect(isValidFingerprint('g'.repeat(64))).toBe(false);
      expect(isValidFingerprint('')).toBe(false);
    });
  });
});

describe('MAC Address Validation', () => {
  describe('isValidMACAddress', () => {
    it('should validate correct MAC addresses', () => {
      expect(isValidMACAddress('00:11:22:33:44:55')).toBe(true);
      expect(isValidMACAddress('AA:BB:CC:DD:EE:FF')).toBe(true);
      expect(isValidMACAddress('aa:bb:cc:dd:ee:ff')).toBe(true);
    });

    it('should reject invalid MAC addresses', () => {
      expect(isValidMACAddress('00:11:22:33:44')).toBe(false);
      expect(isValidMACAddress('00-11-22-33-44-55')).toBe(false);
      expect(isValidMACAddress('gg:hh:ii:jj:kk:ll')).toBe(false);
      expect(isValidMACAddress('')).toBe(false);
    });
  });
});

describe('UUID Validation', () => {
  describe('isValidUUID', () => {
    it('should validate correct UUIDs', () => {
      expect(isValidUUID('123e4567-e89b-12d3-a456-426614174000')).toBe(true);
      expect(isValidUUID('00000000-0000-0000-0000-000000000000')).toBe(true);
    });

    it('should reject invalid UUIDs', () => {
      expect(isValidUUID('123e4567-e89b-12d3-a456')).toBe(false);
      expect(isValidUUID('not-a-uuid')).toBe(false);
      expect(isValidUUID('')).toBe(false);
    });
  });
});

describe('Device Name Validation', () => {
  describe('isValidDeviceName', () => {
    it('should validate correct device names', () => {
      expect(isValidDeviceName('Device 1')).toBe(true);
      expect(isValidDeviceName('My Device')).toBe(true);
      expect(isValidDeviceName('a'.repeat(255))).toBe(true);
    });

    it('should reject invalid device names', () => {
      expect(isValidDeviceName('')).toBe(false);
      expect(isValidDeviceName('   ')).toBe(false);
      expect(isValidDeviceName('a'.repeat(256))).toBe(false);
    });
  });
});

describe('Input Sanitization', () => {
  describe('sanitizeInput', () => {
    it('should remove control characters', () => {
      expect(sanitizeInput('hello\x00world')).toBe('helloworld');
      expect(sanitizeInput('test\x1Fdata')).toBe('testdata');
    });

    it('should trim whitespace', () => {
      expect(sanitizeInput('  hello  ')).toBe('hello');
      expect(sanitizeInput('\n\ttest\n\t')).toBe('test');
    });

    it('should handle normal text', () => {
      expect(sanitizeInput('normal text')).toBe('normal text');
      expect(sanitizeInput('code-123')).toBe('code-123');
    });
  });

  describe('validateAndSanitizeConnectionCode', () => {
    it('should validate and sanitize valid codes', () => {
      const result = validateAndSanitizeConnectionCode(
        '  YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwCg==  '
      );
      expect(result.valid).toBe(true);
      expect(result.sanitized).toBe('YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwCg==');
      expect(result.error).toBeUndefined();
    });

    it('should reject empty codes', () => {
      const result = validateAndSanitizeConnectionCode('   ');
      expect(result.valid).toBe(false);
      expect(result.error).toBe('Connection code cannot be empty');
    });

    it('should reject invalid codes', () => {
      const result = validateAndSanitizeConnectionCode('short');
      expect(result.valid).toBe(false);
      expect(result.error).toBe('Invalid connection code format');
    });
  });
});
