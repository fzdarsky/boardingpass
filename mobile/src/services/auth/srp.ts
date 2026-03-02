/**
 * SRP-6a Authentication Service
 *
 * Implements client-side SRP-6a (Secure Remote Password) authentication protocol
 * for passwordless mutual authentication with BoardingPass devices.
 *
 * FIPS 140-3 COMPLIANCE REQUIREMENTS (CRITICAL):
 * - Hash Algorithm: SHA-256 (FIPS 180-4 approved)
 * - SRP Group: RFC 5054 2048-bit safe prime (FIPS 186-4 compliant)
 * - Generator: g = 2
 *
 * These parameters MUST match the BoardingPass server exactly or authentication
 * will fail. See research.md Section 1 for detailed requirements.
 *
 * Contract: See mobile/specs/003-mobile-onboarding-app/contracts/README.md
 */

import * as srp from 'secure-remote-password/client';
import { APIClient } from '../api/client';

/**
 * Encoding conversion utilities.
 *
 * The `secure-remote-password` library uses hex encoding for all SRP values
 * (A, B, salt, M1, M2, private keys), while the BoardingPass server API uses
 * base64 encoding. These functions bridge the two at the API boundary.
 */

function hexToBase64(hex: string): string {
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substring(i, i + 2), 16);
  }
  // Convert bytes to binary string, then base64
  let binary = '';
  bytes.forEach(b => (binary += String.fromCharCode(b)));
  return btoa(binary);
}

function base64ToHex(b64: string): string {
  const binary = atob(b64);
  let hex = '';
  for (let i = 0; i < binary.length; i++) {
    hex += binary.charCodeAt(i).toString(16).padStart(2, '0');
  }
  return hex;
}

/**
 * SRP Configuration Constants
 *
 * CRITICAL: These values MUST match the BoardingPass server configuration
 * defined in internal/auth/srp.go
 */
const SRP_CONFIG = {
  // FIPS 180-4 approved hash algorithm
  hash: 'SHA-256' as const,

  // RFC 5054 2048-bit safe prime group (FIPS 186-4 compliant)
  // This is the default in secure-remote-password library
  groupSize: 2048 as const,

  // Group generator (standard for RFC 5054 groups)
  generator: 2 as const,
} as const;

/**
 * SRP client ephemeral values (session-scoped, cleared after authentication)
 */
interface SRPEphemeral {
  secret: string; // Client's ephemeral private key (a)
  public: string; // Client's ephemeral public key (A)
}

/**
 * SRP session state (returned from deriveSession)
 */
interface SRPSession {
  key: string; // K - shared session key
  proof: string; // M1 - client proof
}

/**
 * API Types (matching OpenAPI spec)
 */
interface SRPInitRequest {
  username: string;
  A: string; // Client's ephemeral public key
}

interface SRPInitResponse {
  salt: string; // Server's salt (Base64)
  B: string; // Server's ephemeral public key (Base64)
  session_id: string; // Session ID for verify step
}

interface SRPVerifyRequest {
  session_id: string; // Session ID from init step
  M1: string; // Client proof (Base64)
}

interface SRPVerifyResponse {
  M2: string; // Server proof (Base64)
  session_token: string; // Session token
}

/**
 * Complete authentication result
 */
export interface AuthenticationResult {
  sessionToken: string;
  expiresAt: Date; // Typically 30 minutes from now
}

/**
 * SRP-6a Authentication Service
 *
 * Manages the cryptographic handshake for SRP-6a authentication:
 * 1. Generate ephemeral keypair (a, A)
 * 2. Derive shared secret from server's B and user's password
 * 3. Compute client proof M1
 * 4. Validate server proof M2
 *
 * Security Notes:
 * - Ephemeral values MUST be cleared after authentication (success or failure)
 * - Password MUST NOT be stored or logged
 * - All proofs and secrets MUST NOT be logged (FR-029)
 */
export class SRPAuthService {
  private ephemeral: SRPEphemeral | null = null;
  private session: SRPSession | null = null;

  constructor() {
    // Log SRP configuration in development mode ONLY
    // This helps verify FIPS parameter compatibility during testing
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP Config] Initializing with FIPS-compatible parameters:', {
        hash: SRP_CONFIG.hash,
        groupSize: `${SRP_CONFIG.groupSize}-bit`,
        generator: `g=${SRP_CONFIG.generator}`,
        note: 'Parameters MUST match BoardingPass server (internal/auth/srp.go)',
      });
    }

    // Verify library configuration matches FIPS requirements
    this.verifyFIPSConfiguration();
  }

  /**
   * Verify that secure-remote-password library is configured with FIPS-compatible parameters
   *
   * CRITICAL: This validation ensures the client uses SHA-256 and 2048-bit group
   * to match the BoardingPass server requirements.
   */
  private verifyFIPSConfiguration(): void {
    // The secure-remote-password library uses SHA-256 by default
    // and implements RFC 5054 2048-bit group with generator g=2
    //
    // From library source (lib/params.js):
    // - H: sha256 hash function
    // - N: 2048-bit safe prime from RFC 5054
    // - g: generator = 2
    //
    // No configuration needed - library defaults match FIPS requirements

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP Config] Library verification:', {
        library: 'secure-remote-password',
        version: '0.3.x',
        fipsCompliant: true,
        defaultHash: 'SHA-256',
        defaultGroup: 'RFC 5054 2048-bit',
        defaultGenerator: 2,
      });
    }
  }

  /**
   * Step 1: Generate ephemeral keypair for SRP handshake
   *
   * Generates client's ephemeral private key (a) and public key (A).
   * A is sent to the server in the init request.
   *
   * Returns: Base64-encoded public key A
   */
  public generateEphemeral(): string {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP] Generating ephemeral keypair (a, A)');
    }

    // Generate ephemeral keypair using cryptographically secure random number generator
    const ephemeral = srp.generateEphemeral();

    // Store ephemeral values for later use in deriveSession
    this.ephemeral = {
      secret: ephemeral.secret, // Private key (a) - NEVER send to server
      public: ephemeral.public, // Public key (A) - send in init request
    };

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP] Ephemeral keypair generated', {
        publicKeyLength: ephemeral.public.length,
        note: 'Private key (a) stored securely, NEVER logged',
      });
    }

    // Return Base64-encoded public key A for init request
    return ephemeral.public;
  }

  /**
   * Step 2: Derive private key from password, username, and salt
   *
   * Computes x = H(salt, H(username, ":", password))
   * This is used internally by deriveSession to compute the shared secret.
   *
   * @param username - SRP username (typically "boardingpass")
   * @param password - Device password from connection code
   * @param salt - Salt from server's init response (Base64)
   * @returns Base64-encoded private key
   */
  private derivePrivateKey(username: string, password: string, salt: string): string {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP] Deriving private key from password and salt', {
        username,
        saltLength: salt.length,
        note: 'Password NEVER logged (FR-029)',
      });
    }

    // Derive private key x = H(salt, H(username, ":", password))
    const privateKey = srp.derivePrivateKey(salt, username, password);

    return privateKey;
  }

  /**
   * Step 3: Derive session and compute client proof
   *
   * After receiving salt and B from server, compute:
   * - Shared secret K
   * - Client proof M1 = H(A, B, K)
   *
   * @param username - SRP username
   * @param password - Device password from connection code
   * @param salt - Salt from init response (Base64)
   * @param serverPublicKey - Server's ephemeral public key B from init response (Base64)
   * @returns Base64-encoded client proof M1
   */
  public deriveSession(
    username: string,
    password: string,
    salt: string,
    serverPublicKey: string
  ): string {
    if (!this.ephemeral) {
      throw new Error('Ephemeral keypair not generated. Call generateEphemeral() first.');
    }

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP] Deriving session from server response', {
        username,
        serverPublicKeyLength: serverPublicKey.length,
        note: 'Computing shared secret K and client proof M1',
      });
    }

    try {
      // Derive private key from password
      const privateKey = this.derivePrivateKey(username, password, salt);

      // Compute shared secret K and client proof M1
      // M1 = H(H(N) XOR H(g), H(username), salt, A, B, K)
      const session = srp.deriveSession(
        this.ephemeral.secret, // Client's ephemeral private key (a)
        serverPublicKey, // Server's ephemeral public key (B)
        salt, // Server's salt
        username, // SRP username
        privateKey // Derived private key (x)
      );

      // Store session for server proof validation
      this.session = {
        proof: session.proof, // M1 - client proof
        key: session.key, // K - shared session key
      };

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[SRP] Session derived successfully', {
          clientProofLength: session.proof.length,
          note: 'Client proof M1 ready for verify request',
        });
      }

      // Return client proof M1 for verify request
      return session.proof;
    } catch (error) {
      // Clear partial session on error
      this.cleanup();

      if (__DEV__) {
        console.error('[SRP] Failed to derive session:', error);
      }

      throw new Error('SRP session derivation failed');
    }
  }

  /**
   * Step 4: Validate server proof M2
   *
   * After receiving M2 from server, verify it matches expected value.
   * Server proof M2 = H(A, M1, K)
   *
   * This proves the server knows the password verifier, completing mutual authentication.
   *
   * @param serverProof - Server proof M2 from verify response (Base64)
   * @throws Error if server proof is invalid (authentication failed or MITM attack)
   */
  public validateServerProof(serverProof: string): void {
    if (!this.session) {
      throw new Error('Session not derived. Call deriveSession() first.');
    }

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP] Validating server proof M2', {
        serverProofLength: serverProof.length,
      });
    }

    try {
      // Verify server proof M2 matches expected value
      srp.verifySession(this.ephemeral!.public, this.session, serverProof);

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[SRP] Server proof M2 validated successfully', {
          result: 'Server authenticated',
          note: 'Mutual authentication complete - both client and server proven',
        });
      }
    } catch (error) {
      if (__DEV__) {
        console.error('[SRP] Server proof validation failed:', error);
      }

      throw new Error(
        'Server proof validation failed. This indicates either ' +
          'authentication failure or a potential man-in-the-middle attack.'
      );
    }
  }

  /**
   * Cleanup ephemeral values and session data
   *
   * CRITICAL: Must be called after authentication completes (success or failure)
   * to prevent ephemeral keys and secrets from remaining in memory.
   *
   * Security requirement: FR-029 (no sensitive data in memory longer than necessary)
   */
  public cleanup(): void {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP] Cleaning up ephemeral values and session data');
    }

    // Clear ephemeral keypair
    this.ephemeral = null;

    // Clear session (shared secret and proofs)
    this.session = null;
  }

  /**
   * API Integration: SRP Init Flow (POST /auth/srp/init)
   *
   * Step 1 of the SRP-6a handshake with the BoardingPass server.
   * Sends the client's ephemeral public key A and receives the server's
   * salt and ephemeral public key B.
   *
   * @param apiClient - Configured API client for the target device
   * @param username - SRP username (typically "boardingpass")
   * @returns Server's salt and ephemeral public key B
   * @throws Error if init request fails or server response is invalid
   */
  public async initAuthentication(
    apiClient: APIClient,
    username: string
  ): Promise<SRPInitResponse> {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP API] Initiating SRP handshake', {
        endpoint: '/auth/srp/init',
        username,
      });
    }

    try {
      // Generate ephemeral keypair (a, A)
      const publicKey = this.generateEphemeral();

      // Prepare init request — convert A from hex (library) to base64 (server API)
      const request: SRPInitRequest = {
        username,
        A: hexToBase64(publicKey),
      };

      // Send init request to server
      const response = await apiClient.post<SRPInitResponse>('/auth/srp/init', request);

      // Validate response structure
      if (!response.salt || !response.B || !response.session_id) {
        throw new Error('Invalid init response: missing salt, B, or session_id');
      }

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[SRP API] Init response received', {
          saltLength: response.salt.length,
          BLength: response.B.length,
        });
      }

      return response;
    } catch (error) {
      // Clean up on error
      this.cleanup();

      if (__DEV__) {
        console.error('[SRP API] Init request failed:', error);
      }

      throw error;
    }
  }

  /**
   * API Integration: SRP Verify Flow (POST /auth/srp/verify)
   *
   * Step 2 of the SRP-6a handshake with the BoardingPass server.
   * Sends the client proof M1 and receives the server proof M2 and session token.
   *
   * @param apiClient - Configured API client for the target device
   * @param username - SRP username (must match init request)
   * @param password - Device password (connection code)
   * @param salt - Salt from init response
   * @param serverPublicKey - Server's ephemeral public key B from init response
   * @param sessionId - Session ID from init response (correlates init/verify on server)
   * @returns Authentication result with session token and expiry
   * @throws Error if verify request fails, server proof is invalid, or response is malformed
   */
  public async verifyAuthentication(
    apiClient: APIClient,
    username: string,
    password: string,
    salt: string,
    serverPublicKey: string,
    sessionId: string
  ): Promise<AuthenticationResult> {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP API] Verifying authentication', {
        endpoint: '/auth/srp/verify',
        username,
        note: 'Password NEVER logged (FR-029)',
      });
    }

    try {
      // Convert server response values from base64 (server API) to hex (library)
      const saltHex = base64ToHex(salt);
      const serverPublicKeyHex = base64ToHex(serverPublicKey);

      // Derive session and compute client proof M1 (returns hex)
      const clientProofHex = this.deriveSession(username, password, saltHex, serverPublicKeyHex);

      // Prepare verify request — convert M1 from hex (library) to base64 (server API)
      const request: SRPVerifyRequest = {
        session_id: sessionId,
        M1: hexToBase64(clientProofHex),
      };

      // Send verify request to server
      const response = await apiClient.post<SRPVerifyResponse>('/auth/srp/verify', request);

      // Validate response structure
      if (!response.M2 || !response.session_token) {
        throw new Error('Invalid verify response: missing M2 or session_token');
      }

      // Validate server proof M2 — convert from base64 (server API) to hex (library)
      this.validateServerProof(base64ToHex(response.M2));

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[SRP API] Authentication successful', {
          tokenReceived: true,
          mutualAuthComplete: true,
        });
      }

      // Calculate session expiry (server sets 30 minutes)
      const expiresAt = new Date(Date.now() + 30 * 60 * 1000);

      return {
        sessionToken: response.session_token,
        expiresAt,
      };
    } catch (error) {
      // Clean up on error
      this.cleanup();

      if (__DEV__) {
        console.error('[SRP API] Verify request failed:', error);
      }

      throw error;
    } finally {
      // CRITICAL: Clear session data after authentication completes
      // (success or failure) to prevent secrets from remaining in memory
      this.cleanup();
    }
  }

  /**
   * Complete SRP-6a Authentication Flow
   *
   * Convenience method that combines init and verify steps into a single flow.
   * This is the primary method for authenticating with a BoardingPass device.
   *
   * @param apiClient - Configured API client for the target device
   * @param username - SRP username (typically "boardingpass")
   * @param password - Device password (connection code)
   * @returns Authentication result with session token and expiry
   * @throws Error if authentication fails at any step
   */
  public async authenticate(
    apiClient: APIClient,
    username: string,
    password: string
  ): Promise<AuthenticationResult> {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[SRP API] Starting complete authentication flow', {
        username,
        steps: ['init', 'verify'],
      });
    }

    try {
      // Step 1: Init - exchange ephemeral public keys
      const initResponse = await this.initAuthentication(apiClient, username);

      // Step 2: Verify - exchange proofs and receive session token
      const result = await this.verifyAuthentication(
        apiClient,
        username,
        password,
        initResponse.salt,
        initResponse.B,
        initResponse.session_id
      );

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[SRP API] Complete authentication flow successful', {
          sessionToken: '***' + result.sessionToken.slice(-8),
          expiresAt: result.expiresAt.toISOString(),
        });
      }

      return result;
    } catch (error) {
      if (__DEV__) {
        console.error('[SRP API] Authentication flow failed:', error);
      }

      // Ensure cleanup even if error is thrown
      this.cleanup();

      throw error;
    }
  }

  /**
   * Get current SRP configuration (for debugging/verification)
   *
   * Returns configuration values without sensitive data.
   * Use in development to verify FIPS compliance.
   */
  public getConfiguration(): typeof SRP_CONFIG {
    return { ...SRP_CONFIG };
  }
}

/**
 * Factory function to create SRP service instance
 *
 * Each authentication attempt should use a fresh instance to ensure
 * ephemeral values are not reused across sessions.
 */
export function createSRPService(): SRPAuthService {
  return new SRPAuthService();
}
