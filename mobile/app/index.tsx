/**
 * Device Discovery Screen
 *
 * Main screen for discovering BoardingPass devices on local network.
 * Features:
 * - Automatic mDNS/Bonjour discovery (T041, T052)
 * - Fallback IP detection (T042)
 * - Pull-to-refresh support (T048)
 * - Auto-refresh on device appear/disappear (T052 - FR-005)
 * - Device status indicators (T047 - online, offline, authenticating, authenticated)
 * - Duplicate device name handling (T049 - FR-006)
 * - Empty state and scanning state UI (T050, T051)
 * - Logging without sensitive data (T053 - FR-029)
 */

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { StyleSheet, View } from 'react-native';
import {
  Button,
  Snackbar,
  Portal,
  Dialog,
  TextInput,
  HelperText,
  useTheme,
} from 'react-native-paper';
import { useFocusEffect, useRouter } from 'expo-router';
import { useDeviceDiscovery } from '@/hooks/useDeviceDiscovery';
import { DeviceList } from '@/components/DeviceList';
import { Device } from '@/types/device';
import { parseAndValidateHostPort } from '@/utils/validation';
import { HapticFeedback } from '@/utils/haptics';
import { sessionManager } from '@/services/auth/session';

export default function DeviceDiscoveryScreen() {
  const router = useRouter();
  const theme = useTheme();
  const { devices, isScanning, error, startDiscovery, refreshDevices, addManualDevice } =
    useDeviceDiscovery();

  const [snackbarVisible, setSnackbarVisible] = useState(false);
  const [snackbarMessage, setSnackbarMessage] = useState('');

  // Manual device entry dialog state
  const [addDialogVisible, setAddDialogVisible] = useState(false);
  const [addressInput, setAddressInput] = useState('');
  const [addressError, setAddressError] = useState<string | undefined>(undefined);

  // Track which devices have valid auth sessions (for UI display)
  const [authenticatedDevices, setAuthenticatedDevices] = useState<Set<string>>(new Set());
  const devicesRef = useRef(devices);
  devicesRef.current = devices;

  // Auto-start discovery on mount
  useEffect(() => {
    startDiscovery();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Refresh auth state for all devices when screen gains focus.
  // Uses a ref for devices to avoid re-triggering on every render
  // (useDeviceDiscovery returns a new array reference each time).
  useFocusEffect(
    useCallback(() => {
      const checkAuthStates = async () => {
        const authenticated = new Set<string>();
        for (const device of devicesRef.current) {
          const validation = await sessionManager.isSessionValid(device.id);
          if (validation.isValid) {
            authenticated.add(device.id);
          }
        }
        setAuthenticatedDevices(authenticated);
      };
      checkAuthStates();
    }, [])
  );

  // Show error snackbar
  useEffect(() => {
    if (error) {
      setSnackbarMessage(error.message);
      setSnackbarVisible(true);
    }
  }, [error]);

  /**
   * Handle device press - navigate to details if authenticated, otherwise to auth screen
   */
  const handleDevicePress = async (device: Device) => {
    if (device.status === 'online' || device.status === 'authenticated') {
      const validation = await sessionManager.isSessionValid(device.id);

      if (validation.isValid) {
        router.push({
          pathname: '/device/[id]',
          params: {
            id: device.id,
            host: device.host,
            port: device.port.toString(),
          },
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } as any);
      } else {
        router.push({
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          pathname: '/device/authenticate' as any,
          params: {
            deviceId: device.id,
            deviceName: device.name,
            host: device.host,
            port: device.port.toString(),
          },
        });
      }
    }
  };

  /**
   * Handle manual scan button press
   */
  const handleStartScan = () => {
    startDiscovery();
  };

  /**
   * Manual device entry handlers
   */
  const handleOpenAddDialog = () => {
    setAddressInput('');
    setAddressError(undefined);
    setAddDialogVisible(true);
  };

  const handleCloseAddDialog = () => {
    setAddDialogVisible(false);
    setAddressInput('');
    setAddressError(undefined);
  };

  const handleAddressChange = (text: string) => {
    setAddressInput(text);
    if (text.trim().length === 0) {
      setAddressError(undefined);
      return;
    }
    const result = parseAndValidateHostPort(text);
    setAddressError(result.valid ? undefined : result.error);
  };

  const handleAddDevice = () => {
    const result = parseAndValidateHostPort(addressInput);
    if (!result.valid || !result.host || !result.port) {
      setAddressError(result.error);
      return;
    }

    addManualDevice(result.host, result.port);
    HapticFeedback.success();
    handleCloseAddDialog();
  };

  const isAddButtonEnabled = addressInput.trim().length > 0 && !addressError;

  return (
    <View style={[styles.container, { backgroundColor: theme.colors.surface }]}>
      {/* Device List with integrated features:
          - T044: DeviceCard component with status indicators
          - T045: DeviceList component
          - T047: Status indicators (online, offline, etc.)
          - T048: Pull-to-refresh
          - T049: Duplicate device name handling (shows IP)
          - T050: Empty state UI
          - T051: Scanning state UI
      */}
      <DeviceList
        devices={devices}
        isScanning={isScanning}
        onRefresh={refreshDevices}
        onDevicePress={handleDevicePress}
        onStartScan={handleStartScan}
        onAddDevice={handleOpenAddDialog}
        authenticatedDeviceIds={authenticatedDevices}
      />

      {/* Action Bar (when devices are shown) */}
      {devices.length > 0 && !isScanning && (
        <View
          style={[
            styles.actionBar,
            { backgroundColor: theme.colors.surface, borderTopColor: theme.colors.outlineVariant },
          ]}
        >
          <View style={styles.actionBarButtons}>
            <Button
              mode="contained"
              onPress={refreshDevices}
              icon="refresh"
              style={styles.actionButton}
            >
              Refresh
            </Button>
            <Button
              mode="outlined"
              onPress={handleOpenAddDialog}
              icon="plus"
              style={styles.actionButton}
            >
              Add Device
            </Button>
          </View>
        </View>
      )}

      {/* Manual Device Entry Dialog */}
      <Portal>
        <Dialog visible={addDialogVisible} onDismiss={handleCloseAddDialog} style={styles.dialog}>
          <Dialog.Title>Add Device</Dialog.Title>
          <Dialog.Content>
            <TextInput
              label="Device address"
              placeholder="192.168.1.100:8443"
              value={addressInput}
              onChangeText={handleAddressChange}
              onSubmitEditing={isAddButtonEnabled ? handleAddDevice : undefined}
              mode="outlined"
              autoCapitalize="none"
              autoCorrect={false}
              keyboardType="url"
              error={!!addressError}
              accessibilityLabel="Device IP address"
            />
            <HelperText type={addressError ? 'error' : 'info'} visible={true}>
              {addressError || 'Enter IP address with optional port (default: 8443)'}
            </HelperText>
          </Dialog.Content>
          <Dialog.Actions>
            <Button onPress={handleCloseAddDialog}>Cancel</Button>
            <Button
              mode="contained"
              onPress={handleAddDevice}
              disabled={!isAddButtonEnabled}
              contentStyle={styles.addDialogButton}
            >
              Add
            </Button>
          </Dialog.Actions>
        </Dialog>
      </Portal>

      {/* Error Snackbar */}
      <Snackbar
        visible={snackbarVisible}
        onDismiss={() => setSnackbarVisible(false)}
        duration={4000}
        action={{
          label: 'Retry',
          onPress: () => {
            setSnackbarVisible(false);
            startDiscovery();
          },
        }}
      >
        {snackbarMessage}
      </Snackbar>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  actionBar: {
    padding: 16,
    borderTopWidth: 1,
  },
  actionBarButtons: {
    flexDirection: 'row',
    gap: 12,
  },
  actionButton: {
    flex: 1,
  },
  dialog: {
    borderRadius: 4,
  },
  addDialogButton: {
    paddingHorizontal: 24,
  },
});
