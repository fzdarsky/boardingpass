/**
 * useDeviceInfo Hook
 *
 * Provides device information retrieval functionality for authenticated BoardingPass devices.
 * Fetches both system information (/info) and network configuration (/network) in parallel.
 *
 * Usage:
 * ```tsx
 * const { systemInfo, networkConfig, isLoading, error, refetch } = useDeviceInfo(client);
 *
 * // Data is automatically fetched on mount
 * // Use refetch() to manually reload data
 * ```
 *
 * Related: T083 - Create useDeviceInfo hook
 */

import { useState, useEffect, useCallback } from 'react';
import type { APIClient } from '../services/api/client';
import { getSystemInfo, type SystemInfo } from '../services/api/info';
import { getNetworkConfig, type NetworkConfig } from '../services/api/network';

/**
 * Loading state for individual endpoints
 */
interface LoadingState {
  info: boolean;
  network: boolean;
}

/**
 * Error state for individual endpoints
 */
interface ErrorState {
  info: Error | null;
  network: Error | null;
}

/**
 * Device information state
 */
export interface DeviceInfoState {
  /** System information from /info endpoint */
  systemInfo: SystemInfo | null;

  /** Network configuration from /network endpoint */
  networkConfig: NetworkConfig | null;

  /** Overall loading state (true if any endpoint is loading) */
  isLoading: boolean;

  /** Individual loading states for each endpoint */
  loadingStates: LoadingState;

  /** Overall error (null if no errors, or combined error message) */
  error: Error | null;

  /** Individual errors for each endpoint */
  errors: ErrorState;

  /** Whether data has been loaded at least once */
  hasData: boolean;

  /** Whether any endpoint has partial data (some succeeded, some failed) */
  hasPartialData: boolean;
}

/**
 * Device information hook result
 */
export interface UseDeviceInfoResult extends DeviceInfoState {
  /**
   * Manually refetch device information
   *
   * Fetches both /info and /network in parallel. If one fails, the other
   * may still succeed (partial data handling per T093).
   *
   * @returns Promise that resolves when fetch completes
   */
  refetch: () => Promise<void>;

  /**
   * Clear the current error state
   */
  clearError: () => void;

  /**
   * Clear all data and reset to initial state
   */
  reset: () => void;
}

/**
 * Device information hook
 *
 * Automatically fetches system info and network config on mount for authenticated client.
 * Implements loading states (T090), error handling (T091), retry mechanism (T092),
 * and partial data handling (T093).
 *
 * @param client - Authenticated API client
 * @param options - Hook options
 * @param options.autoFetch - Whether to automatically fetch on mount (default: true)
 * @returns Device information state and operations
 */
export function useDeviceInfo(
  client: APIClient | null,
  options: { autoFetch?: boolean } = {}
): UseDeviceInfoResult {
  const { autoFetch = true } = options;

  const [state, setState] = useState<DeviceInfoState>({
    systemInfo: null,
    networkConfig: null,
    isLoading: false,
    loadingStates: {
      info: false,
      network: false,
    },
    error: null,
    errors: {
      info: null,
      network: null,
    },
    hasData: false,
    hasPartialData: false,
  });

  /**
   * Fetch device information (both /info and /network in parallel)
   */
  const fetchDeviceInfo = useCallback(async () => {
    if (!client) {
      setState(prev => ({
        ...prev,
        error: new Error('API client not provided'),
      }));
      return;
    }

    // Set loading states
    setState(prev => ({
      ...prev,
      isLoading: true,
      loadingStates: {
        info: true,
        network: true,
      },
      error: null,
      errors: {
        info: null,
        network: null,
      },
    }));

    // Fetch both endpoints in parallel (per tasks.md: fetch info + network together)
    const [infoResult, networkResult] = await Promise.allSettled([
      getSystemInfo(client),
      getNetworkConfig(client),
    ]);

    // Process results
    const newState: Partial<DeviceInfoState> = {
      isLoading: false,
      loadingStates: {
        info: false,
        network: false,
      },
    };

    // Process /info result
    if (infoResult.status === 'fulfilled') {
      newState.systemInfo = infoResult.value;
      newState.errors = { ...state.errors, info: null };
    } else {
      newState.errors = { ...state.errors, info: infoResult.reason };
    }

    // Process /network result
    if (networkResult.status === 'fulfilled') {
      newState.networkConfig = networkResult.value;
      newState.errors = { ...newState.errors, network: null };
    } else {
      newState.errors = { ...newState.errors, network: networkResult.reason };
    }

    // Determine overall state
    const hasSystemInfo = infoResult.status === 'fulfilled';
    const hasNetworkConfig = networkResult.status === 'fulfilled';

    newState.hasData = hasSystemInfo || hasNetworkConfig;
    newState.hasPartialData = hasSystemInfo !== hasNetworkConfig; // One succeeded, one failed

    // Set overall error (combined message if both failed, or null if at least one succeeded)
    if (!hasSystemInfo && !hasNetworkConfig) {
      // Both failed - create combined error
      const infoError = newState.errors?.info?.message || 'Unknown error';
      const networkError = newState.errors?.network?.message || 'Unknown error';
      newState.error = new Error(
        `Failed to fetch device information. System info: ${infoError}. Network config: ${networkError}`
      );
    } else if (!hasSystemInfo) {
      // Only /info failed
      newState.error = new Error(
        `Failed to fetch system information: ${newState.errors?.info?.message}`
      );
    } else if (!hasNetworkConfig) {
      // Only /network failed
      newState.error = new Error(
        `Failed to fetch network configuration: ${newState.errors?.network?.message}`
      );
    } else {
      // Both succeeded
      newState.error = null;
    }

    setState(prev => ({ ...prev, ...newState }));
  }, [client]);

  /**
   * Manually refetch device information
   */
  const refetch = useCallback(async () => {
    await fetchDeviceInfo();
  }, [fetchDeviceInfo]);

  /**
   * Clear the current error state
   */
  const clearError = useCallback(() => {
    setState(prev => ({
      ...prev,
      error: null,
      errors: {
        info: null,
        network: null,
      },
    }));
  }, []);

  /**
   * Reset to initial state
   */
  const reset = useCallback(() => {
    setState({
      systemInfo: null,
      networkConfig: null,
      isLoading: false,
      loadingStates: {
        info: false,
        network: false,
      },
      error: null,
      errors: {
        info: null,
        network: null,
      },
      hasData: false,
      hasPartialData: false,
    });
  }, []);

  /**
   * Auto-fetch on mount if client is available and autoFetch is enabled
   */
  useEffect(() => {
    if (autoFetch && client) {
      fetchDeviceInfo();
    }
  }, [autoFetch, client, fetchDeviceInfo]);

  return {
    ...state,
    refetch,
    clearError,
    reset,
  };
}

/**
 * Helper hook for checking if specific data is available
 *
 * @param deviceInfo - Device info state from useDeviceInfo
 * @returns Boolean flags for data availability
 */
export function useDeviceInfoAvailability(deviceInfo: DeviceInfoState) {
  return {
    /** Whether system info is available */
    hasSystemInfo: deviceInfo.systemInfo !== null,

    /** Whether network config is available */
    hasNetworkConfig: deviceInfo.networkConfig !== null,

    /** Whether both info and network are available */
    hasCompleteData: deviceInfo.systemInfo !== null && deviceInfo.networkConfig !== null,

    /** Whether at least one data source is available */
    hasAnyData: deviceInfo.hasData,

    /** Whether some data is available but not all (T093) */
    hasPartialData: deviceInfo.hasPartialData,

    /** Whether currently loading any data */
    isLoading: deviceInfo.isLoading,

    /** Whether there are any errors */
    hasErrors: deviceInfo.error !== null,
  };
}
