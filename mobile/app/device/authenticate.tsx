/**
 * Device Authentication Screen
 *
 * Allows users to authenticate with a BoardingPass device using either:
 * - QR code scanning (camera-based)
 * - Manual connection code entry
 *
 * Features:
 * - Manual/QR toggle UI (T072)
 * - Connection code validation (T069, T071)
 * - Camera permission handling with rationale (T070, FR-026)
 * - Progressive delay on auth failures (T074, FR-038)
 * - Comprehensive error handling (T075, FR-024, FR-028)
 * - Success transition to device detail (T076)
 * - Secure logging without sensitive data (T077, FR-029)
 * - Connection code cleared from memory after auth (T073, FR-036)
 *
 * Usage:
 * Navigate to this screen with device information as params:
 * ```tsx
 * router.push({
 *   pathname: '/device/authenticate',
 *   params: { deviceId, deviceName, host, port }
 * });
 * ```
 */

import React, { useState, useEffect, useRef } from 'react';
import { StyleSheet, View, ScrollView, BackHandler } from 'react-native';
import { useRouter, useLocalSearchParams } from 'expo-router';
import {
  Surface,
  Text,
  Button,
  SegmentedButtons,
  ActivityIndicator,
  Snackbar,
} from 'react-native-paper';
import { spacing, theme } from '@/theme';
import { useAuth } from '@/hooks/useAuth';
import QRScanner from '@/components/QRScanner';
import ConnectionCodeInput from '@/components/ConnectionCodeInput';
import {
  authenticationStartFeedback,
  authenticationSuccessFeedback,
  authenticationFailureFeedback,
  qrCodeScannedFeedback,
  HapticFeedback,
} from '@/utils/haptics';

/**
 * Authentication mode
 */
type AuthMode = 'manual' | 'qr';

/**
 * Progressive delay configuration (FR-038)
 * Delays: 1s → 2s → 5s → 60s (lockout)
 */
const AUTH_FAILURE_DELAYS = [1000, 2000, 5000, 60000];
const LOCKOUT_THRESHOLD = 3; // After 3 failures, apply 60s lockout

/**
 * Device Authentication Screen
 */
export default function AuthenticateScreen(): React.ReactElement {
  const router = useRouter();
  const params = useLocalSearchParams<{
    deviceId: string;
    deviceName: string;
    host: string;
    port: string;
  }>();

  const { deviceId, deviceName, host, port } = params;
  const portNumber = port ? parseInt(port, 10) : 8443;

  const { authenticate, isAuthenticating, error, clearError } = useAuth(deviceId);

  // State
  const [authMode, setAuthMode] = useState<AuthMode>('manual');
  const [connectionCode, setConnectionCode] = useState('');
  const [failureCount, setFailureCount] = useState(0);
  const [isDelaying, setIsDelaying] = useState(false);
  const [delayRemaining, setDelayRemaining] = useState(0);
  const [showSnackbar, setShowSnackbar] = useState(false);
  const [snackbarMessage, setSnackbarMessage] = useState('');

  // Refs for cleanup
  const delayTimerRef = useRef<NodeJS.Timeout | null>(null);
  const countdownTimerRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Log authentication screen mount
   */
  useEffect(() => {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Authenticate] Screen mounted', {
        deviceId,
        deviceName,
        host,
        port: portNumber,
        note: 'Connection code will NEVER be logged (FR-029)',
      });
    }

    // Cleanup on unmount
    return () => {
      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Authenticate] Screen unmounting - clearing sensitive data (FR-036)');
      }

      // Clear connection code from memory (FR-036)
      setConnectionCode('');

      // Clear timers
      if (delayTimerRef.current) {
        clearTimeout(delayTimerRef.current);
      }
      if (countdownTimerRef.current) {
        clearInterval(countdownTimerRef.current);
      }
    };
  }, [deviceId, deviceName, host, portNumber]);

  /**
   * Handle hardware back button (Android)
   */
  useEffect(() => {
    const backHandler = BackHandler.addEventListener('hardwareBackPress', () => {
      if (authMode === 'qr') {
        setAuthMode('manual');
        return true; // Prevent default back action
      }
      return false; // Allow default back action
    });

    return () => backHandler.remove();
  }, [authMode]);

  /**
   * Clear connection code when navigating away (FR-036)
   */
  useEffect(() => {
    return () => {
      setConnectionCode('');
    };
  }, []);

  /**
   * Apply progressive delay after authentication failure (FR-038)
   */
  const applyFailureDelay = (count: number) => {
    // Determine delay based on failure count
    const delayIndex = Math.min(count, AUTH_FAILURE_DELAYS.length - 1);
    const delay = AUTH_FAILURE_DELAYS[delayIndex];

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Authenticate] Applying failure delay', {
        failureCount: count,
        delayMs: delay,
        isLockout: count >= LOCKOUT_THRESHOLD,
      });
    }

    setIsDelaying(true);
    setDelayRemaining(Math.floor(delay / 1000));

    // Countdown timer (update every second)
    countdownTimerRef.current = setInterval(() => {
      setDelayRemaining(prev => {
        if (prev <= 1) {
          if (countdownTimerRef.current) {
            clearInterval(countdownTimerRef.current);
          }
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    // Main delay timer
    delayTimerRef.current = setTimeout(() => {
      setIsDelaying(false);
      setDelayRemaining(0);
      if (countdownTimerRef.current) {
        clearInterval(countdownTimerRef.current);
      }

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Authenticate] Failure delay expired - retry allowed');
      }
    }, delay);
  };

  /**
   * Handle authentication attempt
   */
  const handleAuthenticate = async () => {
    if (!connectionCode || connectionCode.trim().length === 0) {
      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Authenticate] Authentication blocked - empty code');
      }
      return;
    }

    if (isDelaying) {
      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Authenticate] Authentication blocked - delay active');
      }
      return;
    }

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Authenticate] Starting authentication', {
        deviceId,
        host,
        port: portNumber,
        codeLength: connectionCode.length,
        note: 'Connection code value NEVER logged (FR-029)',
      });
    }

    try {
      clearError();

      // Haptic feedback for authentication start (T129)
      await authenticationStartFeedback();

      // Perform SRP-6a authentication
      await authenticate(host, portNumber, connectionCode);

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Authenticate] Authentication successful', {
          deviceId,
          note: 'Session token stored securely, NEVER logged (FR-029)',
        });
      }

      // Clear connection code from memory (FR-036)
      setConnectionCode('');

      // Reset failure count on success
      setFailureCount(0);

      // Haptic feedback for success (T129)
      await authenticationSuccessFeedback();

      // Show success message
      setSnackbarMessage('Authentication successful!');
      setShowSnackbar(true);

      // Navigate to device detail screen (T076)
      setTimeout(() => {
        router.replace(`/device/${deviceId}` as any);
      }, 500);
    } catch (err) {
      if (__DEV__) {
        console.error('[Authenticate] Authentication failed', {
          error: err instanceof Error ? err.message : 'Unknown error',
          deviceId,
        });
      }

      // Clear connection code from memory on failure (FR-036)
      setConnectionCode('');

      // Increment failure count
      const newFailureCount = failureCount + 1;
      setFailureCount(newFailureCount);

      // Haptic feedback for failure (T129)
      await authenticationFailureFeedback();

      // Apply progressive delay (FR-038)
      applyFailureDelay(newFailureCount);

      // Show error snackbar
      const errorMessage = err instanceof Error ? err.message : 'Authentication failed';
      setSnackbarMessage(errorMessage);
      setShowSnackbar(true);
    }
  };

  /**
   * Handle QR code scanned
   */
  const handleCodeScanned = async (code: string) => {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Authenticate] QR code scanned', {
        codeLength: code.length,
        note: 'Code value NEVER logged (FR-029)',
      });
    }

    // Haptic feedback for QR code scanned (T129)
    await qrCodeScannedFeedback();

    setConnectionCode(code);
    setAuthMode('manual');

    // Auto-submit after QR scan
    setTimeout(() => {
      handleAuthenticate();
    }, 300);
  };

  /**
   * Handle camera permission denied
   */
  const handlePermissionDenied = () => {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Authenticate] Camera permission denied - falling back to manual entry');
    }

    setAuthMode('manual');
    setSnackbarMessage('Camera permission denied. Please use manual entry.');
    setShowSnackbar(true);
  };

  /**
   * Handle mode change
   */
  const handleModeChange = async (value: string) => {
    const newMode = value as AuthMode;

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Authenticate] Auth mode changed', {
        from: authMode,
        to: newMode,
      });
    }

    // Haptic feedback for mode selection (T129)
    await HapticFeedback.selection();

    setAuthMode(newMode);
  };

  /**
   * Format delay message
   */
  const getDelayMessage = (): string => {
    if (failureCount >= LOCKOUT_THRESHOLD) {
      return `Too many failed attempts. Please wait ${delayRemaining}s before trying again.`;
    }
    return `Please wait ${delayRemaining}s before retrying.`;
  };

  /**
   * Render QR scanner mode
   */
  if (authMode === 'qr') {
    return (
      <QRScanner
        onCodeScanned={handleCodeScanned}
        onClose={() => setAuthMode('manual')}
        onPermissionDenied={handlePermissionDenied}
      />
    );
  }

  /**
   * Render manual entry mode
   */
  return (
    <View style={styles.container}>
      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
      >
        <Surface style={styles.surface} elevation={2}>
          {/* Device information */}
          <View style={styles.deviceInfo}>
            <Text variant="headlineSmall" style={styles.deviceName}>
              {deviceName}
            </Text>
            <Text variant="bodyMedium" style={styles.deviceHost}>
              {host}:{portNumber}
            </Text>
          </View>

          {/* Mode selector (T072) */}
          <SegmentedButtons
            value={authMode}
            onValueChange={handleModeChange}
            buttons={[
              {
                value: 'manual',
                label: 'Manual Entry',
                icon: 'keyboard',
              },
              {
                value: 'qr',
                label: 'Scan QR Code',
                icon: 'qrcode-scan',
              },
            ]}
            style={styles.modeSelector}
          />

          {/* Instructions */}
          <Text variant="bodyMedium" style={styles.instructions}>
            Enter the connection code displayed on the device, or scan the QR code for faster
            authentication.
          </Text>

          {/* Connection code input (T069, T071) */}
          <ConnectionCodeInput
            value={connectionCode}
            onChangeText={setConnectionCode}
            onSubmit={handleAuthenticate}
            error={error?.message}
            disabled={isAuthenticating || isDelaying}
            liveValidation={false}
          />

          {/* Delay message (T074) */}
          {isDelaying && (
            <Surface style={styles.delayBanner} elevation={0}>
              <Text variant="bodyMedium" style={styles.delayText}>
                {getDelayMessage()}
              </Text>
            </Surface>
          )}

          {/* Authenticate button */}
          <Button
            mode="contained"
            onPress={handleAuthenticate}
            loading={isAuthenticating}
            disabled={isAuthenticating || isDelaying || !connectionCode}
            style={styles.authenticateButton}
            contentStyle={styles.authenticateButtonContent}
          >
            {isAuthenticating ? 'Authenticating...' : 'Authenticate'}
          </Button>

          {/* Loading indicator */}
          {isAuthenticating && (
            <View style={styles.loadingContainer}>
              <ActivityIndicator animating={true} size="large" />
              <Text variant="bodyMedium" style={styles.loadingText}>
                Performing secure authentication...
              </Text>
            </View>
          )}
        </Surface>
      </ScrollView>

      {/* Error/Success snackbar (T075) */}
      <Snackbar
        visible={showSnackbar}
        onDismiss={() => setShowSnackbar(false)}
        duration={4000}
        action={{
          label: 'Dismiss',
          onPress: () => setShowSnackbar(false),
        }}
      >
        {snackbarMessage}
      </Snackbar>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: theme.colors.background,
  },
  scrollView: {
    flex: 1,
  },
  scrollContent: {
    padding: spacing.md,
  },
  surface: {
    padding: spacing.lg,
    borderRadius: 12,
  },
  deviceInfo: {
    marginBottom: spacing.lg,
    alignItems: 'center',
  },
  deviceName: {
    fontWeight: 'bold',
    marginBottom: spacing.xs,
  },
  deviceHost: {
    color: theme.colors.onSurfaceVariant,
  },
  modeSelector: {
    marginBottom: spacing.lg,
  },
  instructions: {
    marginBottom: spacing.lg,
    textAlign: 'center',
    color: theme.colors.onSurfaceVariant,
  },
  delayBanner: {
    marginTop: spacing.md,
    marginBottom: spacing.md,
    padding: spacing.md,
    borderRadius: 8,
    backgroundColor: theme.colors.errorContainer,
  },
  delayText: {
    color: theme.colors.onErrorContainer,
    textAlign: 'center',
    fontWeight: '500',
  },
  authenticateButton: {
    marginTop: spacing.lg,
  },
  authenticateButtonContent: {
    paddingVertical: spacing.sm,
  },
  loadingContainer: {
    marginTop: spacing.lg,
    alignItems: 'center',
  },
  loadingText: {
    marginTop: spacing.md,
    color: theme.colors.onSurfaceVariant,
  },
});
