/**
 * Certificate Utilities
 *
 * Utilities for fetching, parsing, and validating TLS certificates.
 * Implements certificate pinning for self-signed certificates.
 */

import { computeCertificateFingerprint, isValidSHA256 } from '../../utils/crypto';
import { CertificateInfo, TrustStatus } from '../../types/certificate';
import { fetchServerCertificate } from '../../../modules/certificate-pinning';

/**
 * Raw certificate data from TLS handshake
 */
export interface RawCertificateData {
  pemEncoded: string; // PEM-encoded certificate
  subject: string; // X.509 subject DN
  issuer: string; // X.509 issuer DN
  validFrom: string; // ISO date string
  validTo: string; // ISO date string
}

/**
 * Fetch certificate from HTTPS endpoint via native TLS handshake.
 *
 * Uses the CertificatePinning native module to perform a TLS handshake,
 * capture the server's certificate (even self-signed), and return its metadata.
 *
 * @param host - Server hostname or IP
 * @param port - Server port (default 8443)
 * @returns Raw certificate data
 * @throws Error if certificate cannot be fetched
 */
export async function fetchCertificate(
  host: string,
  port: number = 8443
): Promise<RawCertificateData> {
  const cert = await fetchServerCertificate(host, port);

  return {
    pemEncoded: cert.pemEncoded,
    subject: cert.subject,
    issuer: cert.issuer,
    validFrom: cert.validFrom || new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    validTo: cert.validTo || new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(),
  };
}

/**
 * Parse raw certificate data into CertificateInfo
 *
 * @param deviceId - Device ID to associate with certificate
 * @param rawCert - Raw certificate data from TLS handshake
 * @param existingPin - Existing pinned certificate (if any)
 * @returns Parsed and validated certificate info
 */
export async function parseCertificate(
  deviceId: string,
  rawCert: RawCertificateData,
  existingPin?: CertificateInfo
): Promise<CertificateInfo> {
  // Compute fingerprint
  const fingerprint = await computeCertificateFingerprint(rawCert.pemEncoded);

  // Validate fingerprint format
  if (!isValidSHA256(fingerprint)) {
    throw new Error(`Invalid certificate fingerprint: ${fingerprint}`);
  }

  // Parse dates
  const validFrom = new Date(rawCert.validFrom);
  const validTo = new Date(rawCert.validTo);

  // Determine if self-signed (subject === issuer)
  const isSelfSigned = rawCert.subject === rawCert.issuer;

  // Determine trust status
  const trustStatus = determineTrustStatus(fingerprint, isSelfSigned, existingPin);

  // Check if issued by trusted CA
  // Note: In production, this would check against OS trust store
  // For now, assume any non-self-signed cert is from trusted CA
  const issuedByTrustedCA = !isSelfSigned;

  const now = new Date();

  const certInfo: CertificateInfo = {
    deviceId,
    fingerprint,
    subject: rawCert.subject,
    issuer: rawCert.issuer,
    validFrom,
    validTo,
    isSelfSigned,
    issuedByTrustedCA,
    trustStatus,
    pinnedAt: existingPin?.pinnedAt || now,
    lastVerified: now,
    userConfirmedAt: existingPin?.userConfirmedAt,
  };

  return certInfo;
}

/**
 * Determine trust status based on certificate properties and existing pin
 */
function determineTrustStatus(
  fingerprint: string,
  isSelfSigned: boolean,
  existingPin?: CertificateInfo
): TrustStatus {
  // If no existing pin, this is first connection
  if (!existingPin) {
    if (isSelfSigned) {
      return 'self_signed_new'; // New self-signed cert needs user confirmation
    } else {
      return 'trusted_ca'; // CA-signed certs are trusted immediately
    }
  }

  // Check if fingerprint matches existing pin
  const fingerprintMatches = fingerprint.toLowerCase() === existingPin.fingerprint.toLowerCase();

  if (!fingerprintMatches) {
    return 'changed'; // Certificate changed - alert user!
  }

  // Fingerprint matches - maintain existing trust status
  if (existingPin.trustStatus === 'self_signed_trusted') {
    return 'self_signed_trusted';
  } else if (existingPin.trustStatus === 'trusted_ca') {
    return 'trusted_ca';
  }

  // Default fallback
  return isSelfSigned ? 'self_signed_new' : 'trusted_ca';
}

/**
 * Validate certificate against pinned certificate
 *
 * @param current - Current certificate from connection
 * @param pinned - Pinned certificate from storage
 * @returns true if certificates match, false otherwise
 */
export function validateCertificatePin(current: CertificateInfo, pinned: CertificateInfo): boolean {
  // Compare fingerprints (case-insensitive)
  return current.fingerprint.toLowerCase() === pinned.fingerprint.toLowerCase();
}

/**
 * Check if certificate is expired or not yet valid
 */
export function isCertificateExpired(cert: CertificateInfo): boolean {
  const now = new Date();
  return now < cert.validFrom || now > cert.validTo;
}

/**
 * Check if certificate expires soon (within days)
 */
export function isCertificateExpiringSoon(cert: CertificateInfo, days: number = 30): boolean {
  const now = new Date();
  const daysUntilExpiry = Math.floor(
    (cert.validTo.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  );
  return daysUntilExpiry >= 0 && daysUntilExpiry <= days;
}

/**
 * Get days until certificate expiration
 */
export function getDaysUntilExpiration(cert: CertificateInfo): number {
  const now = new Date();
  const daysUntilExpiry = Math.floor(
    (cert.validTo.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  );
  return daysUntilExpiry;
}

/**
 * Format certificate validity period for display
 */
export function formatValidityPeriod(cert: CertificateInfo): string {
  const from = cert.validFrom.toLocaleDateString();
  const to = cert.validTo.toLocaleDateString();
  return `${from} - ${to}`;
}

/**
 * Extract Common Name (CN) from X.509 Distinguished Name
 *
 * @param dn - X.509 DN string (e.g., "CN=device.local,O=Example")
 * @returns Common Name value or full DN if CN not found
 */
export function extractCommonName(dn: string): string {
  const cnMatch = dn.match(/CN=([^,]+)/i);
  return cnMatch ? cnMatch[1].trim() : dn;
}

/**
 * Mock certificate fetch for testing/development
 *
 * This is a temporary implementation for development and testing.
 * In production, replace with actual platform-specific implementation.
 */
export async function fetchCertificateMock(
  host: string,
  _port: number = 8443
): Promise<RawCertificateData> {
  // Simulate network delay
  await new Promise(resolve => setTimeout(resolve, 500));

  // Return mock self-signed certificate
  return {
    pemEncoded: `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKJ5qG5wqH5fMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEA0test...mock...certificate...data
-----END CERTIFICATE-----`,
    subject: `CN=${host},O=BoardingPass,C=US`,
    issuer: `CN=${host},O=BoardingPass,C=US`, // Self-signed
    validFrom: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days ago
    validTo: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(), // 1 year from now
  };
}
