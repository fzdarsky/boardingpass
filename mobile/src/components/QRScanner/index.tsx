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
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { CameraView, useCameraPermissions } from 'expo-camera';
import type { BarcodeScanningResult } from 'expo-camera';
import { Button, Text, IconButton, Portal, Dialog } from 'react-native-paper';
import { spacing } from '../../theme';
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
export default function QRScanner({ onCodeScanned, onClose }: QRScannerProps): React.ReactElement {
  const insets = useSafeAreaInsets();
  const [permission, requestPermission] = useCameraPermissions();
  const [scanned, setScanned] = useState(false);
  const [showPermissionDialog, setShowPermissionDialog] = useState(false);

  /**
   * Request permission automatically on mount.
   * On iOS, the native dialog already shows NSCameraUsageDescription as rationale,
   * so we skip our own rationale dialog and trigger the OS prompt directly.
   * Only show the settings dialog when permission has been permanently denied.
   */
  useEffect(() => {
    if (!permission) return;

    if (__DEV__) {
      // eslint-disable-next-line no-console
      console.log('[QRScanner] Permission status:', {
        granted: permission.granted,
        canAskAgain: permission.canAskAgain,
      });
    }

    if (!permission.granted) {
      if (permission.canAskAgain) {
        // Trigger the native OS permission dialog directly
        requestPermission();
      } else {
        // User has permanently denied permission - show settings dialog
        setShowPermissionDialog(true);
      }
    }
  }, [permission, requestPermission]);

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
        data, // TODO: remove before committing — temporary debug aid
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
   * Render permission denied dialog (permanent denial)
   */
  if (showPermissionDialog) {
    return (
      <View style={styles.container}>
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
      </View>
    );
  }

  /**
   * Loading state while requesting permission
   */
  if (!permission) {
    return (
      <View style={styles.container}>
        <View style={styles.loadingContainer}>
          <Text variant="bodyLarge" style={{ color: '#FFFFFF' }}>
            Requesting camera permission...
          </Text>
        </View>
      </View>
    );
  }

  /**
   * Permission denied - show error
   */
  if (!permission.granted) {
    return (
      <View style={styles.container}>
        <View style={styles.errorContainer}>
          <Text variant="headlineSmall" style={styles.errorTitle}>
            Camera Access Required
          </Text>
          <Text variant="bodyMedium" style={styles.errorMessage}>
            Please grant camera permission to scan QR codes.
          </Text>
          <Button mode="contained" onPress={onClose} style={styles.closeButton}>
            Use Manual Entry
          </Button>
        </View>
      </View>
    );
  }

  /**
   * Render camera scanner
   */
  return (
    <View style={styles.container}>
      <CameraView
        style={StyleSheet.absoluteFill}
        facing="back"
        onBarcodeScanned={scanned ? undefined : handleBarcodeScanned}
        barcodeScannerSettings={{
          barcodeTypes: [
            'qr',
            'ean13',
            'ean8',
            'code128',
            'code39',
            'upc_a',
            'upc_e',
            'datamatrix',
            'pdf417',
          ],
        }}
      />

      {/* Scanner overlay — sibling of CameraView, not a child */}
      <View style={StyleSheet.absoluteFill} pointerEvents="box-none">
        {/* Top bar with close button */}
        <View style={[styles.topBar, { paddingTop: insets.top + spacing.sm }]}>
          <IconButton
            icon="close"
            iconColor="#FFFFFF"
            size={28}
            onPress={onClose}
            accessibilityLabel="Close scanner"
          />
        </View>

        {/* Scanning frame — centered box with corner brackets */}
        <View style={styles.scannerFrame} pointerEvents="none">
          <View style={styles.frameBox}>
            <View style={[styles.frameCorner, styles.frameCornerTL]} />
            <View style={[styles.frameCorner, styles.frameCornerTR]} />
            <View style={[styles.frameCorner, styles.frameCornerBL]} />
            <View style={[styles.frameCorner, styles.frameCornerBR]} />
          </View>
        </View>

        {/* Instructions */}
        <View style={styles.instructions} pointerEvents="none">
          <View style={styles.instructionsBubble}>
            <Text variant="titleMedium" style={styles.instructionsText}>
              {scanned ? 'Code scanned!' : 'Point camera at a QR code or barcode'}
            </Text>
          </View>
        </View>
      </View>
    </View>
  );
}

const FRAME_SIZE = 260;
const CORNER_SIZE = 40;
const CORNER_WIDTH = 4;

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#000000',
  },
  topBar: {
    paddingHorizontal: spacing.md,
    alignItems: 'flex-end',
  },
  scannerFrame: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  frameBox: {
    width: FRAME_SIZE,
    height: FRAME_SIZE,
  },
  frameCorner: {
    position: 'absolute',
    width: CORNER_SIZE,
    height: CORNER_SIZE,
    borderColor: '#FFFFFF',
  },
  frameCornerTL: {
    top: 0,
    left: 0,
    borderTopWidth: CORNER_WIDTH,
    borderLeftWidth: CORNER_WIDTH,
  },
  frameCornerTR: {
    top: 0,
    right: 0,
    borderTopWidth: CORNER_WIDTH,
    borderRightWidth: CORNER_WIDTH,
  },
  frameCornerBL: {
    bottom: 0,
    left: 0,
    borderBottomWidth: CORNER_WIDTH,
    borderLeftWidth: CORNER_WIDTH,
  },
  frameCornerBR: {
    bottom: 0,
    right: 0,
    borderBottomWidth: CORNER_WIDTH,
    borderRightWidth: CORNER_WIDTH,
  },
  instructions: {
    paddingBottom: spacing.xxl,
    paddingHorizontal: spacing.md,
  },
  instructionsBubble: {
    padding: spacing.md,
    borderRadius: 12,
    backgroundColor: 'rgba(0, 0, 0, 0.6)',
  },
  instructionsText: {
    textAlign: 'center',
    color: '#FFFFFF',
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.xl,
    backgroundColor: '#000000',
  },
  errorContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.xl,
    margin: spacing.md,
    borderRadius: 12,
    backgroundColor: '#000000',
  },
  errorTitle: {
    marginBottom: spacing.md,
    textAlign: 'center',
    color: '#FFFFFF',
  },
  errorMessage: {
    marginBottom: spacing.lg,
    textAlign: 'center',
    color: '#CCCCCC',
  },
  closeButton: {
    marginTop: spacing.md,
  },
  settingsNote: {
    marginTop: spacing.md,
    fontWeight: '500',
  },
});
