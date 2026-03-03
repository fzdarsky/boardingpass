/**
 * Config Wizard Integration Test
 *
 * Tests the full wizard navigation flow: forward/backward navigation,
 * validation blocking, and data preservation across steps.
 */

import { renderHook, act } from '@testing-library/react-native';
import React from 'react';
import { useConfigWizard } from '../../src/hooks/useConfigWizard';
import { WizardProvider } from '../../src/contexts/WizardContext';
import { WIZARD_STEPS } from '../../src/types/wizard';

function createWrapper() {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(WizardProvider, null, children);
  };
}

describe('Config Wizard Integration', () => {
  describe('full navigation flow', () => {
    it('navigates through all 5 steps with valid input', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      // Step 1: Set hostname and advance
      act(() => {
        result.current.updateHostname({ hostname: 'my-device' });
      });
      act(() => {
        const v = result.current.goNext();
        expect(v.isValid).toBe(true);
      });
      expect(result.current.state.currentStep).toBe(WIZARD_STEPS.INTERFACE);

      // Step 2: Select interface and advance
      act(() => {
        result.current.updateInterface({
          interfaceName: 'eth0',
          interfaceType: 'ethernet',
          vlanId: null,
          wifi: null,
        });
      });
      act(() => {
        const v = result.current.goNext();
        expect(v.isValid).toBe(true);
      });
      expect(result.current.state.currentStep).toBe(WIZARD_STEPS.ADDRESSING);

      // Step 3: DHCP defaults are valid, advance
      act(() => {
        const v = result.current.goNext();
        expect(v.isValid).toBe(true);
      });
      expect(result.current.state.currentStep).toBe(WIZARD_STEPS.SERVICES);

      // Step 4: Auto NTP defaults are valid, advance
      act(() => {
        const v = result.current.goNext();
        expect(v.isValid).toBe(true);
      });
      expect(result.current.state.currentStep).toBe(WIZARD_STEPS.ENROLLMENT);

      // Step 5: No enrollment is valid (optional)
      const finalValidation = result.current.validateStep(WIZARD_STEPS.ENROLLMENT);
      expect(finalValidation.isValid).toBe(true);
    });

    it('blocks forward navigation on invalid input', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      // Try to advance without hostname
      act(() => {
        const v = result.current.goNext();
        expect(v.isValid).toBe(false);
        expect(v.errors.length).toBeGreaterThan(0);
      });

      // Should stay on step 1
      expect(result.current.state.currentStep).toBe(WIZARD_STEPS.HOSTNAME);
    });

    it('preserves data when navigating backward', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      // Fill hostname and advance
      act(() => {
        result.current.updateHostname({ hostname: 'preserved-host' });
      });
      act(() => {
        result.current.goNext();
      });

      // Fill interface and advance
      act(() => {
        result.current.updateInterface({
          interfaceName: 'eth1',
          interfaceType: 'ethernet',
          vlanId: null,
          wifi: null,
        });
      });
      act(() => {
        result.current.goNext();
      });

      expect(result.current.state.currentStep).toBe(WIZARD_STEPS.ADDRESSING);

      // Go back to step 1
      act(() => {
        result.current.goBack();
      });
      act(() => {
        result.current.goBack();
      });

      expect(result.current.state.currentStep).toBe(WIZARD_STEPS.HOSTNAME);

      // Verify data is preserved
      expect(result.current.state.hostname.hostname).toBe('preserved-host');
      expect(result.current.state.networkInterface.interfaceName).toBe('eth1');
    });

    it('tracks maxReachedStep correctly', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      expect(result.current.state.maxReachedStep).toBe(1);

      // Advance to step 3
      act(() => result.current.updateHostname({ hostname: 'host' }));
      act(() => result.current.goNext());
      act(() =>
        result.current.updateInterface({
          interfaceName: 'eth0',
          interfaceType: 'ethernet',
          vlanId: null,
          wifi: null,
        })
      );
      act(() => result.current.goNext());

      expect(result.current.state.maxReachedStep).toBe(3);

      // Go back to step 1
      act(() => result.current.goBack());
      act(() => result.current.goBack());

      // maxReachedStep should still be 3
      expect(result.current.state.maxReachedStep).toBe(3);
      expect(result.current.state.currentStep).toBe(1);
    });

    it('allows navigating to previously reached steps', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      // Advance to step 3
      act(() => result.current.updateHostname({ hostname: 'host' }));
      act(() => result.current.goNext());
      act(() =>
        result.current.updateInterface({
          interfaceName: 'eth0',
          interfaceType: 'ethernet',
          vlanId: null,
          wifi: null,
        })
      );
      act(() => result.current.goNext());

      // Go back to step 1
      act(() => result.current.goBack());
      act(() => result.current.goBack());

      // Should be able to jump directly to step 3
      expect(result.current.canNavigateTo(3)).toBe(true);
    });
  });

  describe('wizard reset', () => {
    it('resets all state to initial values', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(),
      });

      // Modify state
      act(() => result.current.updateHostname({ hostname: 'modified' }));
      act(() => result.current.goNext());

      // Reset
      act(() => result.current.reset());

      expect(result.current.state.currentStep).toBe(1);
      expect(result.current.state.hostname.hostname).toBe('');
      expect(result.current.state.maxReachedStep).toBe(1);
    });
  });
});
