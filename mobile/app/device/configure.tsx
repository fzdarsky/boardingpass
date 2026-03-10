/**
 * Wizard Screen
 *
 * Wraps WizardContainer in WizardContext provider.
 * Fetches device info + network data on mount and passes to context as initial state.
 *
 * Route: /device/configure?id=...&host=...&port=...
 */

import React, { useMemo, useState, useEffect, useCallback } from 'react';
import { View, StyleSheet, Pressable } from 'react-native';
import { useLocalSearchParams, useRouter, Stack } from 'expo-router';
import { Text, ActivityIndicator, Button, useTheme } from 'react-native-paper';
import MaterialCommunityIcons from '@expo/vector-icons/MaterialCommunityIcons';
import { WizardProvider } from '../../src/contexts/WizardContext';
import WizardContainer from '../../src/components/ConfigWizard/WizardContainer';
import { createAPIClient } from '../../src/services/api/client';
import { getSystemInfo } from '../../src/services/api/info';
import { getNetworkConfig } from '../../src/services/api/network';
import { sessionManager } from '../../src/services/auth/session';
import { createInitialWizardState } from '../../src/types/wizard';
import type { APIClient } from '../../src/services/api/client';
import type { components } from '../../src/types/api';
import { spacing } from '../../src/theme';

type SystemInfoType = components['schemas']['SystemInfo'];
type NetworkConfigType = components['schemas']['NetworkConfig'];

export default function ConfigureScreen() {
  const { id, host, port } = useLocalSearchParams<{
    id: string;
    host: string;
    port: string;
  }>();

  const router = useRouter();
  const theme = useTheme();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Device data for wizard initialization
  const [systemInfo, setSystemInfo] = useState<SystemInfoType | null>(null);
  const [networkConfig, setNetworkConfig] = useState<NetworkConfigType | null>(null);
  const [apiClient, setApiClient] = useState<APIClient | null>(null);

  // Load session token and fetch device data
  useEffect(() => {
    if (!id) return;

    let cancelled = false;

    async function loadData() {
      try {
        const t = await sessionManager.getValidToken(id!);
        if (cancelled) return;
        if (!t) {
          setError('Session expired. Please authenticate again.');
          setLoading(false);
          return;
        }

        if (!host || !port) {
          setError('Missing device connection parameters.');
          setLoading(false);
          return;
        }

        const client = createAPIClient(host, parseInt(port, 10));
        client.setAuthToken(t);

        // Fetch device info and network config in parallel
        const [info, network] = await Promise.all([
          getSystemInfo(client),
          getNetworkConfig(client),
        ]);

        if (cancelled) return;
        setApiClient(client);
        setSystemInfo(info);
        setNetworkConfig(network);
        setLoading(false);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : 'Failed to load device data');
        setLoading(false);
      }
    }

    loadData();
    return () => {
      cancelled = true;
    };
  }, [id, host, port]);

  // Determine which interface the app is connected through
  const serviceInterfaceName = useMemo(() => {
    if (!networkConfig || !host) return null;

    for (const iface of networkConfig.interfaces) {
      for (const addr of iface.ip_addresses) {
        if (addr.ip === host) {
          return iface.name;
        }
      }
    }
    return null;
  }, [networkConfig, host]);

  // Create initial wizard state pre-populated from device data
  const initialWizardState = useMemo(() => {
    const state = createInitialWizardState();
    if (systemInfo?.hostname) {
      state.hostname.hostname = systemInfo.hostname;
      state.hostname.original = systemInfo.hostname;
    }
    if (serviceInterfaceName) {
      state.serviceInterfaceName = serviceInterfaceName;
    }
    if (systemInfo?.os?.version) {
      state.osVersion = systemInfo.os.version;
    }
    return state;
  }, [systemInfo, serviceInterfaceName]);

  const handleComplete = useCallback(() => {
    // For US1, just navigate back (apply logic comes in US2)
    router.back();
  }, [router]);

  // Missing params
  if (!id || !host || !port) {
    return (
      <View style={styles.container}>
        <Stack.Screen
          options={{
            title: 'Configure Device',
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
        <View style={styles.centerContent}>
          <Text variant="headlineSmall">Missing Parameters</Text>
          <Text variant="bodyMedium" style={styles.errorText}>
            Device information is incomplete.
          </Text>
          <Button mode="contained" onPress={() => router.back()}>
            Go Back
          </Button>
        </View>
      </View>
    );
  }

  // Loading
  if (loading) {
    return (
      <View style={styles.container}>
        <Stack.Screen
          options={{
            title: 'Configure Device',
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
        <View style={styles.centerContent}>
          <ActivityIndicator size="large" animating={true} />
          <Text variant="bodyMedium" style={styles.loadingText}>
            Loading device information...
          </Text>
        </View>
      </View>
    );
  }

  // Error
  if (error) {
    return (
      <View style={styles.container}>
        <Stack.Screen
          options={{
            title: 'Configure Device',
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
        <View style={styles.centerContent}>
          <Text variant="headlineSmall">Error</Text>
          <Text variant="bodyMedium" style={styles.errorText}>
            {error}
          </Text>
          <Button mode="contained" onPress={() => router.back()}>
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
          title: `Configure ${host}`,
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
        }}
      />

      <WizardProvider initialState={initialWizardState}>
        <WizardContainer
          interfaces={networkConfig?.interfaces || []}
          apiClient={apiClient ?? undefined}
          onComplete={handleComplete}
        />
      </WizardProvider>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  centerContent: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: spacing.xl,
  },
  loadingText: {
    marginTop: spacing.md,
    opacity: 0.7,
  },
  errorText: {
    marginVertical: spacing.md,
    textAlign: 'center',
    opacity: 0.7,
  },
  headerIcon: {
    padding: 4,
  },
});
