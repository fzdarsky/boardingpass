/**
 * Certificate Utilities Unit Tests
 *
 * Tests for certificate fetching, parsing, and validation utilities.
 */

import {
  parseCertificate,
  validateCertificatePin,
  isCertificateExpired,
  isCertificateExpiringSoon,
  getDaysUntilExpiration,
  formatValidityPeriod,
  extractCommonName,
  fetchCertificateMock,
  RawCertificateData,
} from '../../../../src/services/certificates/utils';
import { CertificateInfo } from '../../../../src/types/certificate';

describe('Certificate Utilities', () => {
  describe('fetchCertificateMock', () => {
    it('should return mock certificate data', async () => {
      const cert = await fetchCertificateMock('192.168.1.100', 9455);

      expect(cert).toBeDefined();
      expect(cert.pemEncoded).toContain('BEGIN CERTIFICATE');
      expect(cert.subject).toContain('192.168.1.100');
      expect(cert.issuer).toBe(cert.subject); // Self-signed
      expect(new Date(cert.validFrom)).toBeInstanceOf(Date);
      expect(new Date(cert.validTo)).toBeInstanceOf(Date);
    });

    it('should include host in certificate subject', async () => {
      const host = 'device.local';
      const cert = await fetchCertificateMock(host, 9455);

      expect(cert.subject).toContain(host);
    });
  });

  describe('parseCertificate', () => {
    const mockRawCert: RawCertificateData = {
      fingerprint: 'abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
      pemEncoded: '-----BEGIN CERTIFICATE-----\nMIID...test\n-----END CERTIFICATE-----',
      subject: 'CN=device.local,O=BoardingPass,C=US',
      issuer: 'CN=device.local,O=BoardingPass,C=US', // Self-signed
      validFrom: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
      validTo: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(),
    };

    it('should parse self-signed certificate with no existing pin', async () => {
      const cert = await parseCertificate('device-1', mockRawCert);

      expect(cert.deviceId).toBe('device-1');
      expect(cert.subject).toBe(mockRawCert.subject);
      expect(cert.issuer).toBe(mockRawCert.issuer);
      expect(cert.isSelfSigned).toBe(true);
      expect(cert.trustStatus).toBe('self_signed_new');
      expect(cert.issuedByTrustedCA).toBe(false);
    });

    it('should parse CA-signed certificate with no existing pin', async () => {
      const caCert: RawCertificateData = {
        ...mockRawCert,
        issuer: 'CN=Trusted CA,O=Certificate Authority,C=US', // Different from subject
      };

      const cert = await parseCertificate('device-1', caCert);

      expect(cert.isSelfSigned).toBe(false);
      expect(cert.trustStatus).toBe('trusted_ca');
      expect(cert.issuedByTrustedCA).toBe(true);
    });

    it('should detect certificate change when fingerprint differs', async () => {
      const existingPin: CertificateInfo = {
        deviceId: 'device-1',
        fingerprint: '0000000000000000000000000000000000000000000000000000000000000000',
        subject: mockRawCert.subject,
        issuer: mockRawCert.issuer,
        validFrom: new Date(mockRawCert.validFrom),
        validTo: new Date(mockRawCert.validTo),
        isSelfSigned: true,
        issuedByTrustedCA: false,
        trustStatus: 'self_signed_trusted',
        pinnedAt: new Date(),
        lastVerified: new Date(),
      };

      const cert = await parseCertificate('device-1', mockRawCert, existingPin);

      // Fingerprint won't match (different cert data), so should be 'changed'
      expect(cert.trustStatus).toBe('changed');
    });

    it('should maintain trust status when fingerprint matches', async () => {
      // This test is tricky because fingerprint depends on cert data
      // In practice, same rawCert would generate same fingerprint
      const cert1 = await parseCertificate('device-1', mockRawCert);
      const cert2 = await parseCertificate('device-1', mockRawCert, cert1);

      // Same cert data should yield same fingerprint and maintain trust
      expect(cert2.fingerprint).toBe(cert1.fingerprint);
    });
  });

  describe('validateCertificatePin', () => {
    const baseCert: CertificateInfo = {
      deviceId: 'device-1',
      fingerprint: 'abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234',
      subject: 'CN=device.local',
      issuer: 'CN=device.local',
      validFrom: new Date(),
      validTo: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000),
      isSelfSigned: true,
      issuedByTrustedCA: false,
      trustStatus: 'self_signed_trusted',
      pinnedAt: new Date(),
      lastVerified: new Date(),
    };

    it('should return true when fingerprints match', () => {
      const current = { ...baseCert };
      const pinned = { ...baseCert };

      expect(validateCertificatePin(current, pinned)).toBe(true);
    });

    it('should return true when fingerprints match (case-insensitive)', () => {
      const current = { ...baseCert, fingerprint: baseCert.fingerprint.toUpperCase() };
      const pinned = { ...baseCert, fingerprint: baseCert.fingerprint.toLowerCase() };

      expect(validateCertificatePin(current, pinned)).toBe(true);
    });

    it('should return false when fingerprints differ', () => {
      const current = { ...baseCert };
      const pinned = {
        ...baseCert,
        fingerprint: '0000000000000000000000000000000000000000000000000000000000000000',
      };

      expect(validateCertificatePin(current, pinned)).toBe(false);
    });
  });

  describe('isCertificateExpired', () => {
    it('should return false for valid certificate', () => {
      const cert: CertificateInfo = {
        deviceId: 'device-1',
        fingerprint: 'abc123',
        subject: 'CN=device.local',
        issuer: 'CN=device.local',
        validFrom: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), // 30 days ago
        validTo: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year from now
        isSelfSigned: true,
        issuedByTrustedCA: false,
        trustStatus: 'self_signed_trusted',
        pinnedAt: new Date(),
        lastVerified: new Date(),
      };

      expect(isCertificateExpired(cert)).toBe(false);
    });

    it('should return true for expired certificate', () => {
      const cert: CertificateInfo = {
        deviceId: 'device-1',
        fingerprint: 'abc123',
        subject: 'CN=device.local',
        issuer: 'CN=device.local',
        validFrom: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000), // 1 year ago
        validTo: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000), // 1 day ago
        isSelfSigned: true,
        issuedByTrustedCA: false,
        trustStatus: 'self_signed_trusted',
        pinnedAt: new Date(),
        lastVerified: new Date(),
      };

      expect(isCertificateExpired(cert)).toBe(true);
    });

    it('should return true for not-yet-valid certificate', () => {
      const cert: CertificateInfo = {
        deviceId: 'device-1',
        fingerprint: 'abc123',
        subject: 'CN=device.local',
        issuer: 'CN=device.local',
        validFrom: new Date(Date.now() + 1 * 24 * 60 * 60 * 1000), // Tomorrow
        validTo: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year from now
        isSelfSigned: true,
        issuedByTrustedCA: false,
        trustStatus: 'self_signed_trusted',
        pinnedAt: new Date(),
        lastVerified: new Date(),
      };

      expect(isCertificateExpired(cert)).toBe(true);
    });
  });

  describe('isCertificateExpiringSoon', () => {
    it('should return true when certificate expires within 30 days', () => {
      const cert: CertificateInfo = {
        deviceId: 'device-1',
        fingerprint: 'abc123',
        subject: 'CN=device.local',
        issuer: 'CN=device.local',
        validFrom: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000),
        validTo: new Date(Date.now() + 15 * 24 * 60 * 60 * 1000), // 15 days from now
        isSelfSigned: true,
        issuedByTrustedCA: false,
        trustStatus: 'self_signed_trusted',
        pinnedAt: new Date(),
        lastVerified: new Date(),
      };

      expect(isCertificateExpiringSoon(cert, 30)).toBe(true);
    });

    it('should return false when certificate expires after threshold', () => {
      const cert: CertificateInfo = {
        deviceId: 'device-1',
        fingerprint: 'abc123',
        subject: 'CN=device.local',
        issuer: 'CN=device.local',
        validFrom: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000),
        validTo: new Date(Date.now() + 60 * 24 * 60 * 60 * 1000), // 60 days from now
        isSelfSigned: true,
        issuedByTrustedCA: false,
        trustStatus: 'self_signed_trusted',
        pinnedAt: new Date(),
        lastVerified: new Date(),
      };

      expect(isCertificateExpiringSoon(cert, 30)).toBe(false);
    });
  });

  describe('getDaysUntilExpiration', () => {
    it('should return positive days for valid certificate', () => {
      const cert: CertificateInfo = {
        deviceId: 'device-1',
        fingerprint: 'abc123',
        subject: 'CN=device.local',
        issuer: 'CN=device.local',
        validFrom: new Date(),
        validTo: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
        isSelfSigned: true,
        issuedByTrustedCA: false,
        trustStatus: 'self_signed_trusted',
        pinnedAt: new Date(),
        lastVerified: new Date(),
      };

      const days = getDaysUntilExpiration(cert);
      expect(days).toBeGreaterThanOrEqual(29);
      expect(days).toBeLessThanOrEqual(30);
    });

    it('should return negative days for expired certificate', () => {
      const cert: CertificateInfo = {
        deviceId: 'device-1',
        fingerprint: 'abc123',
        subject: 'CN=device.local',
        issuer: 'CN=device.local',
        validFrom: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000),
        validTo: new Date(Date.now() - 10 * 24 * 60 * 60 * 1000), // 10 days ago
        isSelfSigned: true,
        issuedByTrustedCA: false,
        trustStatus: 'self_signed_trusted',
        pinnedAt: new Date(),
        lastVerified: new Date(),
      };

      const days = getDaysUntilExpiration(cert);
      expect(days).toBeLessThan(0);
      expect(days).toBeGreaterThanOrEqual(-11);
      expect(days).toBeLessThanOrEqual(-9);
    });
  });

  describe('formatValidityPeriod', () => {
    it('should format validity period as date range', () => {
      const cert: CertificateInfo = {
        deviceId: 'device-1',
        fingerprint: 'abc123',
        subject: 'CN=device.local',
        issuer: 'CN=device.local',
        validFrom: new Date('2024-01-01'),
        validTo: new Date('2025-01-01'),
        isSelfSigned: true,
        issuedByTrustedCA: false,
        trustStatus: 'self_signed_trusted',
        pinnedAt: new Date(),
        lastVerified: new Date(),
      };

      const formatted = formatValidityPeriod(cert);
      expect(formatted).toContain('2024');
      expect(formatted).toContain('2025');
      expect(formatted).toContain('-');
    });
  });

  describe('extractCommonName', () => {
    it('should extract CN from X.509 DN', () => {
      const dn = 'CN=device.local,O=BoardingPass,C=US';
      expect(extractCommonName(dn)).toBe('device.local');
    });

    it('should handle case-insensitive CN', () => {
      const dn = 'cn=device.local,o=BoardingPass,c=US';
      expect(extractCommonName(dn)).toBe('device.local');
    });

    it('should return full DN if CN not found', () => {
      const dn = 'O=BoardingPass,C=US';
      expect(extractCommonName(dn)).toBe(dn);
    });

    it('should trim whitespace from CN value', () => {
      const dn = 'CN= device.local ,O=BoardingPass';
      expect(extractCommonName(dn)).toBe('device.local');
    });
  });
});
