/**
 * Authentication Entity Types
 *
 * Represents secure SRP-6a authentication sessions and connection codes.
 *
 * CRITICAL: SRP-6a MUST use FIPS 140-3 compliant parameters:
 * - Hash Algorithm: SHA-256 (FIPS 180-4 approved)
 * - SRP Group: RFC 5054 2048-bit safe prime (FIPS 186-4 compliant)
 * - Generator: g = 2
 */

/**
 * AuthenticationSession
 *
 * Represents a secure authentication session with a device.
 * Sessions expire after 30 minutes (server-enforced).
 */
export interface AuthenticationSession {
  // Session identity
  deviceId: string;
  sessionToken: string;

  // Timing
  createdAt: Date;
  expiresAt: Date;

  // SRP-6a protocol state (ephemeral, never persisted)
  srp?: SRPState;

  // Authentication metadata
  authenticatedAt: Date;
  connectionCode?: string; // NEVER PERSISTED - cleared after auth
}

/**
 * SRPState
 *
 * SRP-6a protocol state - ephemeral, exists only in memory during authentication
 */
export interface SRPState {
  ephemeralPublic: string; // Client's ephemeral public key (A)
  ephemeralPrivate: string; // Client's ephemeral private key (a) - NEVER LOGGED
  sessionKey?: string; // Derived session key (K) - NEVER LOGGED
}

/**
 * ConnectionCode
 *
 * Authentication credentials for a device.
 * NEVER PERSISTED - exists only in memory during auth flow.
 */
export interface ConnectionCode {
  // Code value
  value: string; // NEVER PERSISTED, NEVER LOGGED

  // Source
  source: ConnectionCodeSource;

  // Validation
  validationState: ConnectionCodeValidationState;
  validatedAt?: Date;

  // Associated device
  deviceId: string;
}

export type ConnectionCodeSource = 'manual' | 'qr' | 'barcode';

export type ConnectionCodeValidationState = 'pending' | 'valid' | 'invalid' | 'used';

/**
 * SRP Protocol Constants
 *
 * FIPS 140-3 compliant parameters
 */
export const SRP_CONFIG = {
  hash: 'sha256' as const, // MUST be SHA-256 (FIPS 180-4)
  group: 'rfc5054-2048' as const, // MUST be 2048-bit (FIPS 186-4)
  generator: 2 as const, // MUST be g=2
} as const;

/**
 * Session validation
 */
export const isSessionValid = (session: AuthenticationSession): boolean => {
  const now = new Date();
  return now < new Date(session.expiresAt);
};

export const getSessionTimeRemaining = (session: AuthenticationSession): number => {
  const now = new Date();
  const expiresAt = new Date(session.expiresAt);
  return Math.max(0, expiresAt.getTime() - now.getTime());
};

/**
 * Connection code validation
 *
 * Format TBD during implementation - likely base64-encoded
 */
export const validateConnectionCodeFormat = (code: string): boolean => {
  // Base64 pattern: alphanumeric + / + =, minimum 32 chars
  const base64Pattern = /^[A-Za-z0-9+/=]{32,}$/;
  return base64Pattern.test(code);
};
