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

import React, { useEffect } from 'react';
import { StyleSheet, View } from 'react-native';
import { Button, Snackbar } from 'react-native-paper';
import { useRouter } from 'expo-router';
import { useDeviceDiscovery } from '@/hooks/useDeviceDiscovery';
import { DeviceList } from '@/components/DeviceList';
import { Device } from '@/types/device';

export default function DeviceDiscoveryScreen() {
  const router = useRouter();
  const { devices, isScanning, error, startDiscovery, refreshDevices } = useDeviceDiscovery();

  const [snackbarVisible, setSnackbarVisible] = React.useState(false);
  const [snackbarMessage, setSnackbarMessage] = React.useState('');

  // Auto-start discovery on mount
  useEffect(() => {
    startDiscovery();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Show error snackbar
  useEffect(() => {
    if (error) {
      setSnackbarMessage(error.message);
      setSnackbarVisible(true);
    }
  }, [error]);

  /**
   * Handle device press - navigate to authentication screen
   */
  const handleDevicePress = (device: Device) => {
    if (device.status === 'online') {
      router.push({
        pathname: '/device/authenticate',
        params: {
          deviceId: device.id,
          deviceName: device.name,
          deviceHost: device.host,
          devicePort: device.port.toString(),
        },
      });
    }
  };

  /**
   * Handle manual scan button press
   */
  const handleStartScan = () => {
    startDiscovery();
  };

  return (
    <View style={styles.container}>
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
      />

      {/* Manual Scan Button (when devices are shown) */}
      {devices.length > 0 && !isScanning && (
        <View style={styles.actionBar}>
          <Button
            mode="contained"
            onPress={refreshDevices}
            icon="refresh"
            style={styles.refreshButton}
          >
            Refresh
          </Button>
        </View>
      )}

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
    backgroundColor: '#f5f5f5',
  },
  actionBar: {
    padding: 16,
    backgroundColor: '#fff',
    borderTopWidth: 1,
    borderTopColor: '#e0e0e0',
  },
  refreshButton: {
    paddingVertical: 4,
  },
});
