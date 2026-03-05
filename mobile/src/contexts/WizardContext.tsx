/**
 * WizardContext
 *
 * Ephemeral state management for the 5-step enrollment configuration wizard.
 * Uses useReducer for predictable state transitions.
 * This context is local to the wizard screen — not part of the global app state.
 */

import React, { createContext, useContext, useReducer, useCallback } from 'react';
import type {
  WizardState,
  HostnameConfig,
  InterfaceConfig,
  AddressingConfig,
  ServicesConfig,
  EnrollmentConfig,
  ApplyStatus,
} from '../types/wizard';
import { createInitialWizardState } from '../types/wizard';

// ── Actions ──

type WizardAction =
  | { type: 'SET_STEP'; payload: number }
  | { type: 'UPDATE_HOSTNAME'; payload: HostnameConfig }
  | { type: 'UPDATE_INTERFACE'; payload: InterfaceConfig }
  | { type: 'UPDATE_ADDRESSING'; payload: AddressingConfig }
  | { type: 'UPDATE_SERVICES'; payload: ServicesConfig }
  | { type: 'UPDATE_ENROLLMENT'; payload: EnrollmentConfig }
  | { type: 'SET_APPLY_MODE'; payload: 'immediate' | 'deferred' }
  | { type: 'SET_SERVICE_INTERFACE'; payload: string }
  | { type: 'SET_APPLY_STATUS'; payload: { step: number; status: ApplyStatus } }
  | { type: 'RESET' };

// ── Reducer ──

function wizardReducer(state: WizardState, action: WizardAction): WizardState {
  switch (action.type) {
    case 'SET_STEP':
      return {
        ...state,
        currentStep: action.payload,
        maxReachedStep: Math.max(state.maxReachedStep, action.payload),
      };

    case 'UPDATE_HOSTNAME':
      return { ...state, hostname: action.payload };

    case 'UPDATE_INTERFACE':
      return { ...state, networkInterface: action.payload };

    case 'UPDATE_ADDRESSING':
      return { ...state, addressing: action.payload };

    case 'UPDATE_SERVICES':
      return { ...state, services: action.payload };

    case 'UPDATE_ENROLLMENT':
      return { ...state, enrollment: action.payload };

    case 'SET_APPLY_MODE':
      return { ...state, applyMode: action.payload };

    case 'SET_SERVICE_INTERFACE':
      return { ...state, serviceInterfaceName: action.payload };

    case 'SET_APPLY_STATUS':
      return {
        ...state,
        stepApplyStatus: {
          ...state.stepApplyStatus,
          [action.payload.step]: action.payload.status,
        },
      };

    case 'RESET':
      return createInitialWizardState();

    default:
      return state;
  }
}

// ── Context ──

interface WizardContextValue {
  state: WizardState;
  setStep: (step: number) => void;
  updateHostname: (config: HostnameConfig) => void;
  updateInterface: (config: InterfaceConfig) => void;
  updateAddressing: (config: AddressingConfig) => void;
  updateServices: (config: ServicesConfig) => void;
  updateEnrollment: (config: EnrollmentConfig) => void;
  setApplyMode: (mode: 'immediate' | 'deferred') => void;
  setServiceInterface: (name: string) => void;
  setApplyStatus: (step: number, status: ApplyStatus) => void;
  reset: () => void;
}

const WizardContext = createContext<WizardContextValue | undefined>(undefined);

// ── Provider ──

interface WizardProviderProps {
  children: React.ReactNode;
  initialState?: WizardState;
}

export function WizardProvider({ children, initialState }: WizardProviderProps) {
  const [state, dispatch] = useReducer(wizardReducer, initialState || createInitialWizardState());

  const setStep = useCallback((step: number) => {
    dispatch({ type: 'SET_STEP', payload: step });
  }, []);

  const updateHostname = useCallback((config: HostnameConfig) => {
    dispatch({ type: 'UPDATE_HOSTNAME', payload: config });
  }, []);

  const updateInterface = useCallback((config: InterfaceConfig) => {
    dispatch({ type: 'UPDATE_INTERFACE', payload: config });
  }, []);

  const updateAddressing = useCallback((config: AddressingConfig) => {
    dispatch({ type: 'UPDATE_ADDRESSING', payload: config });
  }, []);

  const updateServices = useCallback((config: ServicesConfig) => {
    dispatch({ type: 'UPDATE_SERVICES', payload: config });
  }, []);

  const updateEnrollment = useCallback((config: EnrollmentConfig) => {
    dispatch({ type: 'UPDATE_ENROLLMENT', payload: config });
  }, []);

  const setApplyMode = useCallback((mode: 'immediate' | 'deferred') => {
    dispatch({ type: 'SET_APPLY_MODE', payload: mode });
  }, []);

  const setServiceInterface = useCallback((name: string) => {
    dispatch({ type: 'SET_SERVICE_INTERFACE', payload: name });
  }, []);

  const setApplyStatus = useCallback((step: number, status: ApplyStatus) => {
    dispatch({ type: 'SET_APPLY_STATUS', payload: { step, status } });
  }, []);

  const reset = useCallback(() => {
    dispatch({ type: 'RESET' });
  }, []);

  const value: WizardContextValue = {
    state,
    setStep,
    updateHostname,
    updateInterface,
    updateAddressing,
    updateServices,
    updateEnrollment,
    setApplyMode,
    setServiceInterface,
    setApplyStatus,
    reset,
  };

  return <WizardContext.Provider value={value}>{children}</WizardContext.Provider>;
}

// ── Hook ──

export function useWizard(): WizardContextValue {
  const context = useContext(WizardContext);
  if (context === undefined) {
    throw new Error('useWizard must be used within a WizardProvider');
  }
  return context;
}
