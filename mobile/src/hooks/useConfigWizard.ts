/**
 * useConfigWizard Hook
 *
 * Encapsulates wizard step validation logic, per-step data access,
 * navigation guards, and step completion checks.
 *
 * This hook reads from WizardContext and provides validation + navigation helpers.
 */

import { useCallback, useMemo } from 'react';
import { useWizard } from '../contexts/WizardContext';
import {
  validateHostname,
  validateIPv4,
  validateIPv6,
  validateSubnetMask,
  validatePort,
  validateHttpsUrl,
  validateNtpServer,
} from '../utils/network-validation';
import { WIZARD_STEPS, TOTAL_STEPS } from '../types/wizard';

export interface StepValidation {
  isValid: boolean;
  errors: string[];
}

export function useConfigWizard() {
  const wizard = useWizard();
  const { state } = wizard;

  /**
   * Validate the current step's data.
   */
  const validateStep = useCallback(
    (step: number): StepValidation => {
      const errors: string[] = [];

      switch (step) {
        case WIZARD_STEPS.HOSTNAME: {
          const err = validateHostname(state.hostname.hostname);
          if (err) errors.push(err);
          break;
        }

        case WIZARD_STEPS.INTERFACE: {
          if (!state.networkInterface.interfaceName) {
            errors.push('Please select a network interface');
          }
          if (
            state.networkInterface.vlanId !== null &&
            (state.networkInterface.vlanId < 1 || state.networkInterface.vlanId > 4094)
          ) {
            errors.push('VLAN ID must be between 1 and 4094');
          }
          break;
        }

        case WIZARD_STEPS.ADDRESSING: {
          // IPv4 static validation
          if (state.addressing.ipv4.method === 'static') {
            if (state.addressing.ipv4.address) {
              const err = validateIPv4(state.addressing.ipv4.address);
              if (err) errors.push(`IPv4 address: ${err}`);
            } else {
              errors.push('IPv4 address is required for static configuration');
            }

            if (state.addressing.ipv4.subnetMask) {
              const err = validateSubnetMask(state.addressing.ipv4.subnetMask);
              if (err) errors.push(`Subnet mask: ${err}`);
            } else {
              errors.push('Subnet mask is required for static configuration');
            }

            if (state.addressing.ipv4.gateway) {
              const err = validateIPv4(state.addressing.ipv4.gateway);
              if (err) errors.push(`Gateway: ${err}`);
            } else {
              errors.push('Gateway is required for static configuration');
            }
          }

          // IPv4 manual DNS validation
          if (!state.addressing.ipv4.dnsAuto) {
            if (state.addressing.ipv4.dnsPrimary) {
              const err = validateIPv4(state.addressing.ipv4.dnsPrimary);
              if (err) errors.push(`Primary DNS: ${err}`);
            } else {
              errors.push('Primary DNS is required when auto DNS is disabled');
            }
            if (state.addressing.ipv4.dnsSecondary) {
              const err = validateIPv4(state.addressing.ipv4.dnsSecondary);
              if (err) errors.push(`Secondary DNS: ${err}`);
            }
          }

          // IPv6 static validation
          if (state.addressing.ipv6.method === 'static') {
            if (state.addressing.ipv6.address) {
              const err = validateIPv6(state.addressing.ipv6.address, true);
              if (err) errors.push(`IPv6 address: ${err}`);
            } else {
              errors.push('IPv6 address is required for static configuration');
            }

            if (state.addressing.ipv6.gateway) {
              const err = validateIPv6(state.addressing.ipv6.gateway);
              if (err) errors.push(`IPv6 gateway: ${err}`);
            }
          }

          // IPv6 manual DNS validation
          if (state.addressing.ipv6.method !== 'disabled' && !state.addressing.ipv6.dnsAuto) {
            if (state.addressing.ipv6.dnsPrimary) {
              const err = validateIPv6(state.addressing.ipv6.dnsPrimary);
              if (err) errors.push(`IPv6 primary DNS: ${err}`);
            }
          }
          break;
        }

        case WIZARD_STEPS.SERVICES: {
          // NTP manual servers
          if (state.services.ntp.mode === 'manual') {
            if (state.services.ntp.servers.length === 0) {
              errors.push('At least one NTP server is required in manual mode');
            }
            for (const server of state.services.ntp.servers) {
              const err = validateNtpServer(server);
              if (err) errors.push(`NTP server "${server}": ${err}`);
            }
          }

          // Proxy validation
          if (state.services.proxy) {
            if (!state.services.proxy.hostname) {
              errors.push('Proxy hostname is required');
            }
            const portErr = validatePort(state.services.proxy.port);
            if (portErr) errors.push(`Proxy port: ${portErr}`);
          }
          break;
        }

        case WIZARD_STEPS.ENROLLMENT: {
          // Insights validation
          if (state.enrollment.insights) {
            const urlErr = validateHttpsUrl(state.enrollment.insights.endpoint);
            if (urlErr) errors.push(`Insights endpoint: ${urlErr}`);
            if (!state.enrollment.insights.orgId) {
              errors.push('Organisation ID is required for Insights enrollment');
            }
            if (!state.enrollment.insights.activationKey) {
              errors.push('Activation Key is required for Insights enrollment');
            }
          }

          // Flight Control validation
          if (state.enrollment.flightControl) {
            const urlErr = validateHttpsUrl(state.enrollment.flightControl.endpoint);
            if (urlErr) errors.push(`Flight Control endpoint: ${urlErr}`);
            if (!state.enrollment.flightControl.username) {
              errors.push('Username is required for Flight Control enrollment');
            }
            if (!state.enrollment.flightControl.password) {
              errors.push('Password is required for Flight Control enrollment');
            }
          }
          break;
        }
      }

      return { isValid: errors.length === 0, errors };
    },
    [state]
  );

  /**
   * Check if the user can navigate to a given step.
   */
  const canNavigateTo = useCallback(
    (step: number): boolean => {
      // Can always go back to previously reached steps
      if (step <= state.maxReachedStep) return true;
      // Can only go forward one step at a time, and current step must be valid
      if (step === state.currentStep + 1) {
        return validateStep(state.currentStep).isValid;
      }
      return false;
    },
    [state.currentStep, state.maxReachedStep, validateStep]
  );

  /**
   * Navigate to the next step (validates current step first).
   * Returns validation result — caller should show errors if invalid.
   */
  const goNext = useCallback((): StepValidation => {
    const validation = validateStep(state.currentStep);
    if (validation.isValid && state.currentStep < TOTAL_STEPS) {
      wizard.setStep(state.currentStep + 1);
    }
    return validation;
  }, [state.currentStep, validateStep, wizard]);

  /**
   * Navigate to the previous step (no validation needed).
   */
  const goBack = useCallback(() => {
    if (state.currentStep > WIZARD_STEPS.HOSTNAME) {
      wizard.setStep(state.currentStep - 1);
    }
  }, [state.currentStep, wizard]);

  /**
   * Check if a step is complete (valid data entered).
   */
  const isStepComplete = useCallback(
    (step: number): boolean => {
      if (step > state.maxReachedStep) return false;
      return validateStep(step).isValid;
    },
    [state.maxReachedStep, validateStep]
  );

  /**
   * Whether the wizard is on the first step.
   */
  const isFirstStep = state.currentStep === WIZARD_STEPS.HOSTNAME;

  /**
   * Whether the wizard is on the last step.
   */
  const isLastStep = state.currentStep === TOTAL_STEPS;

  /**
   * Current step validation state.
   */
  const currentValidation = useMemo(
    () => validateStep(state.currentStep),
    [state.currentStep, validateStep]
  );

  return {
    ...wizard,
    validateStep,
    canNavigateTo,
    goNext,
    goBack,
    isStepComplete,
    isFirstStep,
    isLastStep,
    currentValidation,
  };
}
