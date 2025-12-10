/**
 * Crypto Utilities Tests
 */

import {
  sha256,
  isValidSHA256,
  formatFingerprint,
  compareFingerprints,
  truncateFingerprint,
} from '../../../src/utils/crypto';

// Mock expo-crypto
jest.mock('expo-crypto', () => ({
  digestStringAsync: jest.fn((_algorithm, _input) => {
    // Simple mock: return a fake hash based on input
    return Promise.resolve('a'.repeat(64)); // 64-char hex string
  }),
  CryptoDigestAlgorithm: {
    SHA256: 'SHA256',
  },
  getRandomBytesAsync: jest.fn(length => {
    const bytes = new Uint8Array(length);
    for (let i = 0; i < length; i++) {
      bytes[i] = i % 256;
    }
    return Promise.resolve(bytes);
  }),
}));

describe('SHA-256 Hashing', () => {
  describe('sha256', () => {
    it('should compute SHA-256 hash', async () => {
      const hash = await sha256('test string');
      expect(hash).toBe('a'.repeat(64));
      expect(hash.length).toBe(64);
    });

    it('should return lowercase hex', async () => {
      const hash = await sha256('test');
      expect(hash).toMatch(/^[a-f0-9]{64}$/);
    });
  });

  describe('isValidSHA256', () => {
    it('should validate correct SHA-256 hashes', () => {
      expect(isValidSHA256('a'.repeat(64))).toBe(true);
      expect(isValidSHA256('0123456789abcdef'.repeat(4))).toBe(true);
      expect(isValidSHA256('ABCDEF0123456789'.repeat(4))).toBe(true); // Uppercase ok
    });

    it('should reject invalid SHA-256 hashes', () => {
      expect(isValidSHA256('a'.repeat(63))).toBe(false); // Too short
      expect(isValidSHA256('a'.repeat(65))).toBe(false); // Too long
      expect(isValidSHA256('g'.repeat(64))).toBe(false); // Invalid hex
      expect(isValidSHA256('')).toBe(false);
      expect(isValidSHA256('not a hash')).toBe(false);
    });
  });
});

describe('Fingerprint Formatting', () => {
  describe('formatFingerprint', () => {
    it('should format fingerprint with colons', () => {
      const fp = 'a'.repeat(64);
      const formatted = formatFingerprint(fp);
      expect(formatted).toBe('aa:'.repeat(31) + 'aa');
      expect(formatted.split(':').length).toBe(32); // 32 groups
    });

    it('should handle already formatted fingerprints', () => {
      const fp = '0123456789abcdef'.repeat(4);
      const formatted = formatFingerprint(fp);
      expect(formatted).toMatch(/^([0-9a-f]{2}:){31}[0-9a-f]{2}$/);
    });

    it('should return original if invalid format', () => {
      const invalid = 'invalid';
      expect(formatFingerprint(invalid)).toBe('invalid');
    });

    it('should convert to lowercase', () => {
      const fp = 'A'.repeat(64);
      const formatted = formatFingerprint(fp);
      expect(formatted).toMatch(/^[a-f0-9:]+$/);
      expect(formatted.toUpperCase()).not.toBe(formatted);
    });
  });

  describe('compareFingerprints', () => {
    it('should compare fingerprints case-insensitively', () => {
      const fp1 = 'a'.repeat(64);
      const fp2 = 'A'.repeat(64);
      expect(compareFingerprints(fp1, fp2)).toBe(true);
    });

    it('should detect different fingerprints', () => {
      const fp1 = 'a'.repeat(64);
      const fp2 = 'b'.repeat(64);
      expect(compareFingerprints(fp1, fp2)).toBe(false);
    });

    it('should handle mixed case', () => {
      const fp1 = 'aBcDeF0123456789'.repeat(4);
      const fp2 = 'ABCDEF0123456789'.repeat(4);
      expect(compareFingerprints(fp1, fp2)).toBe(true);
    });
  });

  describe('truncateFingerprint', () => {
    it('should truncate long fingerprints', () => {
      const fp = '0123456789abcdef'.repeat(4);
      const truncated = truncateFingerprint(fp, 6);
      expect(truncated).toMatch(/^([0-9a-f]{2}:){5}[0-9a-f]{2}\.\.\.([0-9a-f]{2}:){5}[0-9a-f]{2}$/);
      expect(truncated.length).toBeLessThan(formatFingerprint(fp).length);
    });

    it('should not truncate short fingerprints', () => {
      const fp = 'ab'.repeat(32);
      const truncated = truncateFingerprint(fp, 32);
      // Should return full formatted fingerprint
      expect(truncated).toBe(formatFingerprint(fp));
    });

    it('should handle custom length', () => {
      const fp = '0123456789abcdef'.repeat(4);
      const truncated = truncateFingerprint(fp, 3);
      expect(truncated).toMatch(/^[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}\.\.\./);
    });
  });
});
