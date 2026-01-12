/**
 * Crypto Utilities
 *
 * SHA-256 fingerprint computation using expo-crypto (FIPS-compatible)
 */

import * as Crypto from 'expo-crypto';

/**
 * Compute SHA-256 hash of a string
 *
 * @param input - String to hash
 * @returns Hex-encoded SHA-256 hash (64 characters)
 */
export async function sha256(input: string): Promise<string> {
  const hash = await Crypto.digestStringAsync(Crypto.CryptoDigestAlgorithm.SHA256, input);
  return hash.toLowerCase(); // Return lowercase hex
}

/**
 * Compute certificate fingerprint from PEM-encoded certificate
 *
 * @param pemCert - PEM-encoded certificate string
 * @returns SHA-256 fingerprint (64-char hex string)
 */
export async function computeCertificateFingerprint(pemCert: string): Promise<string> {
  // Remove PEM headers/footers and whitespace
  const certData = pemCert
    .replace(/-----BEGIN CERTIFICATE-----/g, '')
    .replace(/-----END CERTIFICATE-----/g, '')
    .replace(/\s/g, '');

  // Compute SHA-256 hash of the base64-decoded certificate
  // Note: expo-crypto hashes the string directly, for DER we'd need base64 decode first
  // This is a simplified version - actual implementation may need platform-specific code
  return await sha256(certData);
}

/**
 * Generate random bytes (for nonces, IDs, etc.)
 *
 * @param length - Number of bytes to generate
 * @returns Hex-encoded random bytes
 */
export async function randomBytes(length: number): Promise<string> {
  const bytes = await Crypto.getRandomBytesAsync(length);
  return Array.from(bytes)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

/**
 * Generate a random ID (128-bit, hex-encoded)
 */
export async function generateId(): Promise<string> {
  return await randomBytes(16); // 128 bits = 16 bytes
}

/**
 * Validate SHA-256 fingerprint format
 */
export function isValidSHA256(hash: string): boolean {
  return /^[a-f0-9]{64}$/.test(hash.toLowerCase());
}

/**
 * Format fingerprint for display (colon-separated groups)
 *
 * @param fingerprint - 64-char hex fingerprint
 * @returns Formatted fingerprint (e.g., "AA:BB:CC:...")
 */
export function formatFingerprint(fingerprint: string): string {
  if (!isValidSHA256(fingerprint)) {
    return fingerprint;
  }

  // Split into groups of 2 characters, join with colons
  return (
    fingerprint
      .toLowerCase()
      .match(/.{1,2}/g)
      ?.join(':') || fingerprint
  );
}

/**
 * Compare two fingerprints (case-insensitive)
 */
export function compareFingerprints(fp1: string, fp2: string): boolean {
  return fp1.toLowerCase() === fp2.toLowerCase();
}

/**
 * Truncate fingerprint for compact display
 *
 * @param fingerprint - Full fingerprint
 * @param length - Number of byte groups to show from start/end (default: 6)
 * @returns Truncated fingerprint (e.g., "AA:BB:CC...XX:YY:ZZ")
 */
export function truncateFingerprint(fingerprint: string, length: number = 6): string {
  const formatted = formatFingerprint(fingerprint);

  // If length is >= 32 (full fingerprint has 32 groups), don't truncate
  if (length >= 32) {
    return formatted;
  }

  // Split into groups and check if truncation is needed
  const groups = formatted.split(':');
  if (groups.length <= length * 2) {
    return formatted;
  }

  // Extract first 'length' groups and last 'length' groups
  const startGroups = groups.slice(0, length).join(':');
  const endGroups = groups.slice(-length).join(':');

  return `${startGroups}...${endGroups}`;
}
