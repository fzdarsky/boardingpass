/**
 * Device Detail Screen
 *
 * Displays detailed information about an authenticated device including:
 * - System information (CPU, OS, FIPS status)
 * - Board information (manufacturer, model, serial)
 * - TPM information (if present)
 * - Network configuration (interfaces, IP addresses)
 *
 * Implements:
 * - T088: Device detail screen
 * - T089: Data formatting for readability
 * - T090: Loading states for info queries
 * - T091: Error handling for info query failures
 * - T092: Retry mechanism for transient failures
 * - T093: Partial data display (some queries succeed, others fail)
 * - T094: FIPS status indicator
 * - T095: Network interface status indicators
 */

import React, { useMemo, useState, useEffect } from 'react';
import { ScrollView, View, StyleSheet, RefreshControl } from 'react-native';
import { useLocalSearchParams, useRouter, Stack } from 'expo-router';
import { Appbar, Text, ActivityIndicator, Button, Banner, useTheme } from 'react-native-paper';
import { SystemInfo, NetworkConfig, TPMInfo, BoardInfo } from '../../src/components/DeviceInfo';
import { useDeviceInfo, useDeviceInfoAvailability } from '../../src/hooks/useDeviceInfo';
import { createAPIClient } from '../../src/services/api/client';
import { sessionManager } from '../../src/services/auth/session';
import { spacing } from '../../src/theme';
import { SkeletonDeviceDetail } from '../../src/components/Skeleton';

/**
 * Device Detail Screen Component
 *
 * Route: /device/[id]
 * Params: id (device ID), host, port
 * Token is loaded from secure storage (never passed via URL params)
 */
export default function DeviceDetailScreen() {
  const { id, host, port } = useLocalSearchParams<{
    id: string;
    host: string;
    port: string;
  }>();

  const router = useRouter();
  const theme = useTheme();

  // Load session token from secure storage (never passed via URL params)
  const [token, setToken] = useState<string | null>(null);
  const [tokenLoading, setTokenLoading] = useState(true);

  useEffect(() => {
    if (!id) return;

    sessionManager.getValidToken(id).then(t => {
      setToken(t);
      setTokenLoading(false);
    });
  }, [id]);

  // Create API client with authentication token
  const client = useMemo(() => {
    if (!host || !port || !token) {
      return null;
    }

    const apiClient = createAPIClient(host, parseInt(port, 10));
    apiClient.setAuthToken(token);
    return apiClient;
  }, [host, port, token]);

  // Fetch device information (auto-fetch on mount)
  const deviceInfo = useDeviceInfo(client);
  const availability = useDeviceInfoAvailability(deviceInfo);

  // Handle missing route parameters
  if (!id || !host || !port) {
    return (
      <View style={styles.container}>
        <Stack.Screen
          options={{
            title: 'Device Details',
            // eslint-disable-next-line react/no-unstable-nested-components
            headerLeft: () => <Appbar.BackAction onPress={() => router.back()} />,
          }}
        />
        <View style={styles.errorContainer}>
          <Text variant="headlineSmall" style={styles.errorTitle}>
            Missing Parameters
          </Text>
          <Text variant="bodyMedium" style={styles.errorMessage}>
            Device information is incomplete. Please go back and try again.
          </Text>
          <Button mode="contained" onPress={() => router.back()} style={styles.errorButton}>
            Go Back
          </Button>
        </View>
      </View>
    );
  }

  // Show loading while token is being retrieved from secure storage
  if (tokenLoading) {
    return (
      <View style={styles.container}>
        <Stack.Screen
          options={{
            title: 'Device Details',
            // eslint-disable-next-line react/no-unstable-nested-components
            headerLeft: () => <Appbar.BackAction onPress={() => router.back()} />,
          }}
        />
        <View style={styles.loadingContainer}>
          <ActivityIndicator size="large" animating={true} />
          <Text variant="bodyMedium" style={styles.loadingText}>
            Loading session...
          </Text>
        </View>
      </View>
    );
  }

  // Handle missing/expired token
  if (!token) {
    return (
      <View style={styles.container}>
        <Stack.Screen
          options={{
            title: 'Device Details',
            // eslint-disable-next-line react/no-unstable-nested-components
            headerLeft: () => <Appbar.BackAction onPress={() => router.back()} />,
          }}
        />
        <View style={styles.errorContainer}>
          <Text variant="headlineSmall" style={styles.errorTitle}>
            Session Expired
          </Text>
          <Text variant="bodyMedium" style={styles.errorMessage}>
            Your session has expired. Please authenticate again.
          </Text>
          <Button mode="contained" onPress={() => router.back()} style={styles.errorButton}>
            Go Back
          </Button>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Stack.Screen
        options={{
          title: `${host}:${port}`,
          // eslint-disable-next-line react/no-unstable-nested-components
          headerLeft: () => <Appbar.BackAction onPress={() => router.back()} />,
          // eslint-disable-next-line react/no-unstable-nested-components
          headerRight: () => (
            <Appbar.Action
              icon="refresh"
              onPress={deviceInfo.refetch}
              disabled={deviceInfo.isLoading}
            />
          ),
        }}
      />

      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          <RefreshControl
            refreshing={deviceInfo.isLoading}
            onRefresh={deviceInfo.refetch}
            colors={[theme.colors.primary]}
          />
        }
      >
        {/* Initial Loading State with Skeleton (T090, T128) */}
        {deviceInfo.isLoading && !availability.hasAnyData && <SkeletonDeviceDetail />}

        {/* Error Banner for Partial Data (T093) */}
        {availability.hasPartialData && (
          <Banner
            visible={true}
            icon="alert"
            actions={[
              {
                label: 'Retry',
                onPress: deviceInfo.refetch,
              },
              {
                label: 'Dismiss',
                onPress: deviceInfo.clearError,
              },
            ]}
          >
            Some information could not be loaded. Displaying partial data.
            {deviceInfo.error && ` Error: ${deviceInfo.error.message}`}
          </Banner>
        )}

        {/* Error State for Complete Failure (T091) */}
        {!availability.hasAnyData && !deviceInfo.isLoading && deviceInfo.error && (
          <View style={styles.errorContainer}>
            <Text variant="headlineSmall" style={styles.errorTitle}>
              Failed to Load Information
            </Text>
            <Text variant="bodyMedium" style={styles.errorMessage}>
              {deviceInfo.error.message}
            </Text>
            <Button
              mode="contained"
              onPress={deviceInfo.refetch}
              style={styles.errorButton}
              icon="refresh"
            >
              Retry
            </Button>
          </View>
        )}

        {/* System Information (if available) */}
        {deviceInfo.systemInfo && (
          <>
            <SystemInfo
              systemInfo={deviceInfo.systemInfo}
              showFIPSIndicator={true} // T094
            />

            {/* TPM Information (nested from system info) */}
            <TPMInfo tpmInfo={deviceInfo.systemInfo.tpm} />

            {/* Board Information (nested from system info) */}
            <BoardInfo
              boardInfo={deviceInfo.systemInfo.board}
              formatSerial={true} // T089
            />
          </>
        )}

        {/* Loading Indicator for System Info (if loading individually) */}
        {deviceInfo.loadingStates.info && !deviceInfo.systemInfo && (
          <View style={styles.sectionLoading}>
            <ActivityIndicator size="small" animating={true} />
            <Text variant="bodyMedium" style={styles.loadingText}>
              Loading system information...
            </Text>
          </View>
        )}

        {/* Error for System Info Only (if network succeeded) */}
        {deviceInfo.errors.info && deviceInfo.networkConfig && (
          <View style={styles.sectionError}>
            <Text variant="titleMedium" style={styles.sectionErrorTitle}>
              System Information Unavailable
            </Text>
            <Text variant="bodySmall" style={styles.sectionErrorMessage}>
              {deviceInfo.errors.info.message}
            </Text>
          </View>
        )}

        {/* Network Configuration (if available) */}
        {deviceInfo.networkConfig && (
          <NetworkConfig
            networkConfig={deviceInfo.networkConfig}
            showStatusIndicators={true} // T095
          />
        )}

        {/* Loading Indicator for Network Config (if loading individually) */}
        {deviceInfo.loadingStates.network && !deviceInfo.networkConfig && (
          <View style={styles.sectionLoading}>
            <ActivityIndicator size="small" animating={true} />
            <Text variant="bodyMedium" style={styles.loadingText}>
              Loading network configuration...
            </Text>
          </View>
        )}

        {/* Error for Network Config Only (if system info succeeded) */}
        {deviceInfo.errors.network && deviceInfo.systemInfo && (
          <View style={styles.sectionError}>
            <Text variant="titleMedium" style={styles.sectionErrorTitle}>
              Network Configuration Unavailable
            </Text>
            <Text variant="bodySmall" style={styles.sectionErrorMessage}>
              {deviceInfo.errors.network.message}
            </Text>
          </View>
        )}

        {/* Configure Button */}
        {availability.hasAnyData && (
          <Button
            mode="contained"
            icon="cog"
            onPress={() =>
              router.push({
                pathname: '/device/configure',
                params: { id, host, port },
              })
            }
            style={styles.configureButton}
            accessibilityLabel="Configure device"
          >
            Configure
          </Button>
        )}

        {/* Device Connection Info */}
        <View style={styles.connectionInfo}>
          <Text variant="bodySmall" style={styles.connectionInfoText}>
            Connected to {host}:{port}
          </Text>
        </View>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  scrollView: {
    flex: 1,
  },
  scrollContent: {
    padding: spacing.md,
    paddingBottom: spacing.xl,
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingVertical: spacing.xxl,
  },
  loadingText: {
    marginTop: spacing.md,
    opacity: 0.7,
  },
  errorContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.xl,
  },
  errorTitle: {
    marginBottom: spacing.md,
    textAlign: 'center',
  },
  errorMessage: {
    marginBottom: spacing.lg,
    textAlign: 'center',
    opacity: 0.7,
  },
  errorButton: {
    marginTop: spacing.md,
  },
  sectionLoading: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: spacing.lg,
  },
  sectionError: {
    padding: spacing.md,
    marginVertical: spacing.sm,
    backgroundColor: '#FFEBEE', // Light red background
    borderRadius: 8,
  },
  sectionErrorTitle: {
    marginBottom: spacing.xs,
    color: '#C62828', // Dark red text
  },
  sectionErrorMessage: {
    opacity: 0.8,
    color: '#C62828',
  },
  configureButton: {
    marginTop: spacing.lg,
    marginHorizontal: spacing.md,
  },
  connectionInfo: {
    marginTop: spacing.xl,
    paddingTop: spacing.md,
    borderTopWidth: 1,
    borderTopColor: '#E0E0E0',
  },
  connectionInfoText: {
    textAlign: 'center',
    opacity: 0.5,
    fontFamily: 'monospace',
  },
});
