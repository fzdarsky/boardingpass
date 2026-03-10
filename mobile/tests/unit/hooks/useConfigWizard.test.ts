/**
 * useConfigWizard Hook Tests
 *
 * Tests for step validation, navigation guards, step completion checks,
 * apply mode detection, and per-step immediate apply logic.
 */

import { renderHook, act } from '@testing-library/react-native';
import React from 'react';
import {
  useConfigWizard,
  buildStepConfigFiles,
  buildStepCommands,
  buildDeferredBundle,
  parseConnectivityResult,
  formatConnectivityDetail,
  isConnectivityFullPass,
} from '../../../src/hooks/useConfigWizard';
import { WizardProvider } from '../../../src/contexts/WizardContext';
import { createInitialWizardState, WIZARD_STEPS } from '../../../src/types/wizard';
import type { WizardState } from '../../../src/types/wizard';
import { sendConfigBundle, createConfigFile } from '../../../src/services/api/configure';
import { executeCommand } from '../../../src/services/api/command';
import { completeProvisioning } from '../../../src/services/api/complete';
import type { APIClient } from '../../../src/services/api/client';

jest.mock('../../../src/services/api/configure');
jest.mock('../../../src/services/api/command');
jest.mock('../../../src/services/api/complete');

const mockSendConfigBundle = sendConfigBundle as jest.MockedFunction<typeof sendConfigBundle>;
const mockExecuteCommand = executeCommand as jest.MockedFunction<typeof executeCommand>;
const _mockCompleteProvisioning = completeProvisioning as jest.MockedFunction<
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
        authMethod: 'password' as const,
        token: null,
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

    it('isLastStep is true on step 6 (Review)', () => {
      const state = createInitialWizardState();
      state.currentStep = 6;
      state.maxReachedStep = 6;

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      expect(result.current.isLastStep).toBe(true);
    });
  });

  describe('apply mode detection', () => {
    it('sets immediate mode when selected interface differs from service interface', () => {
      const state = createInitialWizardState();
      state.serviceInterfaceName = 'eth0';
      state.hostname.hostname = 'my-device';
      state.currentStep = WIZARD_STEPS.INTERFACE;
      state.maxReachedStep = WIZARD_STEPS.INTERFACE;
      state.networkInterface = {
        interfaceName: 'eth1',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      act(() => {
        result.current.goNext();
      });

      expect(result.current.state.applyMode).toBe('immediate');
    });

    it('sets deferred mode when selected interface matches service interface', () => {
      const state = createInitialWizardState();
      state.serviceInterfaceName = 'eth0';
      state.hostname.hostname = 'my-device';
      state.currentStep = WIZARD_STEPS.INTERFACE;
      state.maxReachedStep = WIZARD_STEPS.INTERFACE;
      state.networkInterface = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      act(() => {
        result.current.goNext();
      });

      expect(result.current.state.applyMode).toBe('deferred');
    });

    it('re-evaluates mode when user goes back and changes interface', () => {
      const state = createInitialWizardState();
      state.serviceInterfaceName = 'eth0';
      state.hostname.hostname = 'my-device';
      state.currentStep = WIZARD_STEPS.ADDRESSING;
      state.maxReachedStep = WIZARD_STEPS.ADDRESSING;
      state.applyMode = 'deferred';
      state.networkInterface = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      // Go back to interface step and change selection
      act(() => {
        result.current.goBack();
      });
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

      expect(result.current.state.applyMode).toBe('immediate');
    });

    it('leaves mode null when no service interface is known', () => {
      const state = createInitialWizardState();
      state.serviceInterfaceName = null;
      state.hostname.hostname = 'my-device';
      state.currentStep = WIZARD_STEPS.INTERFACE;
      state.maxReachedStep = WIZARD_STEPS.INTERFACE;
      state.networkInterface = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      act(() => {
        result.current.goNext();
      });

      // When we can't determine service interface, default to deferred (safe)
      expect(result.current.state.applyMode).toBe('deferred');
    });
  });

  describe('per-step config file building', () => {
    it('builds hostname file for step 1', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';

      const files = buildStepConfigFiles(WIZARD_STEPS.HOSTNAME, state);
      expect(files).toHaveLength(1);
      expect(files[0].path).toBe('hostname');
      expect(files[0].content).toBe('my-device\n');
    });

    it('returns no files for step 2 (interface selection only)', () => {
      const state = createInitialWizardState();
      state.networkInterface.interfaceName = 'eth0';
      state.networkInterface.interfaceType = 'ethernet';

      const files = buildStepConfigFiles(WIZARD_STEPS.INTERFACE, state);
      expect(files).toHaveLength(0);
    });

    it('builds NM connection file for step 3', () => {
      const state = createInitialWizardState();
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

      const files = buildStepConfigFiles(WIZARD_STEPS.ADDRESSING, state);
      expect(files).toHaveLength(1);
      expect(files[0].path).toBe(
        'NetworkManager/system-connections/boardingpass-enrollment.nmconnection'
      );
      expect(files[0].content).toContain('[connection]');
      expect(files[0].content).toContain('type=ethernet');
    });

    it('builds NTP config file for step 4 with manual servers', () => {
      const state = createInitialWizardState();
      state.services.ntp = { mode: 'manual', servers: ['ntp1.example.com', 'ntp2.example.com'] };

      const files = buildStepConfigFiles(WIZARD_STEPS.SERVICES, state);
      const ntpFile = files.find(f => f.path === 'chrony.d/boardingpass-ntp.conf');
      expect(ntpFile).toBeDefined();
      expect(ntpFile!.content).toContain('server ntp1.example.com iburst');
      expect(ntpFile!.content).toContain('server ntp2.example.com iburst');
    });

    it('returns no NTP file for step 4 with automatic NTP', () => {
      const state = createInitialWizardState();
      state.services.ntp = { mode: 'automatic', servers: [] };
      state.services.proxy = null;

      const files = buildStepConfigFiles(WIZARD_STEPS.SERVICES, state);
      expect(files).toHaveLength(0);
    });

    it('builds proxy config file for step 4 with proxy settings', () => {
      const state = createInitialWizardState();
      state.services.proxy = {
        hostname: 'proxy.example.com',
        port: 8080,
        username: null,
        password: null,
      };

      const files = buildStepConfigFiles(WIZARD_STEPS.SERVICES, state);
      const proxyFile = files.find(f => f.path === 'profile.d/boardingpass-proxy.sh');
      expect(proxyFile).toBeDefined();
      expect(proxyFile!.content).toContain('http_proxy');
      expect(proxyFile!.content).toContain('proxy.example.com:8080');
    });

    it('includes proxy auth credentials when provided', () => {
      const state = createInitialWizardState();
      state.services.proxy = {
        hostname: 'proxy.example.com',
        port: 8080,
        username: 'user',
        password: 'pass',
      };

      const files = buildStepConfigFiles(WIZARD_STEPS.SERVICES, state);
      const proxyFile = files.find(f => f.path === 'profile.d/boardingpass-proxy.sh');
      expect(proxyFile).toBeDefined();
      expect(proxyFile!.content).toContain('user:pass@proxy.example.com:8080');
    });

    it('returns no files for step 5 when no enrollment is enabled', () => {
      const state = createInitialWizardState();
      const files = buildStepConfigFiles(WIZARD_STEPS.ENROLLMENT, state);
      expect(files).toHaveLength(0);
    });

    it('builds insights staging file for step 5', () => {
      const state = createInitialWizardState();
      state.enrollment.insights = {
        endpoint: 'https://cert-api.access.redhat.com',
        orgId: 'org123',
        activationKey: 'key456',
      };

      const files = buildStepConfigFiles(WIZARD_STEPS.ENROLLMENT, state);
      const insightsFile = files.find(f => f.path === 'boardingpass/staging/insights.json');
      expect(insightsFile).toBeDefined();
      expect(insightsFile!.mode).toBe(0o600);
      const parsed = JSON.parse(insightsFile!.content);
      expect(parsed.org_id).toBe('org123');
      expect(parsed.activation_key).toBe('key456');
      expect(parsed.disable_remote_management).toBe(false);
    });

    it('builds flightctl staging file with password auth for step 5', () => {
      const state = createInitialWizardState();
      state.enrollment.flightControl = {
        endpoint: 'https://fc.example.com',
        authMethod: 'password' as const,
        token: null,
        username: 'admin',
        password: 'secret',
      };

      const files = buildStepConfigFiles(WIZARD_STEPS.ENROLLMENT, state);
      const fcFile = files.find(f => f.path === 'boardingpass/staging/flightctl.json');
      expect(fcFile).toBeDefined();
      expect(fcFile!.mode).toBe(0o600);
      const parsed = JSON.parse(fcFile!.content);
      expect(parsed.endpoint).toBe('https://fc.example.com');
      expect(parsed.username).toBe('admin');
      expect(parsed.password).toBe('secret');
      expect(parsed.token).toBeUndefined();
    });

    it('builds flightctl staging file with token auth for step 5', () => {
      const state = createInitialWizardState();
      state.enrollment.flightControl = {
        endpoint: 'https://fc.example.com',
        authMethod: 'token' as const,
        token: 'my-bearer-token',
        username: null,
        password: null,
      };

      const files = buildStepConfigFiles(WIZARD_STEPS.ENROLLMENT, state);
      const fcFile = files.find(f => f.path === 'boardingpass/staging/flightctl.json');
      expect(fcFile).toBeDefined();
      const parsed = JSON.parse(fcFile!.content);
      expect(parsed.endpoint).toBe('https://fc.example.com');
      expect(parsed.token).toBe('my-bearer-token');
      expect(parsed.username).toBeUndefined();
      expect(parsed.password).toBeUndefined();
    });

    it('builds both insights and flightctl staging files when both enabled (RHEL 10+)', () => {
      const state = createInitialWizardState();
      state.osVersion = '10.1';
      state.enrollment.insights = {
        endpoint: 'https://cert-api.access.redhat.com',
        orgId: 'org123',
        activationKey: 'key456',
      };
      state.enrollment.flightControl = {
        endpoint: 'https://fc.example.com',
        authMethod: 'password' as const,
        token: null,
        username: 'admin',
        password: 'secret',
      };

      const files = buildStepConfigFiles(WIZARD_STEPS.ENROLLMENT, state);
      expect(files).toHaveLength(2);

      const insightsFile = files.find(f => f.path === 'boardingpass/staging/insights.json');
      expect(insightsFile).toBeDefined();
      expect(insightsFile!.mode).toBe(0o600);
      const insightsParsed = JSON.parse(insightsFile!.content);
      expect(insightsParsed.endpoint).toBe('https://cert-api.access.redhat.com');
      expect(insightsParsed.org_id).toBe('org123');
      expect(insightsParsed.activation_key).toBe('key456');
      expect(insightsParsed.disable_remote_management).toBe(true);

      const fcFile = files.find(f => f.path === 'boardingpass/staging/flightctl.json');
      expect(fcFile).toBeDefined();
      expect(fcFile!.mode).toBe(0o600);
    });

    it('does not set disable_remote_management on RHEL 9 even when both enabled', () => {
      const state = createInitialWizardState();
      state.osVersion = '9.7';
      state.enrollment.insights = {
        endpoint: 'https://cert-api.access.redhat.com',
        orgId: 'org123',
        activationKey: 'key456',
      };
      state.enrollment.flightControl = {
        endpoint: 'https://fc.example.com',
        authMethod: 'password' as const,
        token: null,
        username: 'admin',
        password: 'secret',
      };

      const files = buildStepConfigFiles(WIZARD_STEPS.ENROLLMENT, state);
      const insightsFile = files.find(f => f.path === 'boardingpass/staging/insights.json');
      const insightsParsed = JSON.parse(insightsFile!.content);
      expect(insightsParsed.disable_remote_management).toBe(false);
    });
  });

  describe('per-step command building', () => {
    it('returns set-hostname command for step 1', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';

      const commands = buildStepCommands(WIZARD_STEPS.HOSTNAME, state);
      expect(commands).toHaveLength(1);
      expect(commands[0].id).toBe('set-hostname');
      expect(commands[0].params).toEqual(['my-device']);
    });

    it('returns no commands for step 2', () => {
      const state = createInitialWizardState();
      const commands = buildStepCommands(WIZARD_STEPS.INTERFACE, state);
      expect(commands).toHaveLength(0);
    });

    it('returns reload-connection command for step 3', () => {
      const state = createInitialWizardState();
      state.networkInterface.interfaceName = 'eth0';

      const commands = buildStepCommands(WIZARD_STEPS.ADDRESSING, state);
      expect(commands).toHaveLength(1);
      expect(commands[0].id).toBe('reload-connection');
      expect(commands[0].params).toEqual(['boardingpass-enrollment']);
    });

    it('returns restart-chronyd command for step 4 with manual NTP', () => {
      const state = createInitialWizardState();
      state.services.ntp = { mode: 'manual', servers: ['ntp.example.com'] };

      const commands = buildStepCommands(WIZARD_STEPS.SERVICES, state);
      expect(commands.some(c => c.id === 'restart-chronyd')).toBe(true);
    });

    it('returns no commands for step 4 with automatic NTP and no proxy', () => {
      const state = createInitialWizardState();
      const commands = buildStepCommands(WIZARD_STEPS.SERVICES, state);
      expect(commands).toHaveLength(0);
    });

    it('returns enroll-insights command for step 5 with Insights', () => {
      const state = createInitialWizardState();
      state.enrollment.insights = {
        endpoint: 'https://cert-api.access.redhat.com',
        orgId: 'org123',
        activationKey: 'key456',
      };

      const commands = buildStepCommands(WIZARD_STEPS.ENROLLMENT, state);
      expect(commands.some(c => c.id === 'enroll-insights')).toBe(true);
    });

    it('returns enroll-flightctl command for step 5 with FlightCtl', () => {
      const state = createInitialWizardState();
      state.enrollment.flightControl = {
        endpoint: 'https://fc.example.com',
        authMethod: 'password' as const,
        token: null,
        username: 'admin',
        password: 'secret',
      };

      const commands = buildStepCommands(WIZARD_STEPS.ENROLLMENT, state);
      expect(commands.some(c => c.id === 'enroll-flightctl')).toBe(true);
    });
  });

  describe('per-step immediate apply', () => {
    let mockClient: APIClient;

    beforeEach(() => {
      jest.clearAllMocks();
      mockClient = createMockClient();
      mockSendConfigBundle.mockResolvedValue({ status: 'success' });
      mockExecuteCommand.mockResolvedValue({ exit_code: 0, stdout: '', stderr: '' });
      mockCreateConfigFile.mockImplementation((path, content, mode) => ({
        path,
        content: btoa(content),
        mode: mode || 0o644,
      }));
    });

    it('sends config files and executes commands for step 1', async () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';
      state.applyMode = 'immediate';

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      await act(async () => {
        await result.current.applyStepImmediate(WIZARD_STEPS.HOSTNAME, mockClient);
      });

      expect(mockSendConfigBundle).toHaveBeenCalledTimes(1);
      expect(mockExecuteCommand).toHaveBeenCalledWith(mockClient, 'set-hostname', ['my-device']);
      expect(result.current.state.stepApplyStatus[WIZARD_STEPS.HOSTNAME]?.status).toBe('success');
    });

    it('sets failed status on sendConfigBundle error', async () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';
      state.applyMode = 'immediate';
      mockSendConfigBundle.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      await act(async () => {
        await result.current.applyStepImmediate(WIZARD_STEPS.HOSTNAME, mockClient);
      });

      expect(result.current.state.stepApplyStatus[WIZARD_STEPS.HOSTNAME]?.status).toBe('failed');
      expect(result.current.state.stepApplyStatus[WIZARD_STEPS.HOSTNAME]?.error).toContain(
        'Network error'
      );
    });

    it('sets failed status on command execution error', async () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';
      state.applyMode = 'immediate';
      mockExecuteCommand.mockResolvedValue({
        exit_code: 1,
        stdout: '',
        stderr: 'permission denied',
      });

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      await act(async () => {
        await result.current.applyStepImmediate(WIZARD_STEPS.HOSTNAME, mockClient);
      });

      expect(result.current.state.stepApplyStatus[WIZARD_STEPS.HOSTNAME]?.status).toBe('failed');
      expect(result.current.state.stepApplyStatus[WIZARD_STEPS.HOSTNAME]?.error).toContain(
        'permission denied'
      );
    });

    it('sets applying status during apply', async () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';
      state.applyMode = 'immediate';

      mockSendConfigBundle.mockImplementation(async () => {
        // Capture status during execution (it should be 'applying')
        // This is hard to test synchronously; we verify before/after states
        return { status: 'success' };
      });

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      await act(async () => {
        await result.current.applyStepImmediate(WIZARD_STEPS.HOSTNAME, mockClient);
      });

      // After completion, status should be success
      expect(result.current.state.stepApplyStatus[WIZARD_STEPS.HOSTNAME]?.status).toBe('success');
    });

    it('skips apply when no config files for step 2', async () => {
      const state = createInitialWizardState();
      state.networkInterface.interfaceName = 'eth0';
      state.networkInterface.interfaceType = 'ethernet';
      state.applyMode = 'immediate';

      const { result } = renderHook(() => useConfigWizard(), {
        wrapper: createWrapper(state),
      });

      await act(async () => {
        await result.current.applyStepImmediate(WIZARD_STEPS.INTERFACE, mockClient);
      });

      // No API calls made
      expect(mockSendConfigBundle).not.toHaveBeenCalled();
      expect(mockExecuteCommand).not.toHaveBeenCalled();
      expect(result.current.state.stepApplyStatus[WIZARD_STEPS.INTERFACE]?.status).toBe('success');
    });
  });

  describe('deferred bundle building', () => {
    it('includes all configured files in deferred bundle', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';
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
      state.services = {
        ntp: { mode: 'manual', servers: ['ntp.example.com'] },
        proxy: { hostname: 'proxy.example.com', port: 8080, username: null, password: null },
      };
      state.enrollment.insights = {
        endpoint: 'https://cert-api.access.redhat.com',
        orgId: 'org123',
        activationKey: 'key456',
      };

      const bundle = buildDeferredBundle(state);

      // Should include: hostname, NM connection, NTP config, proxy config, insights staging, systemd service, symlink
      expect(bundle.find(f => f.path === 'hostname')).toBeDefined();
      expect(bundle.find(f => f.path.includes('nmconnection'))).toBeDefined();
      expect(bundle.find(f => f.path === 'chrony.d/boardingpass-ntp.conf')).toBeDefined();
      expect(bundle.find(f => f.path === 'profile.d/boardingpass-proxy.sh')).toBeDefined();
      expect(bundle.find(f => f.path === 'boardingpass/staging/insights.json')).toBeDefined();
      expect(
        bundle.find(f => f.path === 'systemd/system/boardingpass-enroll.service')
      ).toBeDefined();
      expect(
        bundle.find(
          f => f.path === 'systemd/system/multi-user.target.wants/boardingpass-enroll.service'
        )
      ).toBeDefined();
    });

    it('omits optional files when not configured', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';
      state.networkInterface = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };

      const bundle = buildDeferredBundle(state);

      // Should only include hostname and NM connection
      expect(bundle.find(f => f.path === 'hostname')).toBeDefined();
      expect(bundle.find(f => f.path.includes('nmconnection'))).toBeDefined();
      expect(bundle.find(f => f.path.includes('chrony'))).toBeUndefined();
      expect(bundle.find(f => f.path.includes('proxy'))).toBeUndefined();
      expect(bundle.find(f => f.path.includes('staging'))).toBeUndefined();
      expect(bundle.find(f => f.path.includes('systemd'))).toBeUndefined();
    });

    it('includes systemd oneshot only when enrollment is enabled', () => {
      const state = createInitialWizardState();
      state.hostname.hostname = 'my-device';
      state.networkInterface = {
        interfaceName: 'eth0',
        interfaceType: 'ethernet',
        vlanId: null,
        wifi: null,
      };
      state.enrollment.flightControl = {
        endpoint: 'https://fc.example.com',
        authMethod: 'password' as const,
        token: null,
        username: 'admin',
        password: 'secret',
      };

      const bundle = buildDeferredBundle(state);
      const serviceFile = bundle.find(f => f.path === 'systemd/system/boardingpass-enroll.service');
      expect(serviceFile).toBeDefined();
      expect(serviceFile!.content).toContain('After=network-online.target');
      expect(serviceFile!.content).toContain('enroll-flightctl.sh');
    });

    describe('systemd enrollment service unit generation', () => {
      function createEnrollmentState(): WizardState {
        const state = createInitialWizardState();
        state.hostname.hostname = 'my-device';
        state.networkInterface = {
          interfaceName: 'eth0',
          interfaceType: 'ethernet',
          vlanId: null,
          wifi: null,
        };
        return state;
      }

      it('generates service unit with both ExecStart lines when both enrollments enabled', () => {
        const state = createEnrollmentState();
        state.enrollment.insights = {
          endpoint: 'https://cert-api.access.redhat.com',
          orgId: 'org123',
          activationKey: 'key456',
        };
        state.enrollment.flightControl = {
          endpoint: 'https://fc.example.com',
          username: 'admin',
          password: 'secret',
        };

        const bundle = buildDeferredBundle(state);
        const serviceFile = bundle.find(
          f => f.path === 'systemd/system/boardingpass-enroll.service'
        );
        expect(serviceFile).toBeDefined();
        expect(serviceFile!.content).toContain(
          'ExecStart=/usr/libexec/boardingpass/enroll-insights.sh'
        );
        expect(serviceFile!.content).toContain(
          'ExecStart=/usr/libexec/boardingpass/enroll-flightctl.sh'
        );
      });

      it('generates service unit with only insights ExecStart when only insights enabled', () => {
        const state = createEnrollmentState();
        state.enrollment.insights = {
          endpoint: 'https://cert-api.access.redhat.com',
          orgId: 'org123',
          activationKey: 'key456',
        };

        const bundle = buildDeferredBundle(state);
        const serviceFile = bundle.find(
          f => f.path === 'systemd/system/boardingpass-enroll.service'
        );
        expect(serviceFile).toBeDefined();
        expect(serviceFile!.content).toContain(
          'ExecStart=/usr/libexec/boardingpass/enroll-insights.sh'
        );
        expect(serviceFile!.content).not.toContain('enroll-flightctl.sh');
      });

      it('includes ConditionPathExists directive', () => {
        const state = createEnrollmentState();
        state.enrollment.insights = {
          endpoint: 'https://cert-api.access.redhat.com',
          orgId: 'org123',
          activationKey: 'key456',
        };

        const bundle = buildDeferredBundle(state);
        const serviceFile = bundle.find(
          f => f.path === 'systemd/system/boardingpass-enroll.service'
        );
        expect(serviceFile!.content).toContain('ConditionPathExists=/etc/boardingpass/staging');
      });

      it('includes ExecStartPost to self-disable after enrollment', () => {
        const state = createEnrollmentState();
        state.enrollment.flightControl = {
          endpoint: 'https://fc.example.com',
          username: 'admin',
          password: 'secret',
        };

        const bundle = buildDeferredBundle(state);
        const serviceFile = bundle.find(
          f => f.path === 'systemd/system/boardingpass-enroll.service'
        );
        expect(serviceFile!.content).toContain(
          'ExecStartPost=/bin/systemctl disable boardingpass-enroll.service'
        );
      });

      it('includes enablement symlink in deferred bundle', () => {
        const state = createEnrollmentState();
        state.enrollment.insights = {
          endpoint: 'https://cert-api.access.redhat.com',
          orgId: 'org123',
          activationKey: 'key456',
        };

        const bundle = buildDeferredBundle(state);
        const symlink = bundle.find(
          f => f.path === 'systemd/system/multi-user.target.wants/boardingpass-enroll.service'
        );
        expect(symlink).toBeDefined();
        expect(symlink!.content).toContain('[Unit]');
        expect(symlink!.content).toContain('[Service]');
      });

      it('service unit has correct systemd structure', () => {
        const state = createEnrollmentState();
        state.enrollment.insights = {
          endpoint: 'https://cert-api.access.redhat.com',
          orgId: 'org123',
          activationKey: 'key456',
        };

        const bundle = buildDeferredBundle(state);
        const serviceFile = bundle.find(
          f => f.path === 'systemd/system/boardingpass-enroll.service'
        );
        expect(serviceFile).toBeDefined();
        const content = serviceFile!.content;
        expect(content).toContain('[Unit]');
        expect(content).toContain('[Service]');
        expect(content).toContain('Type=oneshot');
        expect(content).toContain('After=network-online.target');
        expect(content).toContain('Wants=network-online.target');
        expect(content).toContain('RemainAfterExit=no');
      });
    });
  });

  describe('parseConnectivityResult', () => {
    it('maps snake_case JSON to camelCase properties', () => {
      const json =
        '{"link_up":true,"ip_assigned":true,"gateway_reachable":true,"dns_resolves":false,"internet_reachable":false}';
      const result = parseConnectivityResult(json);
      expect(result).toEqual({
        linkUp: true,
        ipAssigned: true,
        gatewayReachable: true,
        dnsResolves: false,
        internetReachable: false,
      });
    });

    it('defaults missing fields to false', () => {
      const result = parseConnectivityResult('{}');
      expect(result.linkUp).toBe(false);
      expect(result.ipAssigned).toBe(false);
      expect(result.gatewayReachable).toBe(false);
      expect(result.dnsResolves).toBe(false);
      expect(result.internetReachable).toBe(false);
    });
  });

  describe('formatConnectivityDetail', () => {
    it('shows checkmarks for passing checks', () => {
      const detail = formatConnectivityDetail({
        linkUp: true,
        ipAssigned: true,
        gatewayReachable: true,
        dnsResolves: true,
        internetReachable: true,
      });
      expect(detail).toContain('\u2713 Link up (cable connected)');
      expect(detail).toContain('\u2713 IP address assigned');
      expect(detail).toContain('\u2713 Gateway reachable');
      expect(detail).toContain('\u2713 DNS names resolve');
      expect(detail).toContain('\u2713 Internet reachable');
    });

    it('shows crosses for failing checks', () => {
      const detail = formatConnectivityDetail({
        linkUp: false,
        ipAssigned: false,
        gatewayReachable: false,
        dnsResolves: false,
        internetReachable: false,
      });
      expect(detail).toContain('\u2717 Link down (no cable)');
      expect(detail).toContain('\u2717 No IP address assigned');
      expect(detail).toContain('\u2717 Gateway not reachable');
      expect(detail).toContain('\u2717 DNS resolution failed');
      expect(detail).toContain('\u2717 Internet not reachable');
    });

    it('formats as multi-line string', () => {
      const detail = formatConnectivityDetail({
        linkUp: true,
        ipAssigned: true,
        gatewayReachable: true,
        dnsResolves: false,
        internetReachable: false,
      });
      const lines = detail.split('\n');
      expect(lines).toHaveLength(5);
    });
  });

  describe('isConnectivityFullPass', () => {
    it('returns true when all checks pass', () => {
      expect(
        isConnectivityFullPass({
          linkUp: true,
          ipAssigned: true,
          gatewayReachable: true,
          dnsResolves: true,
          internetReachable: true,
        })
      ).toBe(true);
    });

    it('returns false when any check fails', () => {
      expect(
        isConnectivityFullPass({
          linkUp: true,
          ipAssigned: true,
          gatewayReachable: true,
          dnsResolves: false,
          internetReachable: true,
        })
      ).toBe(false);
    });
  });
});
