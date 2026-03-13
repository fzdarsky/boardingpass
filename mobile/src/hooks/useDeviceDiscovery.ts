/**
 * useDeviceDiscovery Hook
 *
 * Combines mDNS discovery and subnet scanning.
 * Manages device list state, discovery lifecycle, and auto-refresh.
 * Periodically probes devices to track reachability and detect state changes.
 *
 * Contract: See mobile/tests/unit/hooks/useDeviceDiscovery.test.ts
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { AppState, Platform } from 'react-native';
import * as ExpoDevice from 'expo-device';
import { Device, DeviceStatus } from '@/types/device';
import { getMDNSDiscoveryService } from '@/services/discovery/mdns';
import { getSubnetScannerService } from '@/services/discovery/scan';
import { probeDevice, probeAllDevices, ProbeResult } from '@/services/reachability/probe';
import { sessionManager } from '@/services/auth/session';
import * as enrollmentStore from '@/services/enrollment/store';
import { toAppError, isNetworkError, AppError } from '@/utils/error-messages';

/** Reason why mDNS auto-discovery is unavailable */
export type MDNSUnavailableReason = 'simulator' | 'entitlement' | null;

/** Interval between periodic reachability probes (ms) */
const PROBE_INTERVAL = 30000;

export interface UseDeviceDiscoveryResult {
  devices: Device[];
  isScanning: boolean;
  error: AppError | null;
  errorCount: number;
  retry: (() => void) | undefined;
  mdnsUnavailableReason: MDNSUnavailableReason;
  startDiscovery: () => void;
  stopDiscovery: () => void;
  refreshDevices: () => void;
  addManualDevice: (ip: string, port: number) => Device;
  deleteDevice: (deviceId: string) => void;
  markDeviceEnrolled: (deviceId: string) => void;
  refreshAuthStates: () => Promise<void>;
}

/**
 * Device discovery hook
 * Combines mDNS and subnet scanning
 */
export function useDeviceDiscovery(): UseDeviceDiscoveryResult {
  const [devices, setDevices] = useState<Device[]>([]);
  const [isScanning, setIsScanning] = useState(false);
  const [error, setError] = useState<AppError | null>(null);
  const [errorCount, setErrorCount] = useState(0);
  const [mdnsUnavailableReason, setMdnsUnavailableReason] = useState<MDNSUnavailableReason>(null);

  const mdnsService = useRef(getMDNSDiscoveryService());
  const cleanupFunctions = useRef<(() => void)[]>([]);
  const discoveryTimeout = useRef<NodeJS.Timeout | null>(null);
  const probeInterval = useRef<NodeJS.Timeout | null>(null);
  const isScanningRef = useRef(isScanning);
  isScanningRef.current = isScanning;
  const devicesRef = useRef(devices);
  devicesRef.current = devices;

  /**
   * Add or update device in list
   */
  const addOrUpdateDevice = useCallback((device: Device) => {
    setDevices(prev => {
      const existingIndex = prev.findIndex(d => d.id === device.id);

      if (existingIndex >= 0) {
        // Update existing device (refresh lastSeen)
        const updated = [...prev];
        updated[existingIndex] = {
          ...updated[existingIndex],
          ...device,
          lastSeen: new Date(),
        };
        return updated;
      } else {
        // Add new device
        // eslint-disable-next-line no-console
        console.log('Device discovered:', {
          deviceId: device.id,
          discoveryMethod: device.discoveryMethod,
          timestamp: new Date().toISOString(),
        });
        return [...prev, device];
      }
    });
  }, []);

  /**
   * Remove device from list (mark as offline)
   * Used by mDNS when a device disappears from the network
   */
  const removeDevice = useCallback((deviceId: string) => {
    setDevices(prev => {
      const device = prev.find(d => d.id === deviceId);
      if (!device) return prev;

      // eslint-disable-next-line no-console
      console.log('Device removed:', {
        deviceId,
        timestamp: new Date().toISOString(),
      });

      return prev.map(d => (d.id === deviceId ? { ...d, status: 'offline' as const } : d));
    });
  }, []);

  /**
   * Delete device from list entirely (user-initiated)
   */
  const deleteDevice = useCallback((deviceId: string) => {
    setDevices(prev => {
      const device = prev.find(d => d.id === deviceId);
      if (!device) return prev;

      // eslint-disable-next-line no-console
      console.log('Device deleted by user:', {
        deviceId,
        discoveryMethod: device.discoveryMethod,
        timestamp: new Date().toISOString(),
      });

      return prev.filter(d => d.id !== deviceId);
    });
  }, []);

  /**
   * Start subnet scan as fallback when mDNS is unavailable.
   */
  const startSubnetScan = useCallback(() => {
    const scanner = getSubnetScannerService();
    const cleanup = scanner.onDeviceFound(addOrUpdateDevice);
    cleanupFunctions.current.push(cleanup);
    scanner.start().finally(() => {
      // Only clear scanning state when running as standalone fallback (no mDNS timeout active)
      if (!discoveryTimeout.current) {
        setIsScanning(false);
      }
    });
  }, [addOrUpdateDevice]);

  /**
   * Start device discovery
   */
  const startDiscovery = useCallback(() => {
    if (isScanningRef.current) {
      // eslint-disable-next-line no-console
      console.log('Discovery already in progress');
      return;
    }

    setIsScanning(true);
    setError(null);

    try {
      // Setup mDNS listeners
      const cleanupFound = mdnsService.current.onDeviceFound(addOrUpdateDevice);
      const cleanupResolved = mdnsService.current.onDeviceResolved(addOrUpdateDevice);
      const cleanupRemoved = mdnsService.current.onDeviceRemoved(removeDevice);
      const cleanupError = mdnsService.current.onError(err => {
        // On iOS, mDNS may be unavailable on simulator or without multicast entitlement.
        // The subnet scan is already running (started below), so just record the reason.
        if (Platform.OS === 'ios') {
          const reason: MDNSUnavailableReason = ExpoDevice.isDevice ? 'entitlement' : 'simulator';
          setMdnsUnavailableReason(reason);
          setError(null);
          return;
        }

        // Convert to AppError with context for other platforms
        const appError = toAppError(err, isNetworkError(err) ? 'network' : 'unknown');
        appError.context = {
          ...appError.context,
          service: 'mdns',
          operation: 'discovery',
        };

        setError(appError);
        setErrorCount(prev => prev + 1);
      });

      cleanupFunctions.current.push(cleanupFound, cleanupResolved, cleanupRemoved, cleanupError);

      // Start mDNS scan
      mdnsService.current.start();

      // Also start subnet scan (finds devices on USB tethering, Ethernet, etc.)
      startSubnetScan();

      // Set timeout for discovery (10 seconds)
      discoveryTimeout.current = setTimeout(() => {
        setIsScanning(false);
        // eslint-disable-next-line no-console
        console.log('Discovery timeout reached');
      }, 10000);
    } catch (err) {
      // On iOS, mDNS may be unavailable on simulator or without multicast entitlement
      if (Platform.OS === 'ios') {
        const reason: MDNSUnavailableReason = ExpoDevice.isDevice ? 'entitlement' : 'simulator';
        setMdnsUnavailableReason(reason);
        setError(null);
        // Still scan subnet (scan completion will clear isScanning)
        startSubnetScan();
        return;
      }

      const appError = toAppError(err, isNetworkError(err) ? 'network' : 'unknown');
      appError.context = {
        ...appError.context,
        operation: 'discovery',
      };

      setError(appError);
      setErrorCount(prev => prev + 1);
      setIsScanning(false);
    }
  }, [addOrUpdateDevice, removeDevice, startSubnetScan]);

  /**
   * Stop device discovery
   */
  const stopDiscovery = useCallback(() => {
    // Stop mDNS scan
    mdnsService.current.stop();

    // Stop subnet scanner
    getSubnetScannerService().stop();

    // Clear timeout
    if (discoveryTimeout.current) {
      clearTimeout(discoveryTimeout.current);
      discoveryTimeout.current = null;
    }

    // Cleanup listeners
    cleanupFunctions.current.forEach(cleanup => cleanup());
    cleanupFunctions.current = [];

    setIsScanning(false);
    // eslint-disable-next-line no-console
    console.log('Discovery stopped');
  }, []);

  /**
   * Refresh devices (re-scan)
   */
  const refreshDevices = useCallback(() => {
    setError(null);
    stopDiscovery();
    setTimeout(() => {
      startDiscovery();
    }, 100);
  }, [stopDiscovery, startDiscovery]);

  /**
   * Retry after error
   */
  const retry = useCallback(() => {
    setError(null);
    refreshDevices();
  }, [refreshDevices]);

  /**
   * Add a device manually by IP address and port.
   *
   * The device is added with 'offline' status and an immediate reachability
   * probe is triggered. The probe result updates the status to 'online',
   * 'unavailable', or keeps it 'offline'.
   */
  const addManualDevice = useCallback(
    (ip: string, port: number): Device => {
      const manualDevice: Device = {
        id: `manual:${ip}:${port}`,
        name: `Device (${ip})`,
        host: ip,
        port,
        addresses: [ip],
        discoveryMethod: 'manual',
        status: 'offline',
        lastSeen: new Date(),
      };

      addOrUpdateDevice(manualDevice);

      // Immediately probe the device
      probeDevice(ip, port).then(result => {
        setDevices(prev =>
          prev.map(d =>
            d.id === manualDevice.id ? { ...d, status: result, lastSeen: new Date() } : d
          )
        );
      });

      return manualDevice;
    },
    [addOrUpdateDevice]
  );

  /**
   * Mark a device as enrolled (provisioning completed).
   * The service has terminated; status shows 'enrolled' with timestamp.
   * Also writes to the enrollment store so other screens can signal enrollment.
   */
  const markDeviceEnrolled = useCallback((deviceId: string) => {
    enrollmentStore.markEnrolled(deviceId);
    const enrolledAt = enrollmentStore.getEnrolledAt(deviceId) ?? new Date();
    setDevices(prev =>
      prev.map(d => (d.id === deviceId ? { ...d, status: 'enrolled' as const, enrolledAt } : d))
    );
  }, []);

  /**
   * Apply probe results to devices, respecting the enrolled state.
   * Enrolled devices only transition to 'online' (factory reset detection);
   * they stay 'enrolled' for refused/timeout results.
   */
  const applyProbeResults = useCallback((results: Map<string, ProbeResult>) => {
    setDevices(prev =>
      prev.map(d => {
        const result = results.get(d.id);
        if (!result) return d;

        if (d.status === 'enrolled') {
          // Only transition out of enrolled if service comes back online (factory reset)
          if (result === 'online') {
            enrollmentStore.clearEnrolled(d.id);
            return { ...d, status: 'online' as const, enrolledAt: undefined, lastSeen: new Date() };
          }
          return d;
        }

        return { ...d, status: result as DeviceStatus, lastSeen: new Date() };
      })
    );
  }, []);

  /**
   * Run probes for all devices that should be probed.
   * Skips devices in 'authenticating' or 'authenticated' states.
   */
  const runPeriodicProbes = useCallback(async () => {
    const probeable = devicesRef.current.filter(
      d => d.status !== 'authenticating' && d.status !== 'authenticated'
    );
    if (probeable.length === 0) return;

    const results = await probeAllDevices(probeable);
    applyProbeResults(results);
  }, [applyProbeResults]);

  /**
   * Check session validity and enrollment state for all devices.
   * - Enrolled devices (from store) → 'enrolled'
   * - Devices with valid sessions → 'authenticated'
   * - Others remain unchanged (periodic probes handle status)
   */
  const refreshAuthStates = useCallback(async () => {
    const enrolledIds = enrollmentStore.getAllEnrolledIds();
    const updates: { id: string; hasSession: boolean }[] = [];

    for (const device of devicesRef.current) {
      const validation = await sessionManager.isSessionValid(device.id);
      updates.push({ id: device.id, hasSession: validation.isValid });
    }

    setDevices(prev =>
      prev.map(d => {
        // Check enrollment store (handles cross-screen enrollment signaling)
        if (enrolledIds.has(d.id) && d.status !== 'enrolled') {
          return {
            ...d,
            status: 'enrolled' as const,
            enrolledAt: enrollmentStore.getEnrolledAt(d.id) ?? new Date(),
          };
        }

        const update = updates.find(u => u.id === d.id);
        if (!update) return d;

        if (update.hasSession && d.status !== 'authenticated' && d.status !== 'enrolled') {
          return { ...d, status: 'authenticated' as const };
        }

        return d;
      })
    );
  }, []);

  /**
   * Periodic reachability probes.
   * Pauses when app is backgrounded, resumes when foregrounded.
   */
  useEffect(() => {
    const startProbeInterval = () => {
      if (probeInterval.current) return;
      probeInterval.current = setInterval(runPeriodicProbes, PROBE_INTERVAL);
    };

    const stopProbeInterval = () => {
      if (probeInterval.current) {
        clearInterval(probeInterval.current);
        probeInterval.current = null;
      }
    };

    // Start immediately
    startProbeInterval();

    // Pause/resume based on AppState
    const subscription = AppState.addEventListener('change', nextState => {
      if (nextState === 'active') {
        runPeriodicProbes(); // Probe immediately on foreground
        startProbeInterval();
      } else {
        stopProbeInterval();
      }
    });

    return () => {
      stopProbeInterval();
      subscription.remove();
    };
  }, [runPeriodicProbes]);

  /**
   * Cleanup on unmount
   */
  useEffect(() => {
    return () => {
      stopDiscovery();
    };
  }, [stopDiscovery]);

  /**
   * Sort devices by lastSeen (most recent first)
   */
  const sortedDevices = [...devices].sort((a, b) => b.lastSeen.getTime() - a.lastSeen.getTime());

  return {
    devices: sortedDevices,
    isScanning,
    error,
    errorCount,
    retry: error ? retry : undefined,
    mdnsUnavailableReason,
    startDiscovery,
    stopDiscovery,
    refreshDevices,
    addManualDevice,
    deleteDevice,
    markDeviceEnrolled,
    refreshAuthStates,
  };
}
