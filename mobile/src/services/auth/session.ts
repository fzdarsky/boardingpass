/**
 * Session Management Service
 *
 * Manages authentication session lifecycle including:
 * - Secure storage of session tokens
 * - Session expiry checking
 * - Token persistence and retrieval
 *
 * Security:
 * - Uses expo-secure-store for encrypted token storage (OS Keychain/Keystore)
 * - Tokens are never logged (FR-029)
 * - Expired sessions are automatically cleared
 */

import * as SecureStore from 'expo-secure-store';

/**
 * Session data stored securely
 */
export interface SessionData {
  deviceId: string; // Device this session is for
  sessionToken: string; // JWT or opaque token from server
  expiresAt: string; // ISO 8601 date string
  createdAt: string; // ISO 8601 date string
}

/**
 * Session validation result
 */
export interface SessionValidation {
  isValid: boolean;
  reason?: 'expired' | 'not_found' | 'invalid_format';
}

/**
 * Secure storage key prefix for session tokens
 */
const SESSION_KEY_PREFIX = 'boardingpass_session_';

/**
 * Session Management Service
 *
 * Handles secure storage and lifecycle management of authentication sessions.
 */
export class SessionManager {
  /**
   * Store a new session securely
   *
   * @param deviceId - Device identifier
   * @param sessionToken - Session token from server
   * @param expiresAt - Session expiration time
   * @throws Error if storage fails
   */
  async storeSession(deviceId: string, sessionToken: string, expiresAt: Date): Promise<void> {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Session] Storing session', {
        deviceId,
        expiresAt: expiresAt.toISOString(),
        note: 'Token NEVER logged (FR-029)',
      });
    }

    const sessionData: SessionData = {
      deviceId,
      sessionToken,
      expiresAt: expiresAt.toISOString(),
      createdAt: new Date().toISOString(),
    };

    try {
      const key = this.getStorageKey(deviceId);
      await SecureStore.setItemAsync(key, JSON.stringify(sessionData));

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Session] Session stored successfully', {
          deviceId,
          key,
        });
      }
    } catch (error) {
      if (__DEV__) {
        console.error('[Session] Failed to store session:', error);
      }
      throw new Error('Failed to store session securely');
    }
  }

  /**
   * Retrieve a session from secure storage
   *
   * @param deviceId - Device identifier
   * @returns Session data if found, null if not found or expired
   */
  async getSession(deviceId: string): Promise<SessionData | null> {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Session] Retrieving session', { deviceId });
    }

    try {
      const key = this.getStorageKey(deviceId);
      const data = await SecureStore.getItemAsync(key);

      if (!data) {
        if (__DEV__) {
          // eslint-disable-next-line no-console
          console.log('[Session] No session found', { deviceId });
        }
        return null;
      }

      const sessionData = JSON.parse(data) as SessionData;

      // Validate session structure
      if (!sessionData.sessionToken || !sessionData.expiresAt) {
        if (__DEV__) {
          console.error('[Session] Invalid session data structure');
        }
        // Clear invalid session
        await this.clearSession(deviceId);
        return null;
      }

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Session] Session retrieved', {
          deviceId,
          expiresAt: sessionData.expiresAt,
          note: 'Token NEVER logged (FR-029)',
        });
      }

      return sessionData;
    } catch (error) {
      if (__DEV__) {
        console.error('[Session] Failed to retrieve session:', error);
      }
      return null;
    }
  }

  /**
   * Check if a session is valid (exists and not expired)
   *
   * @param deviceId - Device identifier
   * @returns Validation result with reason if invalid
   */
  async isSessionValid(deviceId: string): Promise<SessionValidation> {
    const session = await this.getSession(deviceId);

    if (!session) {
      return {
        isValid: false,
        reason: 'not_found',
      };
    }

    const expiresAt = new Date(session.expiresAt);
    const now = new Date();

    if (expiresAt <= now) {
      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Session] Session expired', {
          deviceId,
          expiresAt: session.expiresAt,
          now: now.toISOString(),
        });
      }

      // Clear expired session
      await this.clearSession(deviceId);

      return {
        isValid: false,
        reason: 'expired',
      };
    }

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Session] Session is valid', {
        deviceId,
        expiresAt: session.expiresAt,
        timeRemaining: `${Math.floor((expiresAt.getTime() - now.getTime()) / 1000 / 60)} minutes`,
      });
    }

    return { isValid: true };
  }

  /**
   * Get the session token for a device (if valid)
   *
   * @param deviceId - Device identifier
   * @returns Session token if valid, null otherwise
   */
  async getValidToken(deviceId: string): Promise<string | null> {
    const validation = await this.isSessionValid(deviceId);

    if (!validation.isValid) {
      return null;
    }

    const session = await this.getSession(deviceId);
    return session?.sessionToken || null;
  }

  /**
   * Clear a session from secure storage
   *
   * @param deviceId - Device identifier
   */
  async clearSession(deviceId: string): Promise<void> {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Session] Clearing session', { deviceId });
    }

    try {
      const key = this.getStorageKey(deviceId);
      await SecureStore.deleteItemAsync(key);

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Session] Session cleared', { deviceId });
      }
    } catch (error) {
      if (__DEV__) {
        console.error('[Session] Failed to clear session:', error);
      }
    }
  }

  /**
   * Clear all sessions (logout from all devices)
   */
  async clearAllSessions(): Promise<void> {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Session] Clearing all sessions');
    }

    // Note: expo-secure-store doesn't provide a way to list all keys,
    // so this is a placeholder for when we track device IDs separately.
    // For now, sessions are cleared individually when accessed.

    // TODO: If we need to track all sessions, we should maintain a list
    // of device IDs in a separate storage location.
  }

  /**
   * Get time remaining until session expires
   *
   * @param deviceId - Device identifier
   * @returns Time remaining in milliseconds, or null if session invalid
   */
  async getTimeRemaining(deviceId: string): Promise<number | null> {
    const session = await this.getSession(deviceId);

    if (!session) {
      return null;
    }

    const expiresAt = new Date(session.expiresAt);
    const now = new Date();
    const remaining = expiresAt.getTime() - now.getTime();

    return remaining > 0 ? remaining : null;
  }

  /**
   * Get storage key for a device's session
   *
   * SecureStore keys may only contain alphanumeric characters, '.', '-', and '_'.
   * Device IDs use colons as separators (e.g., "manual:192.168.1.1:9455"),
   * so we replace them with underscores.
   *
   * @param deviceId - Device identifier
   * @returns Secure storage key safe for SecureStore
   */
  private getStorageKey(deviceId: string): string {
    const sanitized = deviceId.replace(/[^a-zA-Z0-9._-]/g, '_');
    return `${SESSION_KEY_PREFIX}${sanitized}`;
  }
}

/**
 * Factory function to create session manager instance
 */
export function createSessionManager(): SessionManager {
  return new SessionManager();
}

/**
 * Singleton session manager instance for app-wide use
 */
export const sessionManager = createSessionManager();
