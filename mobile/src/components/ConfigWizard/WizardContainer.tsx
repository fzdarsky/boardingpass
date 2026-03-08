/**
 * WizardContainer Component
 *
 * Renders the current step component, StepIndicator, and Next/Back buttons.
 * Validates step before forward navigation and tracks maxReachedStep.
 *
 * Steps 1–5 navigate with "Next" only (no per-step apply).
 * Step 6 (Review & Apply) handles both immediate and deferred apply flows.
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
import ReviewApplyPage from './ReviewApplyPage';
import { useConfigWizard } from '../../hooks/useConfigWizard';
import { WIZARD_STEPS } from '../../types/wizard';
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
  const [terminalState, setTerminalState] = useState<TerminalState>(null);

  const handleNext = useCallback(() => {
    // Review step (Step 6) is handled by ReviewApplyPage — no handleNext needed
    if (wizard.state.currentStep === WIZARD_STEPS.REVIEW) {
      return;
    }

    // Validate current step before navigating
    const validation = wizard.validateStep(wizard.state.currentStep);
    if (!validation.isValid) {
      setErrors(validation.errors);
      setShowError(true);
      return;
    }

    wizard.goNext();
  }, [wizard]);

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

  const isWifiSelected = wizard.state.networkInterface.interfaceType === 'wifi';
  const isReviewStep = wizard.state.currentStep === WIZARD_STEPS.REVIEW;

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
      case WIZARD_STEPS.REVIEW:
        return (
          <ReviewApplyPage
            apiClient={apiClient}
            onComplete={(terminal: TerminalState) => {
              setTerminalState(terminal);
              if (terminal) {
                const delay = terminal === 'rebooting' ? 8000 : 5000;
                setTimeout(() => onComplete?.(), delay);
              }
            }}
            onError={(msg: string) => {
              setErrors([msg]);
              setShowError(true);
            }}
          />
        );
      default:
        return null;
    }
  };

  // Determine if current step is being applied (hostname only)
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

        {/* Show ApplyFeedback for hostname failures or connectivity results (Step 3) */}
        {!isReviewStep &&
          currentApplyStatus &&
          (currentApplyStatus.status === 'failed' ||
            (currentApplyStatus.status === 'success' && currentApplyStatus.connectivityResult)) && (
            <ApplyFeedback
              applyStatus={currentApplyStatus}
              onRetry={
                apiClient
                  ? () => wizard.applyStepImmediate(wizard.state.currentStep, apiClient)
                  : undefined
              }
            />
          )}
      </ScrollView>

      {/* Hide nav buttons on Review step — ReviewApplyPage has its own buttons */}
      {!isReviewStep && (
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
            accessibilityLabel="Next step"
          >
            {isApplying ? 'Applying...' : 'Next'}
          </Button>
        </View>
      )}

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
