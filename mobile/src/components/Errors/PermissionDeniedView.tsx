/**
 * Permission Denied Error View Component
 * Displays when user denies required permissions (camera, local network, etc.)
 * Implements FR-106 (permission error handling)
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Button, Text, Icon } from 'react-native-paper';
import * as Linking from 'expo-linking';
import { AppError } from '../../utils/error-messages';

interface PermissionDeniedViewProps {
  error: AppError;
  onRetry?: () => void;
  onManualEntry?: () => void;
  canAskAgain?: boolean;
}

export const PermissionDeniedView: React.FC<PermissionDeniedViewProps> = ({
  error,
  onRetry,
  onManualEntry,
  canAskAgain = true,
}) => {
  const permissionType = error.context?.permission || 'permission';
  const isCameraPermission = permissionType === 'camera';

  const handleOpenSettings = async () => {
    try {
      await Linking.openSettings();
    } catch (err) {
      console.error('Failed to open settings:', err);
    }
  };

  return (
    <View style={styles.container}>
      <View style={styles.iconContainer}>
        <Icon source="shield-alert" size={64} color="#FF6B6B" />
      </View>

      <Text variant="headlineSmall" style={styles.title}>
        Permission Required
      </Text>

      <Text variant="bodyMedium" style={styles.message}>
        {error.context?.canAskAgain === false
          ? `${
              permissionType.charAt(0).toUpperCase() + permissionType.slice(1)
            } permission is required. Please enable it in Settings.`
          : `This feature requires ${permissionType} permission to function.`}
      </Text>

      {error.context?.helpText && (
        <View style={styles.helpBox}>
          <Icon source="information" size={20} color="#4A90E2" />
          <Text variant="bodySmall" style={styles.helpText}>
            {error.context.helpText}
          </Text>
        </View>
      )}

      <View style={styles.actions}>
        {/* Show "Open Settings" if permission permanently denied */}
        {!canAskAgain && (
          <Button
            mode="contained"
            onPress={handleOpenSettings}
            style={styles.primaryButton}
            icon="cog"
          >
            Open Settings
          </Button>
        )}

        {/* Show "Grant Permission" if can ask again */}
        {canAskAgain && onRetry && (
          <Button
            mode="contained"
            onPress={onRetry}
            style={styles.primaryButton}
            icon="shield-check"
          >
            Grant Permission
          </Button>
        )}

        {/* Show "Enter Manually" option for camera permission */}
        {isCameraPermission && onManualEntry && (
          <Button
            mode="outlined"
            onPress={onManualEntry}
            style={styles.secondaryButton}
            icon="keyboard"
          >
            Enter Code Manually
          </Button>
        )}
      </View>

      <Text variant="bodySmall" style={styles.footerText}>
        {isCameraPermission
          ? 'Camera access is required for QR scanning. You can also enter the connection code manually.'
          : `${
              permissionType.charAt(0).toUpperCase() + permissionType.slice(1)
            } access is required for this feature.`}
      </Text>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
    backgroundColor: '#FFFFFF',
  },
  iconContainer: {
    marginBottom: 24,
  },
  title: {
    fontWeight: 'bold',
    textAlign: 'center',
    marginBottom: 12,
    color: '#333333',
  },
  message: {
    textAlign: 'center',
    marginBottom: 24,
    color: '#666666',
    lineHeight: 22,
  },
  helpBox: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    backgroundColor: '#E3F2FD',
    padding: 16,
    borderRadius: 8,
    marginBottom: 24,
    maxWidth: '100%',
  },
  helpText: {
    flex: 1,
    marginLeft: 12,
    color: '#1976D2',
    lineHeight: 20,
  },
  actions: {
    width: '100%',
    gap: 12,
  },
  primaryButton: {
    marginBottom: 8,
  },
  secondaryButton: {
    marginBottom: 8,
  },
  footerText: {
    marginTop: 24,
    textAlign: 'center',
    color: '#999999',
    lineHeight: 18,
  },
});

export default PermissionDeniedView;
