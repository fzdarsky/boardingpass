/**
 * WizardContainer Component
 *
 * Renders the current step component, StepIndicator, and Next/Back buttons.
 * Validates step before forward navigation and tracks maxReachedStep.
 */

import React, { useState } from 'react';
import { View, ScrollView, StyleSheet, KeyboardAvoidingView, Platform } from 'react-native';
import { Button, Snackbar, useTheme } from 'react-native-paper';
import StepIndicator from './StepIndicator';
import HostnameStep from './HostnameStep';
import InterfaceStep from './InterfaceStep';
import AddressingStep from './AddressingStep';
import ServicesStep from './ServicesStep';
import EnrollmentStep from './EnrollmentStep';
import { useConfigWizard } from '../../hooks/useConfigWizard';
import { WIZARD_STEPS } from '../../types/wizard';
import type { components } from '../../types/api';
import { spacing } from '../../theme';

type NetworkInterface = components['schemas']['NetworkInterface'];

interface WizardContainerProps {
  interfaces: NetworkInterface[];
  serviceInterfaceName: string | null;
  onComplete?: () => void;
}

export default function WizardContainer({
  interfaces,
  serviceInterfaceName,
  onComplete,
}: WizardContainerProps) {
  const wizard = useConfigWizard();
  const theme = useTheme();
  const [errors, setErrors] = useState<string[]>([]);
  const [showError, setShowError] = useState(false);

  const handleNext = () => {
    if (wizard.isLastStep) {
      // Final step — validate and complete
      const validation = wizard.validateStep(wizard.state.currentStep);
      if (validation.isValid) {
        onComplete?.();
      } else {
        setErrors(validation.errors);
        setShowError(true);
      }
    } else {
      const validation = wizard.goNext();
      if (!validation.isValid) {
        setErrors(validation.errors);
        setShowError(true);
      }
    }
  };

  const handleStepPress = (step: number) => {
    if (wizard.canNavigateTo(step)) {
      wizard.setStep(step);
    }
  };

  const renderStep = () => {
    switch (wizard.state.currentStep) {
      case WIZARD_STEPS.HOSTNAME:
        return <HostnameStep />;
      case WIZARD_STEPS.INTERFACE:
        return (
          <InterfaceStep interfaces={interfaces} serviceInterfaceName={serviceInterfaceName} />
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
      </ScrollView>

      <View style={[styles.navigation, { borderTopColor: theme.colors.outlineVariant }]}>
        <Button
          mode="outlined"
          onPress={wizard.goBack}
          disabled={wizard.isFirstStep}
          style={styles.navButton}
          accessibilityLabel="Previous step"
        >
          Back
        </Button>

        <Button
          mode="contained"
          onPress={handleNext}
          style={styles.navButton}
          accessibilityLabel={wizard.isLastStep ? 'Finish wizard' : 'Next step'}
        >
          {wizard.isLastStep ? 'Finish' : 'Next'}
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
});
