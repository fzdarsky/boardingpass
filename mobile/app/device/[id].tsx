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
import { ScrollView, View, StyleSheet, RefreshControl, Pressable } from 'react-native';
import { useLocalSearchParams, useRouter, Stack } from 'expo-router';
import { Text, ActivityIndicator, Button, Banner, useTheme } from 'react-native-paper';
import MaterialCommunityIcons from '@expo/vector-icons/MaterialCommunityIcons';
import { SystemInformationCard } from '../../src/components/DeviceInfo';
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

  // Detect server-side session invalidation and clear local session
  useEffect(() => {
    const msg = deviceInfo.error?.message ?? '';
    if (msg.includes('Invalid session token') && id) {
      sessionManager.clearSession(id);
      router.back();
    }
  }, [deviceInfo.error, id, router]);

  // Handle missing route parameters
  if (!id || !host || !port) {
    return (
      <View style={styles.container}>
        <Stack.Screen
          options={{
            title: 'Device Details',
            // eslint-disable-next-line react/no-unstable-nested-components
            headerLeft: () => (
              <Pressable
                onPress={() => router.back()}
                style={styles.headerIcon}
                accessibilityLabel="Go back"
              >
                <MaterialCommunityIcons
                  name="arrow-left"
                  size={22}
                  color={theme.colors.onPrimary}
                />
              </Pressable>
            ),
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
            headerLeft: () => (
              <Pressable
                onPress={() => router.back()}
                style={styles.headerIcon}
                accessibilityLabel="Go back"
              >
                <MaterialCommunityIcons
                  name="arrow-left"
                  size={22}
                  color={theme.colors.onPrimary}
                />
              </Pressable>
            ),
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
            headerLeft: () => (
              <Pressable
                onPress={() => router.back()}
                style={styles.headerIcon}
                accessibilityLabel="Go back"
              >
                <MaterialCommunityIcons
                  name="arrow-left"
                  size={22}
                  color={theme.colors.onPrimary}
                />
              </Pressable>
            ),
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
          headerLeft: () => (
            <Pressable
              onPress={() => router.back()}
              style={styles.headerIcon}
              accessibilityLabel="Go back"
            >
              <MaterialCommunityIcons name="arrow-left" size={22} color={theme.colors.onPrimary} />
            </Pressable>
          ),
          // eslint-disable-next-line react/no-unstable-nested-components
          headerRight: () => (
            <View style={styles.headerRight}>
              <Pressable
                onPress={deviceInfo.refetch}
                disabled={deviceInfo.isLoading}
                style={styles.headerIcon}
                accessibilityLabel="Refresh"
              >
                <MaterialCommunityIcons
                  name="refresh"
                  size={22}
                  color={
                    deviceInfo.isLoading ? theme.colors.onPrimary + '66' : theme.colors.onPrimary
                  }
                />
              </Pressable>
              <Pressable
                onPress={() =>
                  router.push({
                    pathname: '/device/configure',
                    params: { id, host, port },
                  })
                }
                style={styles.headerIcon}
                accessibilityLabel="Configure device"
              >
                <MaterialCommunityIcons name="cog" size={22} color={theme.colors.onPrimary} />
              </Pressable>
            </View>
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
          <SystemInformationCard
            systemInfo={deviceInfo.systemInfo}
            networkConfig={deviceInfo.networkConfig}
            connectionHost={host}
          />
        )}

        {/* Loading Indicator (if neither query has returned yet) */}
        {deviceInfo.loadingStates.info &&
          !deviceInfo.systemInfo &&
          deviceInfo.loadingStates.network &&
          !deviceInfo.networkConfig && (
            <View style={styles.sectionLoading}>
              <ActivityIndicator size="small" animating={true} />
              <Text variant="bodyMedium" style={styles.loadingText}>
                Loading device information...
              </Text>
            </View>
          )}

        {/* Error for System Info (if network succeeded) */}
        {deviceInfo.errors.info && deviceInfo.networkConfig && (
          <View style={[styles.sectionError, { backgroundColor: theme.colors.errorContainer }]}>
            <Text
              variant="titleMedium"
              style={[styles.sectionErrorTitle, { color: theme.colors.onErrorContainer }]}
            >
              System Information Unavailable
            </Text>
            <Text
              variant="bodySmall"
              style={[styles.sectionErrorMessage, { color: theme.colors.onErrorContainer }]}
            >
              {deviceInfo.errors.info.message}
            </Text>
          </View>
        )}

        {/* Error for Network Config (if system info succeeded) */}
        {deviceInfo.errors.network && deviceInfo.systemInfo && (
          <View style={[styles.sectionError, { backgroundColor: theme.colors.errorContainer }]}>
            <Text
              variant="titleMedium"
              style={[styles.sectionErrorTitle, { color: theme.colors.onErrorContainer }]}
            >
              Network Configuration Unavailable
            </Text>
            <Text
              variant="bodySmall"
              style={[styles.sectionErrorMessage, { color: theme.colors.onErrorContainer }]}
            >
              {deviceInfo.errors.network.message}
            </Text>
          </View>
        )}
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
    borderRadius: 8,
  },
  sectionErrorTitle: {
    marginBottom: spacing.xs,
  },
  sectionErrorMessage: {
    opacity: 0.8,
  },
  headerRight: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 16,
  },
  headerIcon: {
    padding: 4,
  },
});
