/**
 * Config Wizard Integration Test
 *
 * Tests the full wizard navigation flow: forward/backward navigation,
 * validation blocking, data preservation across steps, and apply modes.
 */

import { renderHook, act } from '@testing-library/react-native';
import React from 'react';
import { useConfigWizard, buildDeferredBundle } from '../../src/hooks/useConfigWizard';
import { WizardProvider } from '../../src/contexts/WizardContext';
import { WIZARD_STEPS, createInitialWizardState } from '../../src/types/wizard';
import type { WizardState } from '../../src/types/wizard';
import { sendConfigBundle, createConfigFile } from '../../src/services/api/configure';
import { completeProvisioning } from '../../src/services/api/complete';
import type { APIClient } from '../../src/services/api/client';

jest.mock('../../src/services/api/configure');
jest.mock('../../src/services/api/command');
jest.mock('../../src/services/api/complete');

const mockSendConfigBundle = sendConfigBundle as jest.MockedFunction<typeof sendConfigBundle>;
const mockCompleteProvisioning = completeProvisioning as jest.MockedFunction<
  typeof completeProvisioning
>;
const mockCreateConfigFile = createConfigFile as jest.MockedFunction<typeof createConfigFile>;

function createMockClient(): APIClient {
  return {
    getAuthToken: jest.fn().mockReturnValue('test-token'),
    setAuthToken: jest.fn(),
    clearAuthToken: jest.fn(),
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    getBaseURL: jest.fn().mockReturnValue('https://192.168.1.10:8443'),
    setBaseURL: jest.fn(),
  } as unknown as APIClient;
}

function createWrapper(initialState?: WizardState) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(WizardProvider, { initialState }, children);
  };
}

function createWrapperNoState() {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(WizardProvider, null, children);
  };
}

describe('Config Wizard Integration', () => {
  describe('full navigation flow', () => {
    it('navigates through all 5 steps with valid input', () => {
      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapperNoState(),
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
        wrapper: createWrapperNoState(),
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
        wrapper: createWrapperNoState(),
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
        wrapper: createWrapperNoState(),
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
        wrapper: createWrapperNoState(),
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
        wrapper: createWrapperNoState(),
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

  describe('deferred apply flow', () => {
    let mockClient: APIClient;

    beforeEach(() => {
      jest.clearAllMocks();
      mockClient = createMockClient();
      mockSendConfigBundle.mockResolvedValue({ status: 'success' });
      mockCompleteProvisioning.mockResolvedValue({
        status: 'rebooting',
        sentinel_file: '/etc/boardingpass/issued',
        message: 'Provisioning complete. Device will reboot.',
      });
      mockCreateConfigFile.mockImplementation((path, content, mode) => ({
        path,
        content: btoa(content),
        mode: mode || 0o644,
      }));
    });

    it('bundles all config into single /configure call then /complete with reboot', async () => {
      // Build a fully-configured deferred state
      const state = createInitialWizardState();
      state.serviceInterfaceName = 'eth0';
      state.applyMode = 'deferred';
      state.hostname.hostname = 'my-device';
      state.networkInterface = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };
      state.addressing = {
        ipv4: {
          method: 'static',
          address: '10.0.0.100',
          subnetMask: '255.255.255.0',
          gateway: '10.0.0.1',
          dnsAuto: false,
          dnsPrimary: '8.8.8.8',
          dnsSecondary: null,
        },
        ipv6: {
          method: 'disabled',
          address: null,
          gateway: null,
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
      };
      state.services = {
        ntp: { mode: 'manual', servers: ['ntp.example.com'] },
        proxy: { hostname: 'proxy.example.com', port: 3128, username: null, password: null },
      };
      state.enrollment.insights = {
        endpoint: 'https://cert-api.access.redhat.com',
        orgId: 'org123',
        activationKey: 'key456',
      };
      state.currentStep = WIZARD_STEPS.ENROLLMENT;
      state.maxReachedStep = WIZARD_STEPS.ENROLLMENT;

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      // Execute deferred apply
      await act(async () => {
        await result.current.applyDeferred(mockClient);
      });

      // Verify: single /configure call with all config files
      expect(mockSendConfigBundle).toHaveBeenCalledTimes(1);
      // The bundle is encoded via createConfigFile, so check it was called with the right paths
      expect(mockCreateConfigFile).toHaveBeenCalled();
      const createdPaths = mockCreateConfigFile.mock.calls.map(call => call[0]);
      expect(createdPaths).toContain('hostname');
      expect(createdPaths.some((p: string) => p.includes('nmconnection'))).toBe(true);
      expect(createdPaths).toContain('chrony.d/boardingpass-ntp.conf');
      expect(createdPaths).toContain('profile.d/boardingpass-proxy.sh');
      expect(createdPaths).toContain('boardingpass/staging/insights.json');
      expect(createdPaths).toContain('systemd/system/boardingpass-enroll.service');

      // Verify: /complete called with reboot: true
      expect(mockCompleteProvisioning).toHaveBeenCalledTimes(1);
      expect(mockCompleteProvisioning).toHaveBeenCalledWith(mockClient, true);
    });

    it('deferred bundle content matches quickstart API flow', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'edge-device-01';
      state.networkInterface = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };
      state.addressing = {
        ipv4: {
          method: 'dhcp',
          address: null,
          subnetMask: null,
          gateway: null,
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
        ipv6: {
          method: 'disabled',
          address: null,
          gateway: null,
          dnsAuto: true,
          dnsPrimary: null,
          dnsSecondary: null,
        },
      };

      const bundle = buildDeferredBundle(state);

      // Hostname file
      const hostnameFile = bundle.find(f => f.path === 'hostname');
      expect(hostnameFile?.content).toBe('edge-device-01\n');

      // NM connection file
      const nmFile = bundle.find(f => f.path.includes('nmconnection'));
      expect(nmFile?.content).toContain('[connection]');
      expect(nmFile?.content).toContain('interface-name=eth0');
      expect(nmFile?.content).toContain('method=auto');
    });

    it('does not include enrollment systemd service when no enrollment configured', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';
      state.networkInterface = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };

      const bundle = buildDeferredBundle(state);
      expect(bundle.find(f => f.path.includes('systemd'))).toBeUndefined();
      expect(bundle.find(f => f.path.includes('staging'))).toBeUndefined();
    });
  });
});
