/**
 * Secure Storage Hook
 *
 * Wrapper around expo-secure-store for encrypted storage.
 * Uses iOS Keychain and Android Keystore for secure data persistence.
 */

import { useCallback } from 'react';
import * as SecureStore from 'expo-secure-store';

export interface SecureStorageOptions {
  keychainService?: string; // iOS only
  keychainAccessible?: SecureStore.KeychainAccessibilityConstant;
}

/**
 * useSecureStorage
 *
 * Hook for secure storage operations with iOS Keychain / Android Keystore
 */
export function useSecureStorage() {
  /**
   * Save item to secure storage
   */
  const saveItem = useCallback(
    async (key: string, value: string, options?: SecureStorageOptions): Promise<void> => {
      try {
        await SecureStore.setItemAsync(key, value, options);
      } catch (error) {
        throw new Error(`Failed to save ${key} to secure storage: ${error}`);
      }
    },
    []
  );

  /**
   * Get item from secure storage
   */
  const getItem = useCallback(
    async (key: string, options?: SecureStorageOptions): Promise<string | null> => {
      try {
        return await SecureStore.getItemAsync(key, options);
      } catch (error) {
        throw new Error(`Failed to retrieve ${key} from secure storage: ${error}`);
      }
    },
    []
  );

  /**
   * Delete item from secure storage
   */
  const deleteItem = useCallback(
    async (key: string, options?: SecureStorageOptions): Promise<void> => {
      try {
        await SecureStore.deleteItemAsync(key, options);
      } catch (error) {
        throw new Error(`Failed to delete ${key} from secure storage: ${error}`);
      }
    },
    []
  );

  /**
   * Check if item exists in secure storage
   */
  const hasItem = useCallback(async (key: string): Promise<boolean> => {
    try {
      const value = await SecureStore.getItemAsync(key);
      return value !== null;
    } catch {
      return false;
    }
  }, []);

  /**
   * Save JSON object to secure storage
   */
  const saveJSON = useCallback(
    async <T>(key: string, data: T, options?: SecureStorageOptions): Promise<void> => {
      const json = JSON.stringify(data);
      await saveItem(key, json, options);
    },
    [saveItem]
  );

  /**
   * Get JSON object from secure storage
   */
  const getJSON = useCallback(
    async <T>(key: string, options?: SecureStorageOptions): Promise<T | null> => {
      const json = await getItem(key, options);
      if (json === null) return null;
      try {
        return JSON.parse(json) as T;
      } catch (error) {
        throw new Error(`Failed to parse JSON for ${key}: ${error}`);
      }
    },
    [getItem]
  );

  return {
    saveItem,
    getItem,
    deleteItem,
    hasItem,
    saveJSON,
    getJSON,
  };
}

/**
 * Storage key constants for consistent naming
 */
export const STORAGE_KEYS = {
  // Session tokens by device ID
  sessionToken: (deviceId: string) => `session_token_${deviceId}`,

  // Certificate pins by device ID
  certificatePin: (deviceId: string) => `cert_pin_${deviceId}`,

  // Last known device list (optional - for quick startup)
  deviceList: 'device_list',

  // User preferences
  preferences: 'user_preferences',
} as const;
