/**
 * ApplyFeedback Component
 *
 * Displays per-step apply status in immediate mode:
 * - Spinner while config is being applied
 * - Success checkmark when done
 * - Error message with retry button on failure
 * - Connectivity test results after Step 3 (addressing)
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, ActivityIndicator, Button, useTheme } from 'react-native-paper';
import MaterialCommunityIcons from '@expo/vector-icons/MaterialCommunityIcons';
import type { ApplyStatus, ConnectivityResult } from '../../types/wizard';
import { spacing } from '../../theme';

interface ApplyFeedbackProps {
  applyStatus: ApplyStatus | undefined;
  onRetry?: () => void;
}

function ConnectivityResults({ result }: { result: ConnectivityResult }) {
  const theme = useTheme();

  const checks = [
    { label: 'IP assigned', passed: result.ipAssigned },
    { label: 'Gateway reachable', passed: result.gatewayReachable },
    { label: 'DNS resolves', passed: result.dnsResolves },
    { label: 'Internet reachable', passed: result.internetReachable },
  ];

  return (
    <View style={styles.connectivityContainer}>
      <Text
        variant="labelLarge"
        style={[styles.connectivityTitle, { color: theme.colors.onSurface }]}
      >
        Connectivity Test
      </Text>
      {checks.map(check => (
        <View key={check.label} style={styles.checkRow}>
          <MaterialCommunityIcons
            name={check.passed ? 'check-circle' : 'close-circle'}
            size={18}
            color={check.passed ? '#4CAF50' : theme.colors.error}
          />
          <Text
            variant="bodyMedium"
            style={[styles.checkLabel, { color: theme.colors.onSurfaceVariant }]}
          >
            {check.label}
          </Text>
        </View>
      ))}
    </View>
  );
}

export default function ApplyFeedback({ applyStatus, onRetry }: ApplyFeedbackProps) {
  const theme = useTheme();

  if (!applyStatus || applyStatus.status === 'pending') {
    return null;
  }

  if (applyStatus.status === 'applying') {
    return (
      <View style={[styles.container, styles.applyingContainer]}>
        <ActivityIndicator size="small" animating={true} />
        <Text
          variant="bodyMedium"
          style={[styles.statusText, { color: theme.colors.onSurfaceVariant }]}
        >
          Applying configuration...
        </Text>
      </View>
    );
  }

  if (applyStatus.status === 'success') {
    return (
      <View style={styles.container}>
        <View style={styles.statusRow}>
          <MaterialCommunityIcons name="check-circle" size={22} color="#4CAF50" />
          <Text variant="bodyMedium" style={[styles.statusText, styles.successText]}>
            Configuration applied successfully
          </Text>
        </View>
        {applyStatus.connectivityResult && (
          <ConnectivityResults result={applyStatus.connectivityResult} />
        )}
      </View>
    );
  }

  // status === 'failed'
  return (
    <View style={styles.container}>
      <View style={styles.statusRow}>
        <MaterialCommunityIcons name="alert-circle" size={22} color={theme.colors.error} />
        <Text variant="bodyMedium" style={[styles.statusText, { color: theme.colors.error }]}>
          Apply failed
        </Text>
      </View>
      {applyStatus.error && (
        <Text
          variant="bodySmall"
          style={[styles.errorDetail, { color: theme.colors.onSurfaceVariant }]}
        >
          {applyStatus.error}
        </Text>
      )}
      {onRetry && (
        <Button
          mode="outlined"
          onPress={onRetry}
          compact
          style={styles.retryButton}
          accessibilityLabel="Retry applying configuration"
        >
          Retry
        </Button>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
  },
  applyingContainer: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  statusRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  statusText: {
    marginLeft: spacing.sm,
  },
  successText: {
    color: '#4CAF50',
  },
  errorDetail: {
    marginTop: spacing.xs,
    marginLeft: 30,
  },
  retryButton: {
    marginTop: spacing.sm,
    alignSelf: 'flex-start',
    marginLeft: 30,
  },
  connectivityContainer: {
    marginTop: spacing.sm,
    marginLeft: 30,
    padding: spacing.sm,
    borderRadius: 8,
  },
  connectivityTitle: {
    marginBottom: spacing.xs,
  },
  checkRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginVertical: 2,
  },
  checkLabel: {
    marginLeft: spacing.sm,
  },
});
