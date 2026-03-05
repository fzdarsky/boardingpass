/**
 * useDeviceDiscovery Hook
 *
 * Combines mDNS discovery and fallback IP detection.
 * Manages device list state, discovery lifecycle, and auto-refresh.
 *
 * Contract: See mobile/tests/unit/hooks/useDeviceDiscovery.test.ts
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { Platform } from 'react-native';
import * as ExpoDevice from 'expo-device';
import { Device } from '@/types/device';
import { getMDNSDiscoveryService } from '@/services/discovery/mdns';
import { getFallbackIPService } from '@/services/discovery/fallback';
import { toAppError, isNetworkError, AppError } from '@/utils/error-messages';

/** Reason why mDNS auto-discovery is unavailable */
export type MDNSUnavailableReason = 'simulator' | 'entitlement' | null;

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
}

/**
 * Device discovery hook
 * Combines mDNS and fallback IP detection
 */
export function useDeviceDiscovery(): UseDeviceDiscoveryResult {
  const [devices, setDevices] = useState<Device[]>([]);
  const [isScanning, setIsScanning] = useState(false);
  const [error, setError] = useState<AppError | null>(null);
  const [errorCount, setErrorCount] = useState(0);
  const [mdnsUnavailableReason, setMdnsUnavailableReason] = useState<MDNSUnavailableReason>(null);

  const mdnsService = useRef(getMDNSDiscoveryService());
  const fallbackService = useRef(getFallbackIPService());
  const cleanupFunctions = useRef<(() => void)[]>([]);
  const discoveryTimeout = useRef<NodeJS.Timeout | null>(null);

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
   * Check fallback IP
   */
  const checkFallback = useCallback(async () => {
    try {
      const fallbackDevice = await fallbackService.current.check();

      if (fallbackDevice) {
        // Only add if no mDNS device found at same IP
        setDevices(prev => {
          const mdnsDeviceExists = prev.some(
            d => d.host === fallbackDevice.host && d.discoveryMethod === 'mdns'
          );

          if (mdnsDeviceExists) {
            // mDNS takes precedence
            return prev;
          }

          addOrUpdateDevice(fallbackDevice);
          return prev;
        });
      }
    } catch (err) {
      console.warn('Fallback check failed:', err);
      // Don't set error state for fallback failures
    }
  }, [addOrUpdateDevice]);

  /**
   * Start device discovery
   */
  const startDiscovery = useCallback(() => {
    if (isScanning) {
      console.warn('Discovery already in progress');
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
        // Clear the discovery timeout since we're handling the error now
        if (discoveryTimeout.current) {
          clearTimeout(discoveryTimeout.current);
          discoveryTimeout.current = null;
        }

        // On iOS, mDNS may be unavailable on simulator or without multicast entitlement
        if (Platform.OS === 'ios') {
          // Determine reason: simulator vs missing entitlement (paid developer account)
          const reason: MDNSUnavailableReason = ExpoDevice.isDevice ? 'entitlement' : 'simulator';
          setMdnsUnavailableReason(reason);
          setError(null);
          setIsScanning(false);
          // Check fallback IP since mDNS failed
          checkFallback();
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
        setIsScanning(false);
      });

      cleanupFunctions.current.push(cleanupFound, cleanupResolved, cleanupRemoved, cleanupError);

      // Start mDNS scan
      mdnsService.current.start();

      // Check fallback IP
      checkFallback();

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
        setIsScanning(false);
        // Still check fallback IP
        checkFallback();
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
  }, [isScanning, addOrUpdateDevice, removeDevice, checkFallback]);

  /**
   * Stop device discovery
   */
  const stopDiscovery = useCallback(() => {
    if (!isScanning) {
      return;
    }

    // Stop mDNS scan
    mdnsService.current.stop();

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
  }, [isScanning]);

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
   * No HTTPS probe is performed because React Native's native TLS stack
   * rejects self-signed certificates at the handshake level, making probes
   * fail against BoardingPass devices. Instead, the device is added directly
   * and reachability is verified when the user initiates authentication.
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
        status: 'online',
        lastSeen: new Date(),
      };

      addOrUpdateDevice(manualDevice);
      return manualDevice;
    },
    [addOrUpdateDevice]
  );

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
  };
}
