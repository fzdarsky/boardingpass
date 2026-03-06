/**
 * useConfigWizard Hook
 *
 * Encapsulates wizard step validation logic, per-step data access,
 * navigation guards, step completion checks, apply mode detection,
 * and configuration apply orchestration for both immediate and deferred modes.
 *
 * This hook reads from WizardContext and provides validation + navigation +
 * apply helpers.
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
import type { WizardState, ConnectivityResult, PlannedAction } from '../types/wizard';
import { generateNmConnection, getConnectionPath } from '../utils/nm-connection';
import { buildActionList } from '../utils/action-list';
import { sendConfigBundle, createConfigFile } from '../services/api/configure';
import { executeCommand } from '../services/api/command';
import { completeProvisioning } from '../services/api/complete';
import { getSystemInfo } from '../services/api/info';
import type { APIClient } from '../services/api/client';

const CONNECTION_NAME = 'boardingpass-enrollment';

export interface StepValidation {
  isValid: boolean;
  errors: string[];
}

interface RawConfigFile {
  path: string;
  content: string;
  mode?: number;
}

interface StepCommand {
  id: string;
  params?: string[];
}

// ── Pure Functions (exported for testing) ──

/**
 * Build the configuration files for a given wizard step.
 * Returns raw (unencoded) file content suitable for createConfigFile().
 */
export function buildStepConfigFiles(step: number, state: WizardState): RawConfigFile[] {
  const files: RawConfigFile[] = [];

  switch (step) {
    case WIZARD_STEPS.HOSTNAME: {
      if (state.hostname.hostname !== state.hostname.original) {
        files.push({ path: 'hostname', content: `${state.hostname.hostname}\n` });
      }
      break;
    }

    case WIZARD_STEPS.INTERFACE:
      // Interface selection only — no config files
      break;

    case WIZARD_STEPS.ADDRESSING: {
      const nmContent = generateNmConnection({
        interfaceConfig: state.networkInterface,
        addressing: state.addressing,
        connectionName: CONNECTION_NAME,
      });
      // NM requires 0600 on keyfiles — it silently ignores world-readable ones
      files.push({ path: getConnectionPath(CONNECTION_NAME), content: nmContent, mode: 0o600 });
      break;
    }

    case WIZARD_STEPS.SERVICES: {
      // NTP config (manual only)
      if (state.services.ntp.mode === 'manual' && state.services.ntp.servers.length > 0) {
        const ntpLines = state.services.ntp.servers.map(s => `server ${s} iburst`);
        files.push({
          path: 'chrony.d/boardingpass-ntp.conf',
          content: ntpLines.join('\n') + '\n',
        });
      }

      // Proxy config
      if (state.services.proxy) {
        files.push({
          path: 'profile.d/boardingpass-proxy.sh',
          content: buildProxyScript(state.services.proxy),
        });
      }
      break;
    }

    case WIZARD_STEPS.ENROLLMENT: {
      if (state.enrollment.insights) {
        files.push({
          path: 'boardingpass/staging/insights.json',
          content: JSON.stringify({
            endpoint: state.enrollment.insights.endpoint,
            org_id: state.enrollment.insights.orgId,
            activation_key: state.enrollment.insights.activationKey,
            disable_remote_management: state.enrollment.flightControl !== null,
          }),
          mode: 0o600,
        });
      }

      if (state.enrollment.flightControl) {
        files.push({
          path: 'boardingpass/staging/flightctl.json',
          content: JSON.stringify({
            endpoint: state.enrollment.flightControl.endpoint,
            username: state.enrollment.flightControl.username,
            password: state.enrollment.flightControl.password,
          }),
          mode: 0o600,
        });
      }
      break;
    }
  }

  return files;
}

/**
 * Get the commands to execute after applying config for a given step.
 */
export function buildStepCommands(step: number, state: WizardState): StepCommand[] {
  const commands: StepCommand[] = [];

  switch (step) {
    case WIZARD_STEPS.HOSTNAME:
      if (state.hostname.hostname !== state.hostname.original) {
        commands.push({ id: 'set-hostname', params: [state.hostname.hostname] });
      }
      break;

    case WIZARD_STEPS.INTERFACE:
      // No commands for interface selection
      break;

    case WIZARD_STEPS.ADDRESSING:
      commands.push({ id: 'reload-connection', params: [CONNECTION_NAME] });
      break;

    case WIZARD_STEPS.SERVICES:
      if (state.services.ntp.mode === 'manual' || state.services.proxy) {
        commands.push({ id: 'restart-chronyd' });
      }
      break;

    case WIZARD_STEPS.ENROLLMENT:
      if (state.enrollment.insights) {
        commands.push({ id: 'enroll-insights' });
      }
      if (state.enrollment.flightControl) {
        commands.push({ id: 'enroll-flightctl' });
      }
      break;
  }

  return commands;
}

/**
 * Build the complete deferred configuration bundle (all steps combined).
 * Includes systemd oneshot service for enrollment if any enrollment is configured.
 */
export function buildDeferredBundle(state: WizardState): RawConfigFile[] {
  const files: RawConfigFile[] = [];

  // Collect files from all steps
  for (let step = WIZARD_STEPS.HOSTNAME; step <= TOTAL_STEPS; step++) {
    files.push(...buildStepConfigFiles(step, state));
  }

  // Add systemd oneshot service for enrollment (deferred mode only)
  const hasEnrollment = state.enrollment.insights || state.enrollment.flightControl;
  if (hasEnrollment) {
    files.push({
      path: 'systemd/system/boardingpass-enroll.service',
      content: buildEnrollmentServiceUnit(state),
    });
    // Symlink to enable the service (written as a regular file — NM keyfile format)
    files.push({
      path: 'systemd/system/multi-user.target.wants/boardingpass-enroll.service',
      content: buildEnrollmentServiceUnit(state),
    });
  }

  return files;
}

// ── Helper Functions ──

function buildProxyScript(proxy: {
  hostname: string;
  port: number;
  username: string | null;
  password: string | null;
}): string {
  const auth = proxy.username && proxy.password ? `${proxy.username}:${proxy.password}@` : '';
  const proxyUrl = `http://${auth}${proxy.hostname}:${proxy.port}`;
  return [
    '# BoardingPass proxy configuration',
    `export http_proxy=${proxyUrl}`,
    `export https_proxy=${proxyUrl}`,
    `export HTTP_PROXY=${proxyUrl}`,
    `export HTTPS_PROXY=${proxyUrl}`,
    'export no_proxy=localhost,127.0.0.1,::1',
    'export NO_PROXY=localhost,127.0.0.1,::1',
    '',
  ].join('\n');
}

function buildEnrollmentServiceUnit(state: WizardState): string {
  const execLines: string[] = [];
  if (state.enrollment.insights) {
    execLines.push('ExecStart=/usr/libexec/boardingpass/enroll-insights.sh');
  }
  if (state.enrollment.flightControl) {
    execLines.push('ExecStart=/usr/libexec/boardingpass/enroll-flightctl.sh');
  }

  return [
    '[Unit]',
    'Description=BoardingPass post-boot enrollment',
    'After=network-online.target',
    'Wants=network-online.target',
    'ConditionPathExists=/etc/boardingpass/staging',
    '',
    '[Service]',
    'Type=oneshot',
    ...execLines,
    'ExecStartPost=/bin/systemctl disable boardingpass-enroll.service',
    'RemainAfterExit=no',
    '',
  ].join('\n');
}

// ── Hook ──

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
            state.networkInterface.interfaceType === 'wifi' &&
            !state.networkInterface.wifi?.ssid
          ) {
            errors.push('Please select a WiFi network');
          }
          if (
            state.networkInterface.interfaceType === 'wifi' &&
            state.networkInterface.wifi &&
            state.networkInterface.wifi.security !== 'open' &&
            state.networkInterface.wifi.security !== '--' &&
            !state.networkInterface.wifi.password
          ) {
            errors.push('WiFi password is required for secured networks');
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
   * When leaving Step 2, determines apply mode by comparing selected
   * interface against the service interface.
   * Returns validation result — caller should show errors if invalid.
   */
  const goNext = useCallback((): StepValidation => {
    const validation = validateStep(state.currentStep);
    if (validation.isValid && state.currentStep < TOTAL_STEPS) {
      // Determine apply mode when leaving Step 2 (interface selection)
      if (state.currentStep === WIZARD_STEPS.INTERFACE) {
        const mode =
          !state.serviceInterfaceName ||
          state.networkInterface.interfaceName === state.serviceInterfaceName
            ? 'deferred'
            : 'immediate';
        wizard.setApplyMode(mode);
      }
      wizard.setStep(state.currentStep + 1);
    }
    return validation;
  }, [
    state.currentStep,
    state.serviceInterfaceName,
    state.networkInterface.interfaceName,
    validateStep,
    wizard,
  ]);

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

  /**
   * Apply a single step's configuration in immediate mode.
   * Sends config files via /configure, then executes step commands via /command.
   * Updates stepApplyStatus in context.
   */
  const applyStepImmediate = useCallback(
    async (step: number, client: APIClient): Promise<string | null> => {
      const configFiles = buildStepConfigFiles(step, state);
      const commands = buildStepCommands(step, state);

      // Nothing to apply for this step
      if (configFiles.length === 0 && commands.length === 0) {
        wizard.setApplyStatus(step, { status: 'success', error: null, connectivityResult: null });
        return null;
      }

      wizard.setApplyStatus(step, { status: 'applying', error: null, connectivityResult: null });

      try {
        // Send config files
        if (configFiles.length > 0) {
          const encoded = configFiles.map(f => createConfigFile(f.path, f.content, f.mode));
          await sendConfigBundle(client, encoded);
        }

        // Execute commands
        for (const cmd of commands) {
          const result = await executeCommand(client, cmd.id, cmd.params);
          if (result.exit_code !== 0) {
            throw new Error(`Command '${cmd.id}' failed: ${result.stderr || 'non-zero exit code'}`);
          }
        }

        // Run connectivity test after addressing step
        let connectivityResult: ConnectivityResult | null = null;
        if (step === WIZARD_STEPS.ADDRESSING) {
          try {
            const gateway = state.addressing.ipv4.gateway || '';
            const iface = state.networkInterface.interfaceName;
            if (gateway && iface) {
              const testResult = await executeCommand(client, 'connectivity-test', [
                iface,
                gateway,
              ]);
              if (testResult.exit_code === 0 && testResult.stdout) {
                connectivityResult = JSON.parse(testResult.stdout);
              }
            }
          } catch {
            // Connectivity test is informational — don't fail the step
          }
        }

        wizard.setApplyStatus(step, {
          status: 'success',
          error: null,
          connectivityResult,
        });
        return null;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Apply failed';
        wizard.setApplyStatus(step, {
          status: 'failed',
          error: message,
          connectivityResult: null,
        });
        return message;
      }
    },
    [state, wizard]
  );

  /**
   * Apply the entire deferred configuration bundle.
   * Sends all config files in a single /configure call, then calls /complete with reboot.
   */
  const applyDeferred = useCallback(
    async (client: APIClient): Promise<void> => {
      const rawFiles = buildDeferredBundle(state);
      const encoded = rawFiles.map(f => createConfigFile(f.path, f.content, f.mode));

      await sendConfigBundle(client, encoded);
      await completeProvisioning(client, true);
    },
    [state]
  );

  /**
   * Generate the action list for the current wizard state and apply mode.
   * Stores the generated list in wizard context.
   */
  const generateActionList = useCallback((): PlannedAction[] => {
    const mode = state.applyMode || 'immediate';
    const actions = buildActionList(state, mode);
    wizard.setActionList(actions);
    return actions;
  }, [state, wizard]);

  /**
   * Apply all actions sequentially in immediate mode.
   * Iterates through steps, sending config files and executing commands.
   * Updates per-action status in context for real-time UI feedback.
   * Halts on first failure, marking remaining actions as skipped.
   */
  const applyAllImmediate = useCallback(
    async (client: APIClient): Promise<void> => {
      const actions = state.actionList;
      if (actions.length === 0) return;

      wizard.setApplyInProgress(true);

      try {
        // Group actions by step for batch config/command operations
        const stepsToApply = [
          WIZARD_STEPS.ADDRESSING,
          WIZARD_STEPS.SERVICES,
          WIZARD_STEPS.ENROLLMENT,
        ];

        // Send config files for addressing + services + enrollment steps
        for (const step of stepsToApply) {
          const configFiles = buildStepConfigFiles(step, state);
          if (configFiles.length > 0) {
            const encoded = configFiles.map(f => createConfigFile(f.path, f.content, f.mode));
            await sendConfigBundle(client, encoded);
          }

          // Execute commands for this step
          const commands = buildStepCommands(step, state);
          for (const cmd of commands) {
            // Find matching action to update status
            const matchingAction = actions.find(
              a => !a.infoOnly && a.step === step && a.status === 'pending'
            );
            if (matchingAction) {
              wizard.updateActionStatus(matchingAction.id, 'running');
            }

            const result = await executeCommand(client, cmd.id, cmd.params);
            if (result.exit_code !== 0) {
              const errMsg = result.stderr || 'non-zero exit code';
              if (matchingAction) {
                wizard.updateActionStatus(matchingAction.id, 'failed', errMsg);
              }
              // Mark remaining pending actions as skipped
              for (const a of actions) {
                if (a.status === 'pending' && !a.infoOnly) {
                  wizard.updateActionStatus(a.id, 'skipped');
                }
              }
              throw new Error(`Command '${cmd.id}' failed: ${errMsg}`);
            }

            if (matchingAction) {
              wizard.updateActionStatus(matchingAction.id, 'success');
            }
          }

          // Mark config-only actions for this step as success
          for (const a of actions) {
            if (a.step === step && a.status === 'pending' && !a.infoOnly) {
              wizard.updateActionStatus(a.id, 'success');
            }
          }
        }

        // Connectivity check
        const connectivityAction = actions.find(a => a.id === 'connectivity-check');
        if (connectivityAction && !connectivityAction.infoOnly) {
          wizard.updateActionStatus('connectivity-check', 'running');
          try {
            const gateway = state.addressing.ipv4.gateway || '';
            const iface = state.networkInterface.interfaceName;
            if (gateway && iface) {
              const testResult = await executeCommand(client, 'connectivity-test', [
                iface,
                gateway,
              ]);
              if (testResult.exit_code === 0 && testResult.stdout) {
                const result: ConnectivityResult = JSON.parse(testResult.stdout);
                const detail = result.gatewayReachable
                  ? 'Gateway reachable'
                  : 'Gateway unreachable';
                wizard.updateActionStatus('connectivity-check', 'success', detail);
              } else {
                wizard.updateActionStatus('connectivity-check', 'success', 'Test completed');
              }
            } else {
              wizard.updateActionStatus('connectivity-check', 'success', 'Skipped (no gateway)');
            }
          } catch {
            // Connectivity test is informational — don't fail
            wizard.updateActionStatus('connectivity-check', 'success', 'Could not verify');
          }
        }

        // DNS resolution check
        const dnsCheckAction = actions.find(a => a.id === 'dns-check');
        if (dnsCheckAction && !dnsCheckAction.infoOnly) {
          wizard.updateActionStatus('dns-check', 'running');
          try {
            const dnsResult = await executeCommand(client, 'connectivity-test', [
              state.networkInterface.interfaceName,
              state.addressing.ipv4.gateway || '',
            ]);
            wizard.updateActionStatus(
              'dns-check',
              'success',
              dnsResult.exit_code === 0 ? 'DNS resolution OK' : 'DNS check completed'
            );
          } catch {
            wizard.updateActionStatus('dns-check', 'success', 'Could not verify');
          }
        }

        // Clock sync wait (poll /info for up to 30 seconds)
        const clockAction = actions.find(a => a.id === 'clock-sync');
        if (clockAction && !clockAction.infoOnly) {
          wizard.updateActionStatus('clock-sync', 'running');
          let synced = false;
          const maxAttempts = 10;
          const pollInterval = 3000;

          for (let i = 0; i < maxAttempts; i++) {
            try {
              const info = await getSystemInfo(client);
              if (info.os.clock_synchronized) {
                synced = true;
                break;
              }
            } catch {
              // Poll failure is not fatal
            }
            if (i < maxAttempts - 1) {
              await new Promise(resolve => setTimeout(resolve, pollInterval));
            }
          }

          wizard.updateActionStatus(
            'clock-sync',
            'success',
            synced ? 'Clock synchronised' : 'Clock not yet synchronised (continuing)'
          );
        }

        // Mark info-only actions as success
        for (const a of actions) {
          if (a.infoOnly && a.status === 'pending') {
            wizard.updateActionStatus(a.id, 'success');
          }
        }

        // All actions complete — call /complete
        await completeProvisioning(client, false);
      } finally {
        wizard.setApplyInProgress(false);
      }
    },
    [state, wizard]
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
    applyStepImmediate,
    applyDeferred,
    generateActionList,
    applyAllImmediate,
  };
}
