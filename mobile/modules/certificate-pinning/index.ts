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

/**
 * Return the full native pin store (UserDefaults) for diagnostics.
 * Keys are "host:port" strings, values are SHA-256 fingerprints.
 */
export function getPinStore(): Record<string, string> {
  return CertificatePinningNative.getPinStore();
}

/**
 * Diagnostic result from testing HTTPS connectivity.
 */
export interface TLSDiagnosticResult {
  success: boolean;
  statusCode?: number;
  bodyLength?: number;
  error?: string;
  challengeHandlerCalled: boolean;
  challengeHost: string;
  challengePort: number;
  decision: string;
  pinStore: Record<string, string>;
}

/**
 * Test HTTPS connectivity to a host using a fresh NSURLSession with
 * our TOFU challenge handler. Bypasses RCTHTTPRequestHandler to verify
 * TLS handling works at the native level.
 */
export async function diagnoseTLS(host: string, port: number = 8443): Promise<TLSDiagnosticResult> {
  return await CertificatePinningNative.diagnoseTLS(host, port);
}

/**
 * Result from invalidating the HTTP session.
 */
export interface InvalidateSessionResult {
  success: boolean;
  method?: string;
  error?: string;
}

/**
 * Invalidate RCTHTTPRequestHandler's cached NSURLSession.
 *
 * Call this after pinning a certificate to force the networking layer
 * to create a fresh session that picks up the TLS challenge handler
 * and recognizes the newly pinned certificate.
 *
 * NSURLSession caches delegate capability checks at creation time.
 * If the session was created before our TOFU handler was injected,
 * it won't call our handler. Invalidation forces recreation.
 */
export function invalidateHTTPSession(): InvalidateSessionResult {
  return CertificatePinningNative.invalidateHTTPSession();
}

/**
 * Get the number of times the TLS challenge handler has been called.
 * Use with resetChallengeCount() to measure if the handler fires for
 * a specific request (diagnostic only).
 */
export function getChallengeCount(): number {
  return CertificatePinningNative.getChallengeCount();
}

/**
 * Reset the TLS challenge handler call counter to 0 (diagnostic only).
 */
export function resetChallengeCount(): void {
  CertificatePinningNative.resetChallengeCount();
}

/**
 * Result from a native HTTPS fetch request.
 */
export interface NativeFetchResult {
  /** HTTP status code (0 if request failed before getting a response) */
  status: number;
  /** Response headers */
  headers?: Record<string, string>;
  /** Response body as string */
  body?: string;
  /** Error message if the request failed */
  error?: string;
  /** Error code (e.g., ERR_NETWORK, ECONNABORTED, ECONNREFUSED) */
  code?: string;
}

/**
 * Perform an HTTPS request using a native URLSession with TOFU certificate
 * handling. Bypasses React Native's networking stack entirely to avoid
 * NSURLSession delegate injection issues with self-signed certificates.
 *
 * @param url - Full URL (e.g., "https://192.168.4.205:8443/auth/srp/init")
 * @param method - HTTP method (GET, POST, PUT, DELETE)
 * @param headers - Request headers
 * @param body - Request body (empty string for no body)
 * @param timeoutMs - Request timeout in milliseconds (default: 30000)
 */
export async function nativeFetch(
  url: string,
  method: string,
  headers: Record<string, string>,
  body: string = '',
  timeoutMs: number = 30000
): Promise<NativeFetchResult> {
  return await CertificatePinningNative.nativeFetch(url, method, headers, body, timeoutMs);
}
