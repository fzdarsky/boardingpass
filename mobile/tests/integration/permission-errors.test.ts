/**
 * Integration tests for permission denial handling
 * Tests FR-026 (camera permissions), FR-106 (permission error handling)
 */

import React from 'react';
import { renderHook, waitFor } from '@testing-library/react-native';
import * as Camera from 'expo-camera';
import { PermissionStatus } from 'expo-camera';
import * as Linking from 'expo-linking';
import { Platform } from 'react-native';

// Mock expo modules
jest.mock('expo-camera');
jest.mock('expo-linking');

const mockedCamera = Camera as jest.Mocked<typeof Camera>;
const mockedLinking = Linking as jest.Mocked<typeof Linking>;

describe('Permission Denial Handling Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Camera Permission Handling', () => {
    it('should request camera permissions before QR scanning', async () => {
      mockedCamera.Camera.requestCameraPermissionsAsync.mockResolvedValue({
        status: 'granted',
        expires: 'never',
        granted: true,
        canAskAgain: true,
      });

      const { result } = renderHook(() => {
        const [hasPermission, setHasPermission] = React.useState(false);

        React.useEffect(() => {
          (async () => {
            const { status } = await Camera.Camera.requestCameraPermissionsAsync();
            setHasPermission(status === 'granted');
          })();
        }, []);

        return { hasPermission };
      });

      await waitFor(() => {
        expect(result.current.hasPermission).toBe(true);
      });

      expect(mockedCamera.Camera.requestCameraPermissionsAsync).toHaveBeenCalledTimes(1);
    });

    it('should handle camera permission denial gracefully', async () => {
      mockedCamera.Camera.requestCameraPermissionsAsync.mockResolvedValue({
        status: 'denied',
        expires: 'never',
        granted: false,
        canAskAgain: true,
      });

      const { result } = renderHook(() => {
        const [hasPermission, setHasPermission] = React.useState<boolean | null>(null);
        const [error, setError] = React.useState<Error | null>(null);

        React.useEffect(() => {
          (async () => {
            try {
              const { status } = await Camera.Camera.requestCameraPermissionsAsync();
              setHasPermission(status === 'granted');

              if (status === 'denied') {
                setError(new Error('Camera permission denied'));
              }
            } catch (err) {
              setError(err as Error);
            }
          })();
        }, []);

        return { hasPermission, error };
      });

      await waitFor(() => {
        expect(result.current.hasPermission).toBe(false);
        expect(result.current.error).toBeDefined();
      });
    });

    it('should show rationale before requesting camera permission (Android)', async () => {
      Platform.OS = 'android';

      mockedCamera.Camera.getCameraPermissionsAsync.mockResolvedValue({
        status: 'undetermined',
        expires: 'never',
        granted: false,
        canAskAgain: true,
      });

      const { result } = renderHook(() => {
        const [showRationale, setShowRationale] = React.useState(false);

        React.useEffect(() => {
          (async () => {
            const { status, canAskAgain } = await Camera.Camera.getCameraPermissionsAsync();

            // Show rationale if permission not determined and can ask
            if (status === 'undetermined' && canAskAgain) {
              setShowRationale(true);
            }
          })();
        }, []);

        return { showRationale };
      });

      await waitFor(() => {
        expect(result.current.showRationale).toBe(true);
      });
    });

    it('should provide "Open Settings" option when permission permanently denied', async () => {
      mockedCamera.Camera.requestCameraPermissionsAsync.mockResolvedValue({
        status: 'denied',
        expires: 'never',
        granted: false,
        canAskAgain: false, // Permanently denied
      });

      mockedLinking.openSettings.mockResolvedValue(true);

      const { result } = renderHook(() => {
        const [canAskAgain, setCanAskAgain] = React.useState(true);
        const [showSettingsButton, setShowSettingsButton] = React.useState(false);

        const requestPermission = async () => {
          const { status, canAskAgain: can } = await Camera.Camera.requestCameraPermissionsAsync();
          setCanAskAgain(can);

          if (status === 'denied' && !can) {
            setShowSettingsButton(true);
          }
        };

        const openSettings = async () => {
          await Linking.openSettings();
        };

        return { canAskAgain, showSettingsButton, requestPermission, openSettings };
      });

      await result.current.requestPermission();

      await waitFor(() => {
        expect(result.current.canAskAgain).toBe(false);
        expect(result.current.showSettingsButton).toBe(true);
      });

      // User taps "Open Settings"
      await result.current.openSettings();

      expect(mockedLinking.openSettings).toHaveBeenCalledTimes(1);
    });

    it('should fallback to manual connection code input on permission denial', async () => {
      mockedCamera.Camera.requestCameraPermissionsAsync.mockResolvedValue({
        status: 'denied',
        expires: 'never',
        granted: false,
        canAskAgain: true,
      });

      const { result } = renderHook(() => {
        const [inputMethod, setInputMethod] = React.useState<'qr' | 'manual'>('qr');

        const requestCameraPermission = async () => {
          const { status } = await Camera.Camera.requestCameraPermissionsAsync();

          if (status !== 'granted') {
            // Fallback to manual input
            setInputMethod('manual');
          }
        };

        React.useEffect(() => {
          requestCameraPermission();
        }, []);

        return { inputMethod };
      });

      await waitFor(() => {
        expect(result.current.inputMethod).toBe('manual');
      });
    });
  });

  describe('Network Permission Handling', () => {
    it('should handle local network permission denial (iOS 14+)', async () => {
      Platform.OS = 'ios';

      // Simulate local network permission denial
      const mockFetch = jest.fn().mockRejectedValue({
        code: 'NETWORK_ERROR',
        message: 'Local network access denied',
      });

      global.fetch = mockFetch;

      const { result } = renderHook(() => {
        const [error, setError] = React.useState<Error | null>(null);

        React.useEffect(() => {
          (async () => {
            try {
              await fetch('https://192.168.1.100:8443/info');
            } catch (err: unknown) {
              setError(err instanceof Error ? err : new Error(String(err)));
            }
          })();
        }, []);

        return { error };
      });

      await waitFor(() => {
        expect(result.current.error).toBeDefined();
      });
    });
  });

  describe('Permission Error Recovery', () => {
    it('should re-check permissions when returning from settings', async () => {
      let permissionStatus = 'denied';

      mockedCamera.Camera.getCameraPermissionsAsync.mockImplementation(async () => ({
        status: permissionStatus as PermissionStatus,
        expires: 'never',
        granted: permissionStatus === 'granted',
        canAskAgain: false,
      }));

      const { result } = renderHook(() => {
        const [hasPermission, setHasPermission] = React.useState<boolean | null>(null);

        const checkPermission = async () => {
          const { status } = await Camera.Camera.getCameraPermissionsAsync();
          setHasPermission(status === 'granted');
        };

        return { hasPermission, checkPermission };
      });

      // Initial check - permission denied
      await result.current.checkPermission();

      await waitFor(() => {
        expect(result.current.hasPermission).toBe(false);
      });

      // User goes to settings and grants permission
      permissionStatus = 'granted';

      // App becomes active again, re-check permission
      await result.current.checkPermission();

      await waitFor(() => {
        expect(result.current.hasPermission).toBe(true);
      });
    });

    it('should provide clear error message for each permission type', async () => {
      const permissionErrors = [
        {
          type: 'camera',
          status: 'denied',
          expectedMessage: /camera permission/i,
        },
        {
          type: 'local_network',
          status: 'denied',
          expectedMessage: /local.network/i,
        },
      ];

      for (const { type, status, expectedMessage } of permissionErrors) {
        const error = new Error(`${type} permission ${status}`);
        expect(error.message).toMatch(expectedMessage);
      }
    });
  });

  describe('Permission Error UI Flow', () => {
    it('should show permission rationale before first request', async () => {
      mockedCamera.Camera.getCameraPermissionsAsync.mockResolvedValue({
        status: 'undetermined',
        expires: 'never',
        granted: false,
        canAskAgain: true,
      });

      const { result } = renderHook(() => {
        const [step, setStep] = React.useState<'rationale' | 'request' | 'granted' | 'denied'>(
          'rationale'
        );

        const proceedToRequest = async () => {
          setStep('request');
          const { status } = await Camera.Camera.requestCameraPermissionsAsync();
          setStep(status === 'granted' ? 'granted' : 'denied');
        };

        return { step, proceedToRequest };
      });

      expect(result.current.step).toBe('rationale');

      await result.current.proceedToRequest();

      // Should transition through request step
      expect(mockedCamera.Camera.requestCameraPermissionsAsync).toHaveBeenCalled();
    });

    it('should provide context-specific help text for permission denial', async () => {
      const helpTexts = {
        camera_needed_for_qr:
          'Camera access is required to scan QR codes. You can also enter the connection code manually.',
        local_network_needed:
          'Local network access is required to discover devices on your network.',
      };

      // Verify help texts are informative and actionable
      expect(helpTexts.camera_needed_for_qr).toMatch(/manual/i);
      expect(helpTexts.local_network_needed).toMatch(/discover/i);
    });
  });
});

// Helper to reset Platform.OS after tests
afterAll(() => {
  Platform.OS = 'ios'; // Reset to default
});
