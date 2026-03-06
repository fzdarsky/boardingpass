/**
 * Wizard Types
 *
 * TypeScript types for the 6-step enrollment configuration wizard.
 * All types are ephemeral — wizard state is not persisted.
 *
 * Steps: Hostname → Interface → Addressing → Services → Enrollment → Review
 */

// ── Step 1: Hostname ──

export interface HostnameConfig {
  hostname: string;
  original: string;
}

// ── Step 2: Network Interface ──

export interface InterfaceConfig {
  interfaceName: string;
  interfaceType: string;
  vlanId: number | null;
  wifi: WiFiConfig | null;
}

export interface WiFiConfig {
  ssid: string;
  bssid: string;
  security: string;
  password: string | null;
}

/** Read-only data from wifi-scan command, deduplicated by SSID */
export interface WiFiNetwork {
  device: string;
  ssid: string;
  bssid: string;
  signal: number;
  security: string;
  channel: number;
  frequency: number;
  band: string;
  bands: string[];
  rate: string;
}

// ── Step 3: Addressing ──

export interface AddressingConfig {
  ipv4: IPv4Config;
  ipv6: IPv6Config;
}

export interface IPv4Config {
  method: 'dhcp' | 'static';
  address: string | null;
  subnetMask: string | null;
  gateway: string | null;
  dnsAuto: boolean;
  dnsPrimary: string | null;
  dnsSecondary: string | null;
}

export interface IPv6Config {
  method: 'dhcp' | 'static' | 'disabled';
  address: string | null;
  gateway: string | null;
  dnsAuto: boolean;
  dnsPrimary: string | null;
  dnsSecondary: string | null;
}

// ── Step 4: Services ──

export interface ServicesConfig {
  ntp: NTPConfig;
  proxy: ProxyConfig | null;
}

export interface NTPConfig {
  mode: 'automatic' | 'manual';
  servers: string[];
}

export interface ProxyConfig {
  hostname: string;
  port: number;
  username: string | null;
  password: string | null;
}

// ── Step 5: Enrollment ──

export interface EnrollmentConfig {
  insights: InsightsConfig | null;
  flightControl: FlightControlConfig | null;
}

export interface InsightsConfig {
  endpoint: string;
  orgId: string;
  activationKey: string;
}

export interface FlightControlConfig {
  endpoint: string;
  username: string;
  password: string;
}

// ── Planned Action (Review & Apply) ──

export interface PlannedAction {
  id: string;
  description: string;
  category: 'config' | 'command' | 'check' | 'wait';
  status: 'pending' | 'running' | 'success' | 'failed' | 'skipped';
  detail: string | null;
  step: number;
  infoOnly: boolean;
}

// ── Apply Status ──

export interface ApplyStatus {
  status: 'pending' | 'applying' | 'success' | 'failed';
  error: string | null;
  connectivityResult: ConnectivityResult | null;
}

/** Result from connectivity-test command */
export interface ConnectivityResult {
  ipAssigned: boolean;
  gatewayReachable: boolean;
  dnsResolves: boolean;
  internetReachable: boolean;
}

// ── Wizard State ──

export interface WizardState {
  currentStep: number;
  maxReachedStep: number;
  applyMode: 'immediate' | 'deferred' | null;
  serviceInterfaceName: string | null;
  hostname: HostnameConfig;
  networkInterface: InterfaceConfig;
  addressing: AddressingConfig;
  services: ServicesConfig;
  enrollment: EnrollmentConfig;
  stepApplyStatus: Record<number, ApplyStatus>;
  actionList: PlannedAction[];
  applyInProgress: boolean;
}

// ── Constants ──

export const WIZARD_STEPS = {
  HOSTNAME: 1,
  INTERFACE: 2,
  ADDRESSING: 3,
  SERVICES: 4,
  ENROLLMENT: 5,
  REVIEW: 6,
} as const;

export const STEP_LABELS: Record<number, string> = {
  [WIZARD_STEPS.HOSTNAME]: 'Hostname',
  [WIZARD_STEPS.INTERFACE]: 'Interface',
  [WIZARD_STEPS.ADDRESSING]: 'Addressing',
  [WIZARD_STEPS.SERVICES]: 'Services',
  [WIZARD_STEPS.ENROLLMENT]: 'Enrollment',
  [WIZARD_STEPS.REVIEW]: 'Review',
};

export const TOTAL_STEPS = 6;

export const DEFAULT_INSIGHTS_ENDPOINT = 'https://cert-api.access.redhat.com';

// ── Initial State Factory ──

export function createInitialWizardState(): WizardState {
  return {
    currentStep: WIZARD_STEPS.HOSTNAME,
    maxReachedStep: WIZARD_STEPS.HOSTNAME,
    applyMode: null,
    serviceInterfaceName: null,
    hostname: { hostname: '', original: '' },
    networkInterface: {
      interfaceName: '',
      interfaceType: '',
      vlanId: null,
      wifi: null,
    },
    addressing: {
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
    },
    services: {
      ntp: { mode: 'automatic', servers: [] },
      proxy: null,
    },
    enrollment: {
      insights: null,
      flightControl: null,
    },
    stepApplyStatus: {},
    actionList: [],
    applyInProgress: false,
  };
}
