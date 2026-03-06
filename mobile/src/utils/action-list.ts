/**
 * Action List Generator
 *
 * Pure function that generates a human-readable, ordered action list
 * from wizard state. Each action describes a discrete operation to be
 * performed during the apply phase on the Review & Apply screen.
 */

import type { PlannedAction, WizardState } from '../types/wizard';
import { WIZARD_STEPS } from '../types/wizard';

/**
 * Build the complete action list from wizard state and apply mode.
 *
 * Returns a deterministic, ordered list of PlannedAction items.
 * The order follows the logical sequence: hostname → interface → wifi →
 * IPv4 → DNS → IPv6 → connectivity check → DNS check → NTP → clock sync →
 * Insights enrollment → Flight Control enrollment.
 */
export function buildActionList(
  state: WizardState,
  applyMode: 'immediate' | 'deferred'
): PlannedAction[] {
  const actions: PlannedAction[] = [];

  // 1. Hostname
  if (state.hostname.hostname !== state.hostname.original) {
    actions.push({
      id: 'hostname',
      description: `Set hostname to "${state.hostname.hostname}"`,
      category: 'command',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.HOSTNAME,
      infoOnly: false,
    });
  } else {
    actions.push({
      id: 'hostname',
      description: `Keep hostname as "${state.hostname.hostname}"`,
      category: 'config',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.HOSTNAME,
      infoOnly: true,
    });
  }

  // 2. Interface selection
  const ifaceType = state.networkInterface.interfaceType || 'ethernet';
  actions.push({
    id: 'interface',
    description: `Use ${ifaceType} interface ${state.networkInterface.interfaceName}`,
    category: 'config',
    status: 'pending',
    detail: null,
    step: WIZARD_STEPS.INTERFACE,
    infoOnly: true,
  });

  // 3. WiFi connection (only if WiFi interface selected)
  if (state.networkInterface.interfaceType === 'wifi' && state.networkInterface.wifi) {
    actions.push({
      id: 'wifi-connect',
      description: `Connect to WiFi network "${state.networkInterface.wifi.ssid}"`,
      category: 'config',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.INTERFACE,
      infoOnly: false,
    });
  }

  // 4. IPv4 configuration
  if (state.addressing.ipv4.method === 'dhcp') {
    actions.push({
      id: 'ipv4-config',
      description: 'Assign IPv4 address via DHCP',
      category: 'config',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.ADDRESSING,
      infoOnly: false,
    });
  } else {
    const parts = [`${state.addressing.ipv4.address}/${state.addressing.ipv4.subnetMask}`];
    if (state.addressing.ipv4.gateway) {
      parts.push(`gateway ${state.addressing.ipv4.gateway}`);
    }
    actions.push({
      id: 'ipv4-config',
      description: `Assign static IPv4 address: ${parts.join(', ')}`,
      category: 'config',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.ADDRESSING,
      infoOnly: false,
    });
  }

  // 5. DNS servers (only if manual DNS configured)
  if (!state.addressing.ipv4.dnsAuto) {
    const servers = [state.addressing.ipv4.dnsPrimary, state.addressing.ipv4.dnsSecondary]
      .filter(Boolean)
      .join(', ');
    actions.push({
      id: 'dns-config',
      description: `Set DNS servers: ${servers}`,
      category: 'config',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.ADDRESSING,
      infoOnly: false,
    });
  }

  // 6. IPv6 configuration (only if not disabled)
  if (state.addressing.ipv6.method !== 'disabled') {
    if (state.addressing.ipv6.method === 'dhcp') {
      actions.push({
        id: 'ipv6-config',
        description: 'Assign IPv6 address via DHCP',
        category: 'config',
        status: 'pending',
        detail: null,
        step: WIZARD_STEPS.ADDRESSING,
        infoOnly: false,
      });
    } else {
      actions.push({
        id: 'ipv6-config',
        description: `Assign static IPv6 address: ${state.addressing.ipv6.address}`,
        category: 'config',
        status: 'pending',
        detail: null,
        step: WIZARD_STEPS.ADDRESSING,
        infoOnly: false,
      });
    }
  }

  // 7. Connectivity check (immediate mode only)
  // Runs gateway ping, DNS resolution, and internet reachability in one call.
  if (applyMode === 'immediate') {
    actions.push({
      id: 'connectivity-check',
      description: 'Test network connectivity',
      category: 'check',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.ADDRESSING,
      infoOnly: false,
    });
  }

  // 9. NTP configuration
  if (state.services.ntp.mode === 'automatic') {
    actions.push({
      id: 'ntp-config',
      description: 'Use automatic time servers',
      category: 'config',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.SERVICES,
      infoOnly: false,
    });
  } else {
    actions.push({
      id: 'ntp-config',
      description: `Set manual NTP servers: ${state.services.ntp.servers.join(', ')}`,
      category: 'config',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.SERVICES,
      infoOnly: false,
    });
  }

  // 10. Clock sync wait
  if (applyMode === 'immediate') {
    actions.push({
      id: 'clock-sync',
      description: 'Wait for clock to be synchronised',
      category: 'wait',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.SERVICES,
      infoOnly: false,
    });
  } else {
    actions.push({
      id: 'clock-sync',
      description: 'Wait for clock to be synchronised after reboot',
      category: 'wait',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.SERVICES,
      infoOnly: true,
    });
  }

  // 11. Insights enrollment (if configured)
  if (state.enrollment.insights) {
    actions.push({
      id: 'enroll-insights',
      description: 'Enroll into Red Hat Insights',
      category: 'command',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.ENROLLMENT,
      infoOnly: false,
    });
  }

  // 12. Flight Control enrollment (if configured)
  if (state.enrollment.flightControl) {
    actions.push({
      id: 'enroll-flightctl',
      description: 'Enroll into Flight Control',
      category: 'command',
      status: 'pending',
      detail: null,
      step: WIZARD_STEPS.ENROLLMENT,
      infoOnly: false,
    });
  }

  return actions;
}
