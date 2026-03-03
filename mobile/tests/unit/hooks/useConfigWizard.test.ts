/**
 * useConfigWizard Hook Tests
 *
 * Tests for step validation, navigation guards, and step completion checks.
 */

import { renderHook, act } from '@testing-library/react-native';
import React from 'react';
import { useConfigWizard } from '../../../src/hooks/useConfigWizard';
import { WizardProvider } from '../../../src/contexts/WizardContext';
import { createInitialWizardState, WIZARD_STEPS } from '../../../src/types/wizard';
import type { WizardState } from '../../../src/types/wizard';

function createWrapper(initialState?: WizardState) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(WizardProvider, { initialState }, children);
  };
}

describe('useConfigWizard', () => {
  describe('step validation', () => {
    it('rejects empty hostname', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.HOSTNAME);
      expect(validation.isValid).toBe(false);
      expect(validation.errors).toContain('Hostname is required');
    });

    it('accepts valid hostname', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.HOSTNAME);
      expect(validation.isValid).toBe(true);
      expect(validation.errors).toHaveLength(0);
    });

    it('rejects empty interface selection', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.INTERFACE);
      expect(validation.isValid).toBe(false);
      expect(validation.errors).toContain('Please select a network interface');
    });

    it('accepts valid interface selection', () => {
      const state = createInitialWizardState();
      state.networkInterface.interfaceName = 'eth0';
      state.networkInterface.interfaceType = 'ethernet';

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.INTERFACE);
      expect(validation.isValid).toBe(true);
    });

    it('rejects invalid VLAN ID', () => {
      const state = createInitialWizardState();
      state.networkInterface.interfaceName = 'eth0';
      state.networkInterface.interfaceType = 'ethernet';
      state.networkInterface.vlanId = 0;

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.INTERFACE);
      expect(validation.isValid).toBe(false);
      expect(validation.errors[0]).toContain('VLAN ID');
    });

    it('validates static IPv4 fields', () => {
      const state = createInitialWizardState();
      state.addressing.ipv4.method = 'static';

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.ADDRESSING);
      expect(validation.isValid).toBe(false);
      expect(validation.errors.length).toBeGreaterThanOrEqual(3);
    });

    it('accepts valid static IPv4 configuration', () => {
      const state = createInitialWizardState();
      state.addressing.ipv4 = {
        method: 'static',
        address: '192.168.1.100',
        subnetMask: '255.255.255.0',
        gateway: '192.168.1.1',
        dnsAuto: true,
        dnsPrimary: null,
        dnsSecondary: null,
      };

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.ADDRESSING);
      expect(validation.isValid).toBe(true);
    });

    it('validates manual DNS when auto is disabled', () => {
      const state = createInitialWizardState();
      state.addressing.ipv4.dnsAuto = false;

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.ADDRESSING);
      expect(validation.isValid).toBe(false);
      expect(validation.errors.some(e => e.includes('Primary DNS'))).toBe(true);
    });

    it('validates manual NTP servers', () => {
      const state = createInitialWizardState();
      state.services.ntp.mode = 'manual';

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.SERVICES);
      expect(validation.isValid).toBe(false);
      expect(validation.errors.some(e => e.includes('NTP server'))).toBe(true);
    });

    it('validates proxy port', () => {
      const state = createInitialWizardState();
      state.services.proxy = {
        hostname: 'proxy.example.com',
        port: 0,
        username: null,
        password: null,
      };

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.SERVICES);
      expect(validation.isValid).toBe(false);
      expect(validation.errors.some(e => e.includes('port'))).toBe(true);
    });

    it('accepts enrollment step with no enrollments', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.ENROLLMENT);
      expect(validation.isValid).toBe(true);
    });

    it('validates insights enrollment fields', () => {
      const state = createInitialWizardState();
      state.enrollment.insights = {
        endpoint: 'not-a-url',
        orgId: '',
        activationKey: '',
      };

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.ENROLLMENT);
      expect(validation.isValid).toBe(false);
      expect(validation.errors.length).toBeGreaterThanOrEqual(2);
    });

    it('validates flight control enrollment fields', () => {
      const state = createInitialWizardState();
      state.enrollment.flightControl = {
        endpoint: 'https://fc.example.com',
        username: '',
        password: '',
      };

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      const validation = result.current.validateStep(WIZARD_STEPS.ENROLLMENT);
      expect(validation.isValid).toBe(false);
      expect(validation.errors.some(e => e.includes('Username'))).toBe(true);
      expect(validation.errors.some(e => e.includes('Password'))).toBe(true);
    });
  });

  describe('navigation guards', () => {
    it('can navigate back to step 1 from step 2', () => {
      const state = createInitialWizardState();
      state.currentStep = 2;
      state.maxReachedStep = 2;

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      expect(result.current.canNavigateTo(1)).toBe(true);
    });

    it('cannot skip forward past maxReachedStep', () => {
      const state = createInitialWizardState();
      state.currentStep = 1;
      state.maxReachedStep = 1;

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      expect(result.current.canNavigateTo(3)).toBe(false);
    });

    it('goNext validates before advancing', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      let validation: { isValid: boolean; errors: string[] } = { isValid: false, errors: [] };
      act(() => {
        validation = result.current.goNext();
      });

      expect(validation.isValid).toBe(false);
      // Should still be on step 1
      expect(result.current.state.currentStep).toBe(1);
    });

    it('goNext advances when step is valid', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'valid-host';

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      let validation: { isValid: boolean; errors: string[] } = { isValid: false, errors: [] };
      act(() => {
        validation = result.current.goNext();
      });

      expect(validation.isValid).toBe(true);
      expect(result.current.state.currentStep).toBe(2);
    });

    it('goBack goes to previous step without validation', () => {
      const state = createInitialWizardState();
      state.currentStep = 3;
      state.maxReachedStep = 3;

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      act(() => {
        result.current.goBack();
      });

      expect(result.current.state.currentStep).toBe(2);
    });

    it('goBack does nothing on first step', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.goBack();
      });

      expect(result.current.state.currentStep).toBe(1);
    });
  });

  describe('step completion checks', () => {
    it('reports incomplete for unvisited steps', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      expect(result.current.isStepComplete(3)).toBe(false);
    });

    it('reports complete for valid visited step', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';
      state.maxReachedStep = 2;

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      expect(result.current.isStepComplete(1)).toBe(true);
    });

    it('isFirstStep is true on step 1', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      expect(result.current.isFirstStep).toBe(true);
    });

    it('isLastStep is true on step 5', () => {
      const state = createInitialWizardState();
      state.currentStep = 5;
      state.maxReachedStep = 5;

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      expect(result.current.isLastStep).toBe(true);
    });
  });
});
