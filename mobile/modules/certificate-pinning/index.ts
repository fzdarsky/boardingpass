/**
 * Certificate Pinning Native Module
 *
 * TypeScript API for the native iOS certificate pinning module.
 * Provides certificate fetching via TLS handshake and pin management
 * through a shared UserDefaults store that the TLS override reads from.
 */

import { requireNativeModule } from 'expo-modules-core';

const CertificatePinningNative = requireNativeModule('CertificatePinning');

/**
 * Server certificate information returned from a TLS handshake.
 */
export interface ServerCertificate {
  /** SHA-256 fingerprint of the DER-encoded certificate (64-char lowercase hex) */
  fingerprint: string;
  /** X.509 subject (typically the CN) */
  subject: string;
  /** X.509 issuer (same as subject for self-signed) */
  issuer: string;
  /** Whether the certificate is self-signed (subject === issuer) */
  isSelfSigned: boolean;
  /** PEM-encoded certificate */
  pemEncoded: string;
  /** ISO 8601 notBefore date (may be empty if parsing failed) */
  validFrom: string;
  /** ISO 8601 notAfter date (may be empty if parsing failed) */
  validTo: string;
}

/**
 * Fetch the TLS certificate from a remote server.
 *
 * Opens a TLS connection, captures the server certificate (including self-signed),
 * computes the SHA-256 fingerprint, and returns certificate metadata.
 *
 * This function accepts ANY certificate during the handshake — it performs
 * inspection, not validation. Validation is the caller's responsibility
 * (via the TOFU flow).
 */
export async function fetchServerCertificate(
  host: string,
  port: number = 8443
): Promise<ServerCertificate> {
  return await CertificatePinningNative.fetchServerCertificate(host, port);
}

/**
 * Pin a certificate fingerprint for a host.
 *
 * After pinning, the TLS override will:
 * - Accept connections whose certificate matches this fingerprint
 * - Reject connections whose certificate does NOT match
 *
 * @param hostKey - Host identifier (e.g., "192.168.4.205" or "192.168.4.205:8443")
 * @param fingerprint - SHA-256 fingerprint (64-char hex)
 */
export function pinCertificate(hostKey: string, fingerprint: string): void {
  CertificatePinningNative.pinCertificate(hostKey, fingerprint);
}

/**
 * Remove a pinned certificate for a host.
 * After removal, the TLS override will revert to TOFU behavior for this host.
 */
export function unpinCertificate(hostKey: string): void {
  CertificatePinningNative.unpinCertificate(hostKey);
}

/**
 * Get the pinned fingerprint for a host.
 * @returns SHA-256 fingerprint or null if no pin exists
 */
export function getPinnedFingerprint(hostKey: string): string | null {
  return CertificatePinningNative.getPinnedFingerprint(hostKey);
}

/**
 * Clear all pinned certificates.
 */
export function clearAllPins(): void {
  CertificatePinningNative.clearAllPins();
}

/**
 * Check if the TLS challenge handler was successfully injected into
 * RCTHTTPRequestHandler. Returns true if HTTPS requests to self-signed
 * servers will use our TOFU logic instead of being rejected.
 */
export function isTLSOverrideActive(): boolean {
  return CertificatePinningNative.isTLSOverrideActive();
}
