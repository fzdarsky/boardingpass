/**
 * Network Error View Component
 * Displays when network connectivity issues occur
 * Implements FR-024 (error messages), FR-025 (retry mechanism)
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Button, Text, Icon } from 'react-native-paper';
import { AppError, getErrorMessage, getErrorHelpText } from '../../utils/error-messages';

interface NetworkErrorViewProps {
  error: AppError;
  onRetry?: () => void;
  onCancel?: () => void;
  deviceName?: string;
}

export const NetworkErrorView: React.FC<NetworkErrorViewProps> = ({
  error,
  onRetry,
  onCancel,
  deviceName,
}) => {
  const errorMessage = getErrorMessage(error);
  const helpText = getErrorHelpText(error);
  const isDeviceUnreachable = error.code === 'ECONNREFUSED' || error.code === 'ECONNRESET';

  return (
    <View style={styles.container}>
      <View style={styles.iconContainer}>
        <Icon
          source={isDeviceUnreachable ? 'access-point-network-off' : 'wifi-off'}
          size={64}
          color="#FF9800"
        />
      </View>

      <Text variant="headlineSmall" style={styles.title}>
        Connection Problem
      </Text>

      <Text variant="bodyMedium" style={styles.message}>
        {errorMessage}
      </Text>

      {helpText && (
        <View style={styles.helpBox}>
          <Icon source="information" size={20} color="#4A90E2" />
          <Text variant="bodySmall" style={styles.helpText}>
            {helpText}
          </Text>
        </View>
      )}

      {/* Device-specific troubleshooting tips */}
      {isDeviceUnreachable && deviceName && (
        <View style={styles.tipsBox}>
          <Text variant="titleSmall" style={styles.tipsTitle}>
            Troubleshooting Tips:
          </Text>
          <Text variant="bodySmall" style={styles.tipItem}>
            • Verify {deviceName} is powered on
          </Text>
          <Text variant="bodySmall" style={styles.tipItem}>
            • Check both devices are on the same network
          </Text>
          <Text variant="bodySmall" style={styles.tipItem}>
            • Ensure no firewall is blocking the connection
          </Text>
        </View>
      )}

      <View style={styles.actions}>
        {onRetry && (
          <Button mode="contained" onPress={onRetry} style={styles.primaryButton} icon="refresh">
            Try Again
          </Button>
        )}

        {onCancel && (
          <Button mode="outlined" onPress={onCancel} style={styles.secondaryButton}>
            Cancel
          </Button>
        )}
      </View>

      <Text variant="bodySmall" style={styles.footerText}>
        Check your network connection and try again
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
    marginBottom: 16,
    maxWidth: '100%',
  },
  helpText: {
    flex: 1,
    marginLeft: 12,
    color: '#1976D2',
    lineHeight: 20,
  },
  tipsBox: {
    backgroundColor: '#FFF3E0',
    padding: 16,
    borderRadius: 8,
    marginBottom: 24,
    width: '100%',
  },
  tipsTitle: {
    fontWeight: 'bold',
    marginBottom: 8,
    color: '#E65100',
  },
  tipItem: {
    marginBottom: 4,
    color: '#F57C00',
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

export default NetworkErrorView;
