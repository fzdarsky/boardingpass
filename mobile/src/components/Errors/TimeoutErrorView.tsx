/**
 * Timeout Error View Component
 * Displays when requests exceed timeout thresholds
 * Implements FR-028 (timeout handling), FR-025 (retry mechanism)
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Button, Text, Icon } from 'react-native-paper';
import { AppError, getErrorMessage, getErrorHelpText } from '../../utils/error-messages';

interface TimeoutErrorViewProps {
  error: AppError;
  onRetry?: () => void;
  onCancel?: () => void;
  operation?: string;
}

export const TimeoutErrorView: React.FC<TimeoutErrorViewProps> = ({
  error,
  onRetry,
  onCancel,
  operation,
}) => {
  const errorMessage = getErrorMessage(error);
  const helpText = getErrorHelpText(error);

  const operationName =
    operation || error.context?.operation || 'request';

  return (
    <View style={styles.container}>
      <View style={styles.iconContainer}>
        <Icon source="clock-alert" size={64} color="#FFA726" />
      </View>

      <Text variant="headlineSmall" style={styles.title}>
        Request Timed Out
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

      {/* Timeout-specific guidance */}
      <View style={styles.guidanceBox}>
        <Text variant="titleSmall" style={styles.guidanceTitle}>
          What can cause timeouts?
        </Text>
        <Text variant="bodySmall" style={styles.guidanceItem}>
          • Slow or unstable network connection
        </Text>
        <Text variant="bodySmall" style={styles.guidanceItem}>
          • Device is busy processing other tasks
        </Text>
        <Text variant="bodySmall" style={styles.guidanceItem}>
          • Weak WiFi signal
        </Text>
      </View>

      <View style={styles.actions}>
        {onRetry && (
          <Button
            mode="contained"
            onPress={onRetry}
            style={styles.primaryButton}
            icon="refresh"
          >
            Try Again
          </Button>
        )}

        {onCancel && (
          <Button
            mode="outlined"
            onPress={onCancel}
            style={styles.secondaryButton}
          >
            Cancel
          </Button>
        )}
      </View>

      <Text variant="bodySmall" style={styles.footerText}>
        The {operationName} took longer than expected
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
  guidanceBox: {
    backgroundColor: '#FFF8E1',
    padding: 16,
    borderRadius: 8,
    marginBottom: 24,
    width: '100%',
  },
  guidanceTitle: {
    fontWeight: 'bold',
    marginBottom: 8,
    color: '#F57F17',
  },
  guidanceItem: {
    marginBottom: 4,
    color: '#F9A825',
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

export default TimeoutErrorView;
