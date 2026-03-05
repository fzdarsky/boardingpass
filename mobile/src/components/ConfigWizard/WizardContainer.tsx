/**
 * WizardContainer Component
 *
 * Renders the current step component, StepIndicator, and Next/Back buttons.
 * Validates step before forward navigation and tracks maxReachedStep.
 *
 * Handles two apply modes:
 * - Immediate: applies config per-step via API, shows ApplyFeedback
 * - Deferred: shows ReviewPage after final step, sends atomic bundle + reboot
 */

import React, { useState, useCallback } from 'react';
import { View, ScrollView, StyleSheet, KeyboardAvoidingView, Platform } from 'react-native';
import { Button, Snackbar, Text, ActivityIndicator, useTheme } from 'react-native-paper';
import MaterialCommunityIcons from '@expo/vector-icons/MaterialCommunityIcons';
import StepIndicator from './StepIndicator';
import HostnameStep from './HostnameStep';
import InterfaceStep from './InterfaceStep';
import WiFiStep from './WiFiStep';
import AddressingStep from './AddressingStep';
import ServicesStep from './ServicesStep';
import EnrollmentStep from './EnrollmentStep';
import ApplyFeedback from './ApplyFeedback';
import ReviewPage from './ReviewPage';
import { useConfigWizard } from '../../hooks/useConfigWizard';
import { WIZARD_STEPS } from '../../types/wizard';
import { completeProvisioning } from '../../services/api/complete';
import type { components } from '../../types/api';
import type { APIClient } from '../../services/api/client';
import { spacing } from '../../theme';

type NetworkInterface = components['schemas']['NetworkInterface'];

interface WizardContainerProps {
  interfaces: NetworkInterface[];
  apiClient?: APIClient;
  onComplete?: () => void;
}

type TerminalState = 'rebooting' | 'shutting_down' | null;

export default function WizardContainer({
  interfaces,
  apiClient,
  onComplete,
}: WizardContainerProps) {
  const wizard = useConfigWizard();
  const theme = useTheme();
  const [errors, setErrors] = useState<string[]>([]);
  const [showError, setShowError] = useState(false);
  const [showReview, setShowReview] = useState(false);
  const [deferredApplying, setDeferredApplying] = useState(false);
  const [terminalState, setTerminalState] = useState<TerminalState>(null);

  const isImmediate = wizard.state.applyMode === 'immediate';

  // Hostname is always applied immediately (safe — doesn't affect connectivity).
  // Other steps are applied immediately only when the wizard is in immediate mode.
  // Interface step (Step 2) never has config to apply.
  const willApplyOnNext =
    !wizard.isLastStep &&
    (wizard.state.currentStep === WIZARD_STEPS.HOSTNAME ||
      (isImmediate && wizard.state.currentStep !== WIZARD_STEPS.INTERFACE));

  const handleNext = useCallback(async () => {
    if (wizard.isLastStep) {
      const validation = wizard.validateStep(wizard.state.currentStep);
      if (!validation.isValid) {
        setErrors(validation.errors);
        setShowError(true);
        return;
      }

      if (isImmediate && apiClient) {
        // Immediate mode: apply the final step, then call /complete
        const applyError = await wizard.applyStepImmediate(wizard.state.currentStep, apiClient);
        if (applyError) {
          setErrors([applyError]);
          setShowError(true);
          return;
        }

        try {
          await completeProvisioning(apiClient, false);
          setTerminalState('shutting_down');
          setTimeout(() => onComplete?.(), 5000);
        } catch {
          // If /complete fails, still allow retry via the Finish flow
          setErrors(['Failed to finalize provisioning. The device may need manual attention.']);
          setShowError(true);
        }
      } else if (wizard.state.applyMode === 'deferred') {
        // Deferred mode: show review page
        setShowReview(true);
      } else {
        // No apply mode (e.g. no apiClient) — just complete
        onComplete?.();
      }
    } else {
      // Validate current step before doing anything
      const validation = wizard.validateStep(wizard.state.currentStep);
      if (!validation.isValid) {
        setErrors(validation.errors);
        setShowError(true);
        return;
      }

      // Apply config first (if needed), then navigate on success
      const shouldApply = wizard.state.currentStep === WIZARD_STEPS.HOSTNAME || isImmediate;
      if (shouldApply && apiClient) {
        const error = await wizard.applyStepImmediate(wizard.state.currentStep, apiClient);
        if (error) {
          setErrors([error]);
          setShowError(true);
          return;
        }
      }

      wizard.goNext();
    }
  }, [wizard, isImmediate, apiClient, onComplete]);

  const handleDeferredConfirm = useCallback(async () => {
    if (!apiClient) return;

    setDeferredApplying(true);
    try {
      await wizard.applyDeferred(apiClient);
      setTerminalState('rebooting');
      setTimeout(() => onComplete?.(), 8000);
    } catch (err) {
      setDeferredApplying(false);
      setErrors([err instanceof Error ? err.message : 'Failed to apply configuration']);
      setShowError(true);
    }
  }, [apiClient, wizard, onComplete]);

  const handleStepPress = (step: number) => {
    if (wizard.canNavigateTo(step)) {
      wizard.setStep(step);
    }
  };

  // Terminal states: device is rebooting or shutting down
  if (terminalState) {
    const isRebooting = terminalState === 'rebooting';
    return (
      <View style={styles.terminalContainer}>
        <ActivityIndicator size="large" animating={true} />
        <MaterialCommunityIcons
          name={isRebooting ? 'restart' : 'power'}
          size={48}
          color={theme.colors.primary}
          style={styles.terminalIcon}
        />
        <Text
          variant="headlineSmall"
          style={[styles.terminalTitle, { color: theme.colors.onSurface }]}
        >
          {isRebooting ? 'Device is Rebooting' : 'Provisioning Complete'}
        </Text>
        <Text
          variant="bodyMedium"
          style={[styles.terminalMessage, { color: theme.colors.onSurfaceVariant }]}
        >
          {isRebooting
            ? 'Configuration has been applied. The device will restart with the new settings.'
            : 'The device is shutting down the provisioning service.'}
        </Text>
      </View>
    );
  }

  // Deferred mode review page
  if (showReview) {
    return (
      <KeyboardAvoidingView
        style={styles.container}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        keyboardVerticalOffset={100}
      >
        <ScrollView
          style={styles.content}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
        >
          <ReviewPage
            onConfirm={handleDeferredConfirm}
            onBack={() => setShowReview(false)}
            applying={deferredApplying}
          />
        </ScrollView>

        <Snackbar
          visible={showError}
          onDismiss={() => setShowError(false)}
          duration={4000}
          action={{ label: 'Dismiss', onPress: () => setShowError(false) }}
        >
          {errors[0] || 'An error occurred.'}
        </Snackbar>
      </KeyboardAvoidingView>
    );
  }

  const isWifiSelected = wizard.state.networkInterface.interfaceType === 'wifi';

  const renderStep = () => {
    switch (wizard.state.currentStep) {
      case WIZARD_STEPS.HOSTNAME:
        return <HostnameStep />;
      case WIZARD_STEPS.INTERFACE:
        return (
          <InterfaceStep interfaces={interfaces}>
            {isWifiSelected && <WiFiStep apiClient={apiClient} />}
          </InterfaceStep>
        );
      case WIZARD_STEPS.ADDRESSING:
        return <AddressingStep />;
      case WIZARD_STEPS.SERVICES:
        return <ServicesStep />;
      case WIZARD_STEPS.ENROLLMENT:
        return <EnrollmentStep />;
      default:
        return null;
    }
  };

  // Determine if current step is being applied (immediate mode)
  const currentApplyStatus = wizard.state.stepApplyStatus[wizard.state.currentStep];
  const isApplying = currentApplyStatus?.status === 'applying';

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={100}
    >
      <StepIndicator
        currentStep={wizard.state.currentStep}
        maxReachedStep={wizard.state.maxReachedStep}
        isStepComplete={wizard.isStepComplete}
        onStepPress={handleStepPress}
      />

      <ScrollView
        style={styles.content}
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
      >
        {renderStep()}

        {/* Show connectivity test results after addressing step */}
        {currentApplyStatus?.status === 'success' && currentApplyStatus.connectivityResult && (
          <ApplyFeedback applyStatus={currentApplyStatus} />
        )}
      </ScrollView>

      <View style={[styles.navigation, { borderTopColor: theme.colors.outlineVariant }]}>
        <Button
          mode="outlined"
          onPress={wizard.goBack}
          disabled={wizard.isFirstStep || isApplying}
          style={styles.navButton}
          accessibilityLabel="Previous step"
        >
          Back
        </Button>

        <Button
          mode="contained"
          onPress={handleNext}
          disabled={isApplying}
          loading={isApplying}
          style={styles.navButton}
          accessibilityLabel={
            wizard.isLastStep
              ? 'Finish wizard'
              : willApplyOnNext
                ? 'Apply and go to next step'
                : 'Next step'
          }
        >
          {isApplying
            ? 'Applying...'
            : wizard.isLastStep
              ? wizard.state.applyMode === 'deferred'
                ? 'Review'
                : 'Finish'
              : willApplyOnNext
                ? 'Apply & Next'
                : 'Next'}
        </Button>
      </View>

      <Snackbar
        visible={showError}
        onDismiss={() => setShowError(false)}
        duration={4000}
        action={{
          label: 'Dismiss',
          onPress: () => setShowError(false),
        }}
      >
        {errors[0] || 'Please fix validation errors before continuing.'}
      </Snackbar>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  content: {
    flex: 1,
  },
  scrollContent: {
    paddingBottom: spacing.xl,
  },
  navigation: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    padding: spacing.md,
    borderTopWidth: 1,
  },
  navButton: {
    minWidth: 100,
  },
  terminalContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.xl,
  },
  terminalIcon: {
    marginTop: spacing.lg,
  },
  terminalTitle: {
    marginTop: spacing.md,
    textAlign: 'center',
  },
  terminalMessage: {
    marginTop: spacing.sm,
    textAlign: 'center',
  },
});
