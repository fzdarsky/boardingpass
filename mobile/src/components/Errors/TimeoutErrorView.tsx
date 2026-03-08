/**
 * Timeout Error View Component
 * Displays when requests exceed timeout thresholds
 * Implements FR-028 (timeout handling), FR-025 (retry mechanism)
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Button, Text, Icon, useTheme } from 'react-native-paper';
import { AppError, getErrorMessage, getErrorHelpText } from '../../utils/error-messages';
import { statusColors } from '../../theme';

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
  const theme = useTheme();
  const errorMessage = getErrorMessage(error);
  const helpText = getErrorHelpText(error);

  const operationName = operation || error.context?.operation || 'request';

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.surface }]}>
      <View style={styles.iconContainer}>
        <Icon source="clock-alert" size={64} color={statusColors.selfSignedNew} />
      </View>

      <Text variant="headlineSmall" style={[styles.title, { color: theme.colors.onSurface }]}>
        Request Timed Out
      </Text>

      <Text variant="bodyMedium" style={[styles.message, { color: theme.colors.onSurfaceVariant }]}>
        {errorMessage}
      </Text>

      {helpText && (
        <View style={[styles.helpBox, { backgroundColor: theme.colors.secondaryContainer }]}>
          <Icon source="information" size={20} color={theme.colors.primary} />
          <Text variant="bodySmall" style={[styles.helpText, { color: theme.colors.secondary }]}>
            {helpText}
          </Text>
        </View>
      )}

      {/* Timeout-specific guidance */}
      <View style={[styles.guidanceBox, { backgroundColor: theme.colors.tertiaryContainer }]}>
        <Text
          variant="titleSmall"
          style={[styles.guidanceTitle, { color: theme.colors.onTertiaryContainer }]}
        >
          What can cause timeouts?
        </Text>
        <Text
          variant="bodySmall"
          style={[styles.guidanceItem, { color: theme.colors.onTertiaryContainer }]}
        >
          • Slow or unstable network connection
        </Text>
        <Text
          variant="bodySmall"
          style={[styles.guidanceItem, { color: theme.colors.onTertiaryContainer }]}
        >
          • Device is busy processing other tasks
        </Text>
        <Text
          variant="bodySmall"
          style={[styles.guidanceItem, { color: theme.colors.onTertiaryContainer }]}
        >
          • Weak WiFi signal
        </Text>
      </View>

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

      <Text variant="bodySmall" style={[styles.footerText, { color: theme.colors.outline }]}>
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
  },
  iconContainer: {
    marginBottom: 24,
  },
  title: {
    fontWeight: 'bold',
    textAlign: 'center',
    marginBottom: 12,
  },
  message: {
    textAlign: 'center',
    marginBottom: 24,
    lineHeight: 22,
  },
  helpBox: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    padding: 16,
    borderRadius: 8,
    marginBottom: 16,
    maxWidth: '100%',
  },
  helpText: {
    flex: 1,
    marginLeft: 12,
    lineHeight: 20,
  },
  guidanceBox: {
    padding: 16,
    borderRadius: 8,
    marginBottom: 24,
    width: '100%',
  },
  guidanceTitle: {
    fontWeight: 'bold',
    marginBottom: 8,
  },
  guidanceItem: {
    marginBottom: 4,
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
    lineHeight: 18,
  },
});

export default TimeoutErrorView;
