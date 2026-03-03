/**
 * StepIndicator Component
 *
 * Shows current wizard step (1-5), step labels, and completion progress.
 * Supports tapping completed steps for direct navigation.
 */

import React from 'react';
import { View, StyleSheet, Pressable } from 'react-native';
import { Text, useTheme } from 'react-native-paper';
import { STEP_LABELS, TOTAL_STEPS } from '../../types/wizard';
import { spacing } from '../../theme';

interface StepIndicatorProps {
  currentStep: number;
  maxReachedStep: number;
  isStepComplete: (step: number) => boolean;
  onStepPress?: (step: number) => void;
}

export default function StepIndicator({
  currentStep,
  maxReachedStep,
  isStepComplete,
  onStepPress,
}: StepIndicatorProps) {
  const theme = useTheme();

  return (
    <View style={styles.container}>
      <View style={styles.stepsRow}>
        {Array.from({ length: TOTAL_STEPS }, (_, i) => i + 1).map(step => {
          const isCurrent = step === currentStep;
          const isComplete = isStepComplete(step);
          const isReachable = step <= maxReachedStep;

          return (
            <React.Fragment key={step}>
              {step > 1 && (
                <View
                  style={[
                    styles.connector,
                    {
                      backgroundColor:
                        isComplete || step <= currentStep
                          ? theme.colors.primary
                          : theme.colors.outlineVariant,
                    },
                  ]}
                />
              )}
              <Pressable
                onPress={() => isReachable && onStepPress?.(step)}
                disabled={!isReachable || !onStepPress}
                accessibilityLabel={`Step ${step}: ${STEP_LABELS[step]}${isCurrent ? ', current' : ''}${isComplete ? ', complete' : ''}`}
                accessibilityRole="button"
              >
                <View
                  style={[
                    styles.stepCircle,
                    {
                      backgroundColor: isCurrent
                        ? theme.colors.primary
                        : isComplete
                          ? theme.colors.primary
                          : theme.colors.surfaceVariant,
                      borderColor: isCurrent
                        ? theme.colors.primary
                        : isComplete
                          ? theme.colors.primary
                          : theme.colors.outlineVariant,
                    },
                  ]}
                >
                  <Text
                    variant="labelSmall"
                    style={[
                      styles.stepNumber,
                      {
                        color:
                          isCurrent || isComplete
                            ? theme.colors.onPrimary
                            : theme.colors.onSurfaceVariant,
                      },
                    ]}
                  >
                    {isComplete && !isCurrent ? '\u2713' : step}
                  </Text>
                </View>
              </Pressable>
            </React.Fragment>
          );
        })}
      </View>

      <Text variant="titleSmall" style={[styles.stepLabel, { color: theme.colors.onSurface }]}>
        {STEP_LABELS[currentStep]}
      </Text>

      <Text variant="bodySmall" style={[styles.progress, { color: theme.colors.onSurfaceVariant }]}>
        Step {currentStep} of {TOTAL_STEPS}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    paddingVertical: spacing.md,
  },
  stepsRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  stepCircle: {
    width: 32,
    height: 32,
    borderRadius: 16,
    borderWidth: 2,
    justifyContent: 'center',
    alignItems: 'center',
  },
  stepNumber: {
    fontWeight: '600',
  },
  connector: {
    height: 2,
    width: 24,
  },
  stepLabel: {
    marginTop: spacing.sm,
    fontWeight: '600',
  },
  progress: {
    marginTop: spacing.xs,
  },
});
