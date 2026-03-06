/**
 * ReviewApplyPage Component
 *
 * Step 6 of the wizard: displays a numbered list of planned actions and
 * executes them when the user taps Apply (immediate) or Apply & Reboot (deferred).
 * Shows real-time per-action status feedback during execution.
 */

import React, { useEffect, useCallback } from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, Button, ActivityIndicator, useTheme } from 'react-native-paper';
import MaterialCommunityIcons from '@expo/vector-icons/MaterialCommunityIcons';
import { useConfigWizard } from '../../hooks/useConfigWizard';
import type { PlannedAction } from '../../types/wizard';
import type { APIClient } from '../../services/api/client';
import { spacing } from '../../theme';

type TerminalState = 'rebooting' | 'shutting_down' | null;

const RebootIcon = ({ size, color }: { size: number; color: string }) => (
  <MaterialCommunityIcons name="restart" size={size} color={color} />
);

interface ReviewApplyPageProps {
  apiClient?: APIClient;
  onComplete: (terminal: TerminalState) => void;
  onError: (message: string) => void;
}

const CATEGORY_ICONS: Record<PlannedAction['category'], string> = {
  config: 'cog',
  command: 'console',
  check: 'magnify',
  wait: 'clock-outline',
};

function ActionStatusIcon({ action }: { action: PlannedAction }) {
  const theme = useTheme();

  switch (action.status) {
    case 'running':
      return <ActivityIndicator size={16} color={theme.colors.primary} />;
    case 'success':
      return <MaterialCommunityIcons name="check-circle" size={18} color="#2e7d32" />;
    case 'failed':
      return <MaterialCommunityIcons name="close-circle" size={18} color="#c62828" />;
    case 'skipped':
      return <MaterialCommunityIcons name="minus-circle-outline" size={18} color="#9e9e9e" />;
    default:
      return <MaterialCommunityIcons name="circle-outline" size={18} color="#bdbdbd" />;
  }
}

function ActionRow({ action, index }: { action: PlannedAction; index: number }) {
  const theme = useTheme();
  const iconName = CATEGORY_ICONS[action.category] || 'cog';
  const isInfoOnly = action.infoOnly;

  return (
    <View style={styles.actionRow}>
      <Text
        variant="bodySmall"
        style={[styles.actionIndex, { color: theme.colors.onSurfaceVariant }]}
      >
        {index + 1}.
      </Text>
      <MaterialCommunityIcons
        name={iconName as keyof typeof MaterialCommunityIcons.glyphMap}
        size={18}
        color={isInfoOnly ? '#9e9e9e' : theme.colors.onSurfaceVariant}
        style={styles.categoryIcon}
      />
      <View style={styles.actionContent}>
        <Text
          variant="bodyMedium"
          style={[
            { color: isInfoOnly ? theme.colors.onSurfaceVariant : theme.colors.onSurface },
            isInfoOnly && styles.infoOnlyText,
          ]}
        >
          {action.description}
        </Text>
        {action.detail && action.status !== 'pending' && (
          <Text
            variant="bodySmall"
            style={[
              styles.actionDetail,
              { color: action.status === 'failed' ? '#c62828' : theme.colors.onSurfaceVariant },
            ]}
          >
            {action.detail}
          </Text>
        )}
      </View>
      <View style={styles.statusIcon}>{!isInfoOnly && <ActionStatusIcon action={action} />}</View>
    </View>
  );
}

export default function ReviewApplyPage({ apiClient, onComplete, onError }: ReviewApplyPageProps) {
  const wizard = useConfigWizard();
  const theme = useTheme();
  const { state } = wizard;
  const isDeferred = state.applyMode === 'deferred';
  const isApplying = state.applyInProgress;
  const actions = state.actionList;

  // Generate action list when entering the review step
  useEffect(() => {
    wizard.generateActionList();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleApply = useCallback(async () => {
    if (!apiClient) return;

    if (isDeferred) {
      try {
        await wizard.applyDeferred(apiClient);
        onComplete('rebooting');
      } catch (err) {
        onError(err instanceof Error ? err.message : 'Failed to apply configuration');
      }
    } else {
      try {
        await wizard.applyAllImmediate(apiClient);
        onComplete('shutting_down');
      } catch (err) {
        onError(err instanceof Error ? err.message : 'Failed to apply configuration');
      }
    }
  }, [apiClient, isDeferred, wizard, onComplete, onError]);

  const hasFailed = actions.some(a => a.status === 'failed');

  return (
    <View style={styles.container}>
      <Text variant="titleMedium" style={[styles.title, { color: theme.colors.onSurface }]}>
        Review & Apply
      </Text>
      <Text
        variant="bodyMedium"
        style={[styles.description, { color: theme.colors.onSurfaceVariant }]}
      >
        {isDeferred
          ? 'The following changes will be applied when the device reboots.'
          : 'The following actions will be executed in order.'}
      </Text>

      {/* Action list */}
      {actions.map((action, index) => (
        <ActionRow key={action.id} action={action} index={index} />
      ))}

      {/* Buttons */}
      <View style={styles.actions}>
        <Button
          mode="outlined"
          onPress={wizard.goBack}
          disabled={isApplying}
          style={styles.actionButton}
          accessibilityLabel="Go back to Enrollment step"
        >
          Back
        </Button>
        <Button
          mode="contained"
          onPress={handleApply}
          loading={isApplying}
          disabled={isApplying || hasFailed || !apiClient}
          style={styles.actionButton}
          icon={isDeferred ? RebootIcon : undefined}
          accessibilityLabel={
            isDeferred ? 'Apply configuration and reboot device' : 'Apply all configuration actions'
          }
        >
          {isDeferred ? 'Apply & Reboot' : 'Apply'}
        </Button>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    paddingHorizontal: spacing.md,
  },
  title: {
    marginBottom: spacing.sm,
  },
  description: {
    marginBottom: spacing.lg,
  },
  actionRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.xs,
  },
  actionIndex: {
    width: 24,
    textAlign: 'right',
    marginRight: spacing.xs,
    marginTop: 2,
  },
  categoryIcon: {
    marginRight: spacing.sm,
    marginTop: 2,
  },
  actionContent: {
    flex: 1,
  },
  infoOnlyText: {
    fontStyle: 'italic',
  },
  actionDetail: {
    marginTop: 2,
  },
  statusIcon: {
    width: 24,
    alignItems: 'center',
    marginLeft: spacing.sm,
    marginTop: 2,
  },
  actions: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: spacing.lg,
    paddingBottom: spacing.md,
  },
  actionButton: {
    minWidth: 120,
  },
});
