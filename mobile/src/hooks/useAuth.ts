/**
 * useAuth Hook
 *
 * Provides authentication functionality for BoardingPass devices.
 * Combines SRP-6a authentication, session management, and API client configuration.
 *
 * Usage:
 * ```tsx
 * const { authenticate, logout, isAuthenticated, error } = useAuth(deviceId);
 *
 * await authenticate(host, port, connectionCode);
 * ```
 */

import { useState, useCallback, useEffect, useRef } from 'react';
import { createSRPService, AuthenticationResult } from '../services/auth/srp';
import { sessionManager, SessionData } from '../services/auth/session';
import { createAPIClient } from '../services/api/client';
import { toAppError, isTimeoutError, isAuthError, isNetworkError, AppError } from '../utils/error-messages';

// Progressive delay thresholds (FR-038)
const PROGRESSIVE_DELAYS = [1000, 2000, 5000, 60000]; // 1s, 2s, 5s, 60s

/**
 * Authentication state
 */
export interface AuthState {
  isAuthenticated: boolean;
  isAuthenticating: boolean;
  sessionToken: string | null;
  expiresAt: Date | null;
  error: AppError | null;
  failureCount: number;
  nextRetryDelay: number;
}

/**
 * Authentication hook result
 */
export interface UseAuthResult extends AuthState {
  /**
   * Authenticate with a device using connection code
   *
   * @param host - Device IP address or hostname
   * @param port - Device HTTPS port (default 8443)
   * @param connectionCode - Device connection code (password)
   * @param username - SRP username (default "boardingpass")
   * @returns Authentication result with session token
   */
  authenticate: (
    host: string,
    port: number,
    connectionCode: string,
    username?: string
  ) => Promise<AuthenticationResult>;

  /**
   * Logout from the device (clear session)
   */
  logout: () => Promise<void>;

  /**
   * Check if there's a valid session
   */
  checkSession: () => Promise<boolean>;

  /**
   * Get the current session token
   */
  getToken: () => Promise<string | null>;

  /**
   * Clear the current error
   */
  clearError: () => void;

  /**
   * Retry authentication (only available when error exists)
   */
  retry: (() => Promise<void>) | undefined;
}

/**
 * Authentication hook for a specific device
 *
 * @param deviceId - Unique device identifier
 * @returns Authentication state and operations
 */
export function useAuth(deviceId: string): UseAuthResult {
  const [state, setState] = useState<AuthState>({
    isAuthenticated: false,
    isAuthenticating: false,
    sessionToken: null,
    expiresAt: null,
    error: null,
    failureCount: 0,
    nextRetryDelay: 0,
  });

  // Track last auth attempt details for retry
  const lastAuthAttempt = useRef<{
    host: string;
    port: number;
    connectionCode: string;
    username: string;
  } | null>(null);

  /**
   * Load existing session on mount
   */
  useEffect(() => {
    const loadSession = async () => {
      try {
        const session = await sessionManager.getSession(deviceId);

        if (session) {
          const validation = await sessionManager.isSessionValid(deviceId);

          if (validation.isValid) {
            setState(prev => ({
              ...prev,
              isAuthenticated: true,
              sessionToken: session.sessionToken,
              expiresAt: new Date(session.expiresAt),
            }));
          } else {
            // Clear expired session
            await sessionManager.clearSession(deviceId);
          }
        }
      } catch (error) {
        if (__DEV__) {
          console.error('[useAuth] Failed to load session:', error);
        }
      }
    };

    loadSession();
  }, [deviceId]);

  /**
   * Authenticate with the device
   */
  const authenticate = useCallback(
    async (
      host: string,
      port: number = 8443,
      connectionCode: string,
      username: string = 'boardingpass'
    ): Promise<AuthenticationResult> => {
      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[useAuth] Starting authentication', {
          deviceId,
          host,
          port,
          username,
          failureCount: state.failureCount,
        });
      }

      // Store for retry
      lastAuthAttempt.current = { host, port, connectionCode, username };

      // Apply progressive delay for failed attempts (FR-038)
      if (state.failureCount > 0) {
        const delayIndex = Math.min(state.failureCount - 1, PROGRESSIVE_DELAYS.length - 1);
        const delay = PROGRESSIVE_DELAYS[delayIndex];

        if (__DEV__) {
          console.log(`[useAuth] Applying progressive delay: ${delay}ms (attempt ${state.failureCount + 1})`);
        }

        await new Promise(resolve => setTimeout(resolve, delay));
      }

      // Set authenticating state
      setState(prev => ({
        ...prev,
        isAuthenticating: true,
        error: null,
      }));

      try {
        // Create API client for the device
        const apiClient = createAPIClient(host, port);

        // Create fresh SRP service instance
        const srpService = createSRPService();

        // Perform SRP-6a authentication
        const result = await srpService.authenticate(apiClient, username, connectionCode);

        // Store session securely
        await sessionManager.storeSession(deviceId, result.sessionToken, result.expiresAt);

        // Update state - clear failure count on success
        setState({
          isAuthenticated: true,
          isAuthenticating: false,
          sessionToken: result.sessionToken,
          expiresAt: result.expiresAt,
          error: null,
          failureCount: 0,
          nextRetryDelay: 0,
        });

        // Clear last attempt
        lastAuthAttempt.current = null;

        if (__DEV__) {
          // eslint-disable-next-line no-console
          console.log('[useAuth] Authentication successful', {
            deviceId,
            expiresAt: result.expiresAt.toISOString(),
          });
        }

        return result;
      } catch (error) {
        const newFailureCount = state.failureCount + 1;
        const nextDelayIndex = Math.min(newFailureCount - 1, PROGRESSIVE_DELAYS.length - 1);
        const nextDelay = PROGRESSIVE_DELAYS[nextDelayIndex];

        // Classify error type
        let errorType: 'timeout' | 'auth' | 'network' | 'unknown' = 'unknown';
        if (isTimeoutError(error)) {
          errorType = 'timeout';
        } else if (isAuthError(error)) {
          errorType = 'auth';
        } else if (isNetworkError(error)) {
          errorType = 'network';
        }

        const appError = toAppError(error, errorType);
        appError.context = {
          ...appError.context,
          operation: 'auth',
          deviceId,
          failureCount: newFailureCount,
        };

        setState({
          isAuthenticated: false,
          isAuthenticating: false,
          sessionToken: null,
          expiresAt: null,
          error: appError,
          failureCount: newFailureCount,
          nextRetryDelay: nextDelay,
        });

        if (__DEV__) {
          console.error('[useAuth] Authentication failed:', {
            error: appError.message,
            type: appError.type,
            failureCount: newFailureCount,
            nextRetryDelay: nextDelay,
          });
        }

        throw appError;
      }
    },
    [deviceId, state.failureCount]
  );

  /**
   * Logout from the device
   */
  const logout = useCallback(async (): Promise<void> => {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[useAuth] Logging out', { deviceId });
    }

    try {
      // Clear session from secure storage
      await sessionManager.clearSession(deviceId);

      // Update state - clear failure count
      setState({
        isAuthenticated: false,
        isAuthenticating: false,
        sessionToken: null,
        expiresAt: null,
        error: null,
        failureCount: 0,
        nextRetryDelay: 0,
      });

      // Clear last attempt
      lastAuthAttempt.current = null;

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[useAuth] Logout successful', { deviceId });
      }
    } catch (error) {
      if (__DEV__) {
        console.error('[useAuth] Logout failed:', error);
      }
    }
  }, [deviceId]);

  /**
   * Check if there's a valid session
   */
  const checkSession = useCallback(async (): Promise<boolean> => {
    try {
      const validation = await sessionManager.isSessionValid(deviceId);

      if (!validation.isValid) {
        // Clear state if session is invalid
        setState(prev => ({
          ...prev,
          isAuthenticated: false,
          sessionToken: null,
          expiresAt: null,
        }));
      }

      return validation.isValid;
    } catch (error) {
      if (__DEV__) {
        console.error('[useAuth] Session check failed:', error);
      }
      return false;
    }
  }, [deviceId]);

  /**
   * Get the current session token
   */
  const getToken = useCallback(async (): Promise<string | null> => {
    try {
      return await sessionManager.getValidToken(deviceId);
    } catch (error) {
      if (__DEV__) {
        console.error('[useAuth] Failed to get token:', error);
      }
      return null;
    }
  }, [deviceId]);

  /**
   * Clear the current error
   */
  const clearError = useCallback(() => {
    setState(prev => ({
      ...prev,
      error: null,
    }));
  }, []);

  /**
   * Retry authentication with last attempt params
   */
  const retry = useCallback(async (): Promise<void> => {
    if (!lastAuthAttempt.current) {
      throw new Error('No previous authentication attempt to retry');
    }

    const { host, port, connectionCode, username } = lastAuthAttempt.current;
    await authenticate(host, port, connectionCode, username);
  }, [authenticate]);

  return {
    ...state,
    authenticate,
    logout,
    checkSession,
    getToken,
    clearError,
    retry: state.error ? retry : undefined,
  };
}

/**
 * Get session data for a device (utility function)
 *
 * @param deviceId - Device identifier
 * @returns Session data if exists and valid, null otherwise
 */
export async function getDeviceSession(deviceId: string): Promise<SessionData | null> {
  const validation = await sessionManager.isSessionValid(deviceId);
  if (!validation.isValid) {
    return null;
  }
  return sessionManager.getSession(deviceId);
}

/**
 * Check if device has valid session (utility function)
 *
 * @param deviceId - Device identifier
 * @returns True if device has valid session
 */
export async function hasValidSession(deviceId: string): Promise<boolean> {
  const validation = await sessionManager.isSessionValid(deviceId);
  return validation.isValid;
}
