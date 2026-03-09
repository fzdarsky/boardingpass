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

import React, { useState, useEffect, useRef, useLayoutEffect } from 'react';
import { StyleSheet, View, ScrollView, BackHandler, Pressable } from 'react-native';
import { useRouter, useLocalSearchParams, useNavigation } from 'expo-router';
import {
  Text,
  Button,
  SegmentedButtons,
  ActivityIndicator,
  Snackbar,
  useTheme,
} from 'react-native-paper';
import MaterialCommunityIcons from '@expo/vector-icons/MaterialCommunityIcons';
import { spacing } from '@/theme';
import { useAuth } from '@/hooks/useAuth';
import { useCertificateValidation } from '@/hooks/useCertificateValidation';
import QRScanner from '@/components/QRScanner';
import ConnectionCodeInput from '@/components/ConnectionCodeInput';
import { CertificateInfoDisplay } from '@/components/CertificateInfo';
import {
  authenticationStartFeedback,
  authenticationSuccessFeedback,
  authenticationFailureFeedback,
  qrCodeScannedFeedback,
  HapticFeedback,
} from '@/utils/haptics';
import { isTLSOverrideActive, invalidateHTTPSession } from '../../modules/certificate-pinning';

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

  const navigation = useNavigation();
  const paperTheme = useTheme();

  // Custom back icon matching device detail header style
  useLayoutEffect(() => {
    navigation.setOptions({
      // eslint-disable-next-line react/no-unstable-nested-components
      headerLeft: () => (
        <Pressable
          onPress={() => router.back()}
          style={styles.headerIcon}
          accessibilityLabel="Go back"
        >
          <MaterialCommunityIcons name="arrow-left" size={22} color={paperTheme.colors.onPrimary} />
        </Pressable>
      ),
    });
  }, [navigation, router, paperTheme.colors.onPrimary]);

  const { authenticate, isAuthenticating, error, clearError } = useAuth(deviceId);

  // Certificate validation (TOFU)
  const {
    isValidating: isCertValidating,
    certificate,
    requiresTrust,
    error: certError,
    validateCertificate,
    trustCertificate,
  } = useCertificateValidation();

  // Whether the certificate has been validated and trusted
  const [certReady, setCertReady] = useState(false);

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
   * Log authentication screen mount and run TOFU certificate check
   */
  useEffect(() => {
    // Check TLS override status — critical for self-signed certificate support
    try {
      const tlsActive = isTLSOverrideActive();
      // eslint-disable-next-line no-console
      console.log('[Authenticate] TLS override active:', tlsActive);
      if (!tlsActive) {
        console.warn(
          '[Authenticate] TLS override NOT active — HTTPS requests to self-signed servers will fail'
        );
      }
    } catch (e) {
      console.warn('[Authenticate] Failed to check TLS override status:', e);
    }

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

    // Run TOFU certificate validation on mount (best-effort, non-blocking)
    if (host) {
      validateCertificate(deviceId, host, portNumber)
        .then(result => {
          if (result.isValid && !result.requiresUserTrust) {
            // Certificate is already trusted (pinned or CA-signed)
            setCertReady(true);
            if (__DEV__) {
              // eslint-disable-next-line no-console
              console.log('[Authenticate] Certificate already trusted', {
                fingerprint: result.certificate.fingerprint.slice(0, 16) + '...',
                trustStatus: result.certificate.trustStatus,
              });
            }
          } else if (__DEV__) {
            // eslint-disable-next-line no-console
            console.log('[Authenticate] Certificate requires user trust', {
              trustStatus: result.certificate.trustStatus,
              isSelfSigned: result.certificate.isSelfSigned,
            });
          }
        })
        .catch(err => {
          // Certificate fetch failed (device unreachable, timeout, etc.)
          // Proceed anyway — the TLS override accepts unpinned certs (TOFU).
          setCertReady(true);
          if (__DEV__) {
            console.warn(
              '[Authenticate] Certificate validation failed, proceeding without TOFU',
              err
            );
          }
        });
    } else {
      setCertReady(true);
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [deviceId, deviceName, host, portNumber]);

  /**
   * Handle user trusting the certificate (TOFU)
   */
  const handleTrustCertificate = async () => {
    if (!certificate) return;

    try {
      await trustCertificate(certificate, host, portNumber);

      // Invalidate RCTHTTPRequestHandler's cached NSURLSession so the next
      // HTTPS request creates a fresh session that recognises our TOFU
      // challenge handler and the newly pinned fingerprint.
      try {
        const sessionResult = invalidateHTTPSession();
        if (__DEV__) {
          // eslint-disable-next-line no-console
          console.log('[Authenticate] HTTP session invalidated:', JSON.stringify(sessionResult));
        }
      } catch (e) {
        console.warn('[Authenticate] Failed to invalidate HTTP session:', e);
      }

      setCertReady(true);

      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[Authenticate] Certificate trusted and pinned', {
          fingerprint: certificate.fingerprint.slice(0, 16) + '...',
          host,
          port: portNumber,
        });
      }
    } catch (err) {
      if (__DEV__) {
        console.error('[Authenticate] Failed to trust certificate', err);
      }
      setSnackbarMessage('Failed to trust certificate');
      setShowSnackbar(true);
    }
  };

  /**
   * Handle user rejecting the certificate
   */
  const handleRejectCertificate = () => {
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[Authenticate] Certificate rejected by user');
    }
    router.back();
  };

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
  const handleAuthenticate = async (codeOverride?: string) => {
    const code = codeOverride || connectionCode;

    if (!code || code.trim().length === 0) {
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
        codeLength: code.length,
        note: 'Connection code value NEVER logged (FR-029)',
      });
    }

    try {
      clearError();

      // Haptic feedback for authentication start (T129)
      await authenticationStartFeedback();

      // Perform SRP-6a authentication
      // (requests route through nativeAdapter on iOS, which uses its own
      // URLSession with TOFU delegate for self-signed certificate support)
      await authenticate(host, portNumber, code);

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
      // Token is retrieved from secure storage by the detail screen — never passed via URL params
      setTimeout(() => {
        /* eslint-disable @typescript-eslint/no-explicit-any */
        router.replace({
          pathname: '/device/[id]',
          params: {
            id: deviceId,
            host,
            port: String(portNumber),
          },
        } as any);
        /* eslint-enable @typescript-eslint/no-explicit-any */
      }, 500);
    } catch (err: unknown) {
      const errObj = err instanceof Error ? err : new Error(String(err));
      if (__DEV__) {
        console.error('[Authenticate] Authentication failed', {
          message: errObj.message,
          code: (err as { code?: string }).code,
          type: (err as { type?: string }).type,
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

      // Show error snackbar — now includes specific error details
      setSnackbarMessage(errObj.message || 'Authentication failed');
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

    // Auto-submit after QR scan — pass code directly to avoid stale closure
    setTimeout(() => {
      handleAuthenticate(code);
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
   * Render certificate validation loading state
   */
  if (isCertValidating) {
    return (
      <View
        style={[
          styles.container,
          styles.centerContent,
          { backgroundColor: paperTheme.colors.background },
        ]}
      >
        <ActivityIndicator animating={true} size="large" />
        <Text
          variant="bodyMedium"
          style={[styles.loadingText, { color: paperTheme.colors.onSurfaceVariant }]}
        >
          Verifying device certificate...
        </Text>
      </View>
    );
  }

  /**
   * Render certificate error state
   */
  if (certError && !requiresTrust && !certReady) {
    return (
      <View
        style={[
          styles.container,
          styles.centerContent,
          { backgroundColor: paperTheme.colors.background },
        ]}
      >
        <Text variant="headlineSmall" style={styles.deviceName}>
          Connection Failed
        </Text>
        <Text
          variant="bodyMedium"
          style={[
            styles.instructions,
            { marginTop: spacing.md, color: paperTheme.colors.onSurfaceVariant },
          ]}
        >
          {certError}
        </Text>
        <Button mode="contained" onPress={() => router.back()} style={styles.authenticateButton}>
          Go Back
        </Button>
      </View>
    );
  }

  /**
   * Render certificate trust screen (TOFU)
   * Rendered inline (not as a Portal dialog) so it stays within the navigation stack.
   */
  if (requiresTrust && certificate) {
    const isCertChanged = certificate.trustStatus === 'changed';

    return (
      <View style={[styles.container, { backgroundColor: paperTheme.colors.background }]}>
        <ScrollView style={styles.scrollView} contentContainerStyle={styles.trustContent}>
          {/* Header */}
          <View style={styles.trustHeader}>
            <Text
              variant="headlineSmall"
              style={
                isCertChanged
                  ? [styles.errorText, { color: paperTheme.colors.error }]
                  : styles.deviceName
              }
            >
              {isCertChanged ? 'Certificate Changed' : 'Trust Certificate?'}
            </Text>
          </View>

          {/* Warning for changed certificates */}
          {isCertChanged && (
            <View
              style={[styles.warningBox, { backgroundColor: paperTheme.colors.errorContainer }]}
            >
              <Text
                variant="bodyMedium"
                style={[styles.warningText, { color: paperTheme.colors.onErrorContainer }]}
              >
                The certificate for this device has changed since your last connection. This could
                indicate a security issue or that the device was reconfigured.
              </Text>
            </View>
          )}

          {/* Info for new self-signed certificates */}
          {!isCertChanged && certificate.isSelfSigned && (
            <View
              style={[styles.infoBox, { backgroundColor: paperTheme.colors.secondaryContainer }]}
            >
              <Text
                variant="bodyMedium"
                style={[styles.infoText, { color: paperTheme.colors.onSecondaryContainer }]}
              >
                This device uses a self-signed certificate. Verify the fingerprint matches the
                device before trusting it.
              </Text>
            </View>
          )}

          {/* Certificate details */}
          <CertificateInfoDisplay certificate={certificate} compact={false} />

          {/* Action buttons */}
          <View style={styles.trustActions}>
            <Button mode="outlined" onPress={handleRejectCertificate} style={styles.trustButton}>
              {isCertChanged ? 'Reject' : 'Cancel'}
            </Button>
            <Button
              mode="contained"
              onPress={handleTrustCertificate}
              style={styles.trustButton}
              buttonColor={isCertChanged ? paperTheme.colors.error : undefined}
            >
              {isCertChanged ? 'Trust Anyway' : 'Trust Certificate'}
            </Button>
          </View>
        </ScrollView>
      </View>
    );
  }

  /**
   * Render QR scanner mode (only after certificate is trusted)
   */
  if (certReady && authMode === 'qr') {
    return (
      <QRScanner
        onCodeScanned={handleCodeScanned}
        onClose={() => setAuthMode('manual')}
        onPermissionDenied={handlePermissionDenied}
      />
    );
  }

  /**
   * Render manual entry mode (only after certificate is trusted)
   */
  return (
    <View style={[styles.container, { backgroundColor: paperTheme.colors.background }]}>
      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
      >
        {/* Device information */}
        <View style={styles.deviceInfo}>
          <Text variant="headlineSmall" style={styles.deviceName}>
            {deviceName}
          </Text>
          <Text
            variant="bodyMedium"
            style={[styles.deviceHost, { color: paperTheme.colors.onSurfaceVariant }]}
          >
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
              label: 'Scan Code',
              icon: 'barcode-scan',
            },
          ]}
          style={styles.modeSelector}
        />

        {/* Instructions */}
        <Text
          variant="bodyMedium"
          style={[styles.instructions, { color: paperTheme.colors.onSurfaceVariant }]}
        >
          Enter the connection code displayed on the device, or scan a code for faster
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
          <View style={[styles.delayBanner, { backgroundColor: paperTheme.colors.errorContainer }]}>
            <Text
              variant="bodyMedium"
              style={[styles.delayText, { color: paperTheme.colors.onErrorContainer }]}
            >
              {getDelayMessage()}
            </Text>
          </View>
        )}

        {/* Authenticate button */}
        <Button
          mode="contained"
          onPress={() => handleAuthenticate()}
          loading={isAuthenticating}
          disabled={isAuthenticating || isDelaying || !connectionCode}
          style={styles.authenticateButton}
          contentStyle={styles.authenticateButtonContent}
        >
          {isAuthenticating ? 'Authenticating...' : 'Authenticate'}
        </Button>
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
  },
  centerContent: {
    justifyContent: 'center' as const,
    alignItems: 'center' as const,
    padding: spacing.lg,
  },
  scrollView: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
    justifyContent: 'center' as const,
    padding: spacing.lg,
  },
  deviceInfo: {
    marginBottom: spacing.lg,
    alignItems: 'center',
  },
  deviceName: {
    fontWeight: 'bold',
    marginBottom: spacing.xs,
  },
  deviceHost: {},
  modeSelector: {
    marginBottom: spacing.lg,
  },
  instructions: {
    marginBottom: spacing.lg,
    textAlign: 'center',
  },
  delayBanner: {
    marginTop: spacing.md,
    marginBottom: spacing.md,
    padding: spacing.md,
    borderRadius: 8,
  },
  delayText: {
    textAlign: 'center',
    fontWeight: '500',
  },
  authenticateButton: {
    marginTop: spacing.lg,
  },
  authenticateButtonContent: {
    paddingVertical: spacing.sm,
  },
  loadingText: {
    marginTop: spacing.md,
  },
  trustContent: {
    flexGrow: 1,
    padding: spacing.lg,
  },
  trustHeader: {
    alignItems: 'center',
    marginBottom: spacing.lg,
  },
  errorText: {
    fontWeight: 'bold',
  },
  warningBox: {
    padding: spacing.md,
    borderRadius: 8,
    marginBottom: spacing.md,
  },
  warningText: {
    fontWeight: '600',
  },
  infoBox: {
    padding: spacing.md,
    borderRadius: 8,
    marginBottom: spacing.md,
  },
  infoText: {},
  trustActions: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    gap: spacing.sm,
    marginTop: spacing.lg,
  },
  trustButton: {
    flex: 1,
  },
  headerIcon: {
    padding: 4,
  },
});
