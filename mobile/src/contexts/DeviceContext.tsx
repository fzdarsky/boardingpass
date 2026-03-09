import React, { createContext, useContext, useReducer, useCallback } from 'react';
import { Device } from '@/types/device';
import { deduplicateDevices } from '@/services/discovery/preference';

interface DeviceState {
  devices: Device[];
  selectedDeviceId: string | null;
  isScanning: boolean;
}

type DeviceAction =
  | { type: 'ADD_DEVICE'; payload: Device }
  | { type: 'UPDATE_DEVICE'; payload: { id: string; updates: Partial<Device> } }
  | { type: 'REMOVE_DEVICE'; payload: string }
  | { type: 'SELECT_DEVICE'; payload: string | null }
  | { type: 'SET_SCANNING'; payload: boolean }
  | { type: 'CLEAR_DEVICES' };

interface DeviceContextValue {
  state: DeviceState;
  addDevice: (device: Device) => void;
  updateDevice: (id: string, updates: Partial<Device>) => void;
  removeDevice: (id: string) => void;
  selectDevice: (id: string | null) => void;
  setScanning: (scanning: boolean) => void;
  clearDevices: () => void;
  getDeviceById: (id: string) => Device | undefined;
}

const DeviceContext = createContext<DeviceContextValue | undefined>(undefined);

const initialState: DeviceState = {
  devices: [],
  selectedDeviceId: null,
  isScanning: false,
};

function deviceReducer(state: DeviceState, action: DeviceAction): DeviceState {
  switch (action.type) {
    case 'ADD_DEVICE': {
      const updated = [...state.devices, action.payload];
      return { ...state, devices: deduplicateDevices(updated) };
    }

    case 'UPDATE_DEVICE': {
      const devices = state.devices.map(device =>
        device.id === action.payload.id ? { ...device, ...action.payload.updates } : device
      );
      return { ...state, devices };
    }

    case 'REMOVE_DEVICE': {
      const devices = state.devices.filter(device => device.id !== action.payload);
      const selectedDeviceId =
        state.selectedDeviceId === action.payload ? null : state.selectedDeviceId;
      return { ...state, devices, selectedDeviceId };
    }

    case 'SELECT_DEVICE':
      return { ...state, selectedDeviceId: action.payload };

    case 'SET_SCANNING':
      return { ...state, isScanning: action.payload };

    case 'CLEAR_DEVICES':
      return { ...state, devices: [], selectedDeviceId: null };

    default:
      return state;
  }
}

export function DeviceProvider({ children }: { children: React.ReactNode }) {
  const [state, dispatch] = useReducer(deviceReducer, initialState);

  const addDevice = useCallback((device: Device) => {
    dispatch({ type: 'ADD_DEVICE', payload: device });
  }, []);

  const updateDevice = useCallback((id: string, updates: Partial<Device>) => {
    dispatch({ type: 'UPDATE_DEVICE', payload: { id, updates } });
  }, []);

  const removeDevice = useCallback((id: string) => {
    dispatch({ type: 'REMOVE_DEVICE', payload: id });
  }, []);

  const selectDevice = useCallback((id: string | null) => {
    dispatch({ type: 'SELECT_DEVICE', payload: id });
  }, []);

  const setScanning = useCallback((scanning: boolean) => {
    dispatch({ type: 'SET_SCANNING', payload: scanning });
  }, []);

  const clearDevices = useCallback(() => {
    dispatch({ type: 'CLEAR_DEVICES' });
  }, []);

  const getDeviceById = useCallback(
    (id: string) => {
      return state.devices.find(device => device.id === id);
    },
    [state.devices]
  );

  const value: DeviceContextValue = {
    state,
    addDevice,
    updateDevice,
    removeDevice,
    selectDevice,
    setScanning,
    clearDevices,
    getDeviceById,
  };

  return <DeviceContext.Provider value={value}>{children}</DeviceContext.Provider>;
}

export function useDevice() {
  const context = useContext(DeviceContext);
  if (context === undefined) {
    throw new Error('useDevice must be used within a DeviceProvider');
  }
  return context;
}
