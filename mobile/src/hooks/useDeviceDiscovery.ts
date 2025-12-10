/**
 * useDeviceDiscovery Hook
 *
 * Combines mDNS discovery and fallback IP detection.
 * Manages device list state, discovery lifecycle, and auto-refresh.
 *
 * Contract: See mobile/tests/unit/hooks/useDeviceDiscovery.test.ts
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { Device } from '@/types/device';
import { getMDNSDiscoveryService } from '@/services/discovery/mdns';
import { getFallbackIPService } from '@/services/discovery/fallback';

export interface UseDeviceDiscoveryResult {
  devices: Device[];
  isScanning: boolean;
  error: Error | null;
  startDiscovery: () => void;
  stopDiscovery: () => void;
  refreshDevices: () => void;
}

/**
 * Device discovery hook
 * Combines mDNS and fallback IP detection
 */
export function useDeviceDiscovery(): UseDeviceDiscoveryResult {
  const [devices, setDevices] = useState<Device[]>([]);
  const [isScanning, setIsScanning] = useState(false);
  const [error, setError] = useState<Error | null>(null);

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
        console.error('mDNS error:', err);
        setError(err);
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
      const discoveryError = err instanceof Error ? err : new Error('Discovery failed');
      setError(discoveryError);
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
    stopDiscovery();
    setTimeout(() => {
      startDiscovery();
    }, 100);
  }, [stopDiscovery, startDiscovery]);

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
    startDiscovery,
    stopDiscovery,
    refreshDevices,
  };
}
