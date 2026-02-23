/**
 * QRScanner Component
 *
 * Camera-based QR code scanner for capturing connection codes.
 * Handles camera permissions, scanning UI, and QR code validation.
 *
 * Features:
 * - Camera permission handling with rationale (FR-026)
 * - QR code format validation (FR-027)
 * - Clear error messages (FR-024)
 * - Accessible UI with clear visual feedback
 *
 * Usage:
 * ```tsx
 * <QRScanner
 *   onCodeScanned={(code) => handleCode(code)}
 *   onClose={() => setShowScanner(false)}
 *   onPermissionDenied={() => handlePermissionDenied()}
 * />
 * ```
 */

import React, { useState, useEffect } from 'react';
import { StyleSheet, View, Linking, Platform } from 'react-native';
import { CameraView, useCameraPermissions } from 'expo-camera';
import type { BarcodeScanningResult } from 'expo-camera';
import { Button, Text, Surface, IconButton, Portal, Dialog } from 'react-native-paper';
import { spacing, theme } from '../../theme';
import { isValidConnectionCode } from '../../utils/validation';

/**
 * QRScanner Props
 */
export interface QRScannerProps {
  /**
   * Callback when a valid QR code is scanned
   * @param code - The scanned connection code
   */
  onCodeScanned: (code: string) => void;

  /**
   * Callback when scanner is closed
   */
  onClose: () => void;

  /**
   * Callback when camera permission is denied
   */
  onPermissionDenied?: () => void;
}

/**
 * QRScanner Component
 */
export default function QRScanner({
  onCodeScanned,
  onClose,
  onPermissionDenied,
}: QRScannerProps): React.ReactElement {
  const [permission, requestPermission] = useCameraPermissions();
  const [scanned, setScanned] = useState(false);
  const [showRationale, setShowRationale] = useState(false);
  const [showPermissionDialog, setShowPermissionDialog] = useState(false);

  /**
   * Handle permission changes
   */
  useEffect(() => {
    if (!permission) {
      // Permission is still loading
      return;
    }

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[QRScanner] Permission status:', {
        granted: permission.granted,
        canAskAgain: permission.canAskAgain,
      });
    }

    if (!permission.granted) {
      // Show rationale dialog if permission was denied but can still ask
      if (permission.canAskAgain) {
        setShowRationale(true);
      } else {
        // User has permanently denied permission - show settings dialog
        setShowPermissionDialog(true);
      }

      // Notify parent component
      onPermissionDenied?.();
    }
  }, [permission, onPermissionDenied]);

  /**
   * Handle barcode scan
   */
  const handleBarcodeScanned = ({ type, data }: BarcodeScanningResult) => {
    if (scanned) {
      return; // Prevent multiple scans
    }

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[QRScanner] Code scanned', {
        type,
        dataLength: data.length,
        note: 'Code value NEVER logged (FR-029)',
      });
    }

    setScanned(true);

    // Validate QR code format
    const trimmedCode = data.trim();
    if (!isValidConnectionCode(trimmedCode)) {
      if (__DEV__) {
        // eslint-disable-next-line no-console
        console.log('[QRScanner] Invalid QR code format - rejected');
      }

      // Reset scan state to allow retry
      setTimeout(() => setScanned(false), 2000);
      return;
    }

    // Valid code - pass to parent
    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[QRScanner] Valid QR code accepted');
    }

    onCodeScanned(trimmedCode);
  };

  /**
   * Open device settings (iOS/Android)
   */
  const openSettings = async () => {
    try {
      if (Platform.OS === 'ios') {
        await Linking.openURL('app-settings:');
      } else {
        await Linking.openSettings();
      }
    } catch (error) {
      if (__DEV__) {
        console.error('[QRScanner] Failed to open settings:', error);
      }
    }
  };

  /**
   * Retry permission request
   */
  const retryPermission = async () => {
    setShowRationale(false);

    try {
      const result = await requestPermission();

      if (!result.granted) {
        setShowPermissionDialog(true);
        onPermissionDenied?.();
      }
    } catch (error) {
      if (__DEV__) {
        console.error('[QRScanner] Permission retry failed:', error);
      }
      onPermissionDenied?.();
    }
  };

  /**
   * Render permission rationale dialog
   */
  if (showRationale) {
    return (
      <Portal>
        <Dialog visible={true} onDismiss={onClose}>
          <Dialog.Icon icon="camera" />
          <Dialog.Title>Camera Permission Required</Dialog.Title>
          <Dialog.Content>
            <Text variant="bodyMedium">
              BoardingPass needs camera access to scan QR codes for device authentication. This
              allows you to quickly authenticate by scanning the code displayed on the device.
            </Text>
            <Text variant="bodyMedium" style={styles.rationaleNote}>
              You can also enter the connection code manually if you prefer not to grant camera
              access.
            </Text>
          </Dialog.Content>
          <Dialog.Actions>
            <Button onPress={onClose}>Use Manual Entry</Button>
            <Button mode="contained" onPress={retryPermission}>
              Grant Permission
            </Button>
          </Dialog.Actions>
        </Dialog>
      </Portal>
    );
  }

  /**
   * Render permission denied dialog (permanent denial)
   */
  if (showPermissionDialog) {
    return (
      <Portal>
        <Dialog visible={true} onDismiss={onClose}>
          <Dialog.Icon icon="camera-off" />
          <Dialog.Title>Camera Access Denied</Dialog.Title>
          <Dialog.Content>
            <Text variant="bodyMedium">
              Camera access has been denied. To use QR code scanning, you need to enable camera
              permissions in your device settings.
            </Text>
            <Text variant="bodyMedium" style={styles.settingsNote}>
              Go to Settings → BoardingPass → Camera and enable camera access.
            </Text>
          </Dialog.Content>
          <Dialog.Actions>
            <Button onPress={onClose}>Use Manual Entry</Button>
            <Button mode="contained" onPress={openSettings}>
              Open Settings
            </Button>
          </Dialog.Actions>
        </Dialog>
      </Portal>
    );
  }

  /**
   * Loading state while requesting permission
   */
  if (!permission) {
    return (
      <View style={styles.container}>
        <Surface style={styles.loadingContainer} elevation={2}>
          <Text variant="bodyLarge">Requesting camera permission...</Text>
        </Surface>
      </View>
    );
  }

  /**
   * Permission denied - show error
   */
  if (!permission.granted) {
    return (
      <View style={styles.container}>
        <Surface style={styles.errorContainer} elevation={2}>
          <Text variant="headlineSmall" style={styles.errorTitle}>
            Camera Access Required
          </Text>
          <Text variant="bodyMedium" style={styles.errorMessage}>
            Please grant camera permission to scan QR codes.
          </Text>
          <Button mode="contained" onPress={onClose} style={styles.closeButton}>
            Use Manual Entry
          </Button>
        </Surface>
      </View>
    );
  }

  /**
   * Render camera scanner
   */
  return (
    <View style={styles.container}>
      <CameraView
        style={styles.camera}
        facing="back"
        onBarcodeScanned={scanned ? undefined : handleBarcodeScanned}
        barcodeScannerSettings={{
          barcodeTypes: ['qr'],
        }}
      >
        {/* Scanner overlay */}
        <View style={styles.overlay}>
          {/* Top bar with close button */}
          <View style={styles.topBar}>
            <IconButton
              icon="close"
              iconColor="#FFFFFF"
              size={28}
              onPress={onClose}
              accessibilityLabel="Close scanner"
            />
          </View>

          {/* Scanning frame */}
          <View style={styles.scannerFrame}>
            <View style={styles.frameCorner} />
            <View style={[styles.frameCorner, styles.frameCornerTopRight]} />
            <View style={[styles.frameCorner, styles.frameCornerBottomLeft]} />
            <View style={[styles.frameCorner, styles.frameCornerBottomRight]} />
          </View>

          {/* Instructions */}
          <View style={styles.instructions}>
            <Surface style={styles.instructionsSurface} elevation={2}>
              <Text variant="titleMedium" style={styles.instructionsText}>
                {scanned ? 'Code scanned!' : 'Position QR code within the frame'}
              </Text>
            </Surface>
          </View>
        </View>
      </CameraView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#000000',
  },
  camera: {
    flex: 1,
  },
  overlay: {
    flex: 1,
    backgroundColor: 'transparent',
  },
  topBar: {
    paddingTop: spacing.xl,
    paddingHorizontal: spacing.md,
    alignItems: 'flex-end',
  },
  scannerFrame: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    position: 'relative',
  },
  frameCorner: {
    position: 'absolute',
    width: 60,
    height: 60,
    borderColor: theme.colors.primary,
    borderTopWidth: 4,
    borderLeftWidth: 4,
    top: -120,
    left: -120,
  },
  frameCornerTopRight: {
    borderTopWidth: 4,
    borderLeftWidth: 0,
    borderRightWidth: 4,
    left: 120,
  },
  frameCornerBottomLeft: {
    borderTopWidth: 0,
    borderBottomWidth: 4,
    top: 120,
  },
  frameCornerBottomRight: {
    borderTopWidth: 0,
    borderLeftWidth: 0,
    borderBottomWidth: 4,
    borderRightWidth: 4,
    top: 120,
    left: 120,
  },
  instructions: {
    paddingBottom: spacing.xxl,
    paddingHorizontal: spacing.md,
  },
  instructionsSurface: {
    padding: spacing.md,
    borderRadius: 12,
    backgroundColor: 'rgba(255, 255, 255, 0.95)',
  },
  instructionsText: {
    textAlign: 'center',
    color: theme.colors.onSurface,
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.xl,
  },
  errorContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.xl,
    margin: spacing.md,
    borderRadius: 12,
  },
  errorTitle: {
    marginBottom: spacing.md,
    textAlign: 'center',
  },
  errorMessage: {
    marginBottom: spacing.lg,
    textAlign: 'center',
  },
  closeButton: {
    marginTop: spacing.md,
  },
  rationaleNote: {
    marginTop: spacing.md,
    fontStyle: 'italic',
  },
  settingsNote: {
    marginTop: spacing.md,
    fontWeight: '500',
  },
});
