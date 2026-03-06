/**
 * WiFiStep Component (Step 2a — conditional)
 *
 * Shown after InterfaceStep when a WiFi interface is selected.
 * Triggers wifi-scan command on mount, displays scanned networks in a table,
 * allows SSID selection, and shows a password field for secured networks.
 */

import React, { useCallback, useEffect, useState } from 'react';
import { View, StyleSheet, ScrollView } from 'react-native';
import {
  Text,
  RadioButton,
  TextInput,
  DataTable,
  Button,
  ActivityIndicator,
  useTheme,
} from 'react-native-paper';
import MaterialCommunityIcons from '@expo/vector-icons/MaterialCommunityIcons';
import { useWizard } from '../../contexts/WizardContext';
import type { WiFiNetwork } from '../../types/wizard';
import type { APIClient } from '../../services/api/client';
import { executeCommand } from '../../services/api/command';
import { spacing } from '../../theme';

interface WiFiStepProps {
  apiClient?: APIClient;
}

/**
 * Derive frequency band from frequency in MHz.
 */
export function deriveBand(frequency: number): string {
  if (frequency >= 2400 && frequency <= 2483) return '2.4 GHz';
  if (frequency >= 5150 && frequency <= 5850) return '5 GHz';
  if (frequency >= 5925 && frequency <= 7125) return '6 GHz';
  return '';
}

/**
 * Parse wifi-scan command JSON output into WiFiNetwork[], deduplicated by SSID.
 *
 * Multiple BSSIDs (access points) for the same SSID are merged into a single
 * entry. The strongest-signal BSSID is kept as the representative; all unique
 * bands are collected into `bands[]`.
 */
export function parseWifiScanOutput(stdout: string): WiFiNetwork[] {
  try {
    const raw = JSON.parse(stdout);
    if (!Array.isArray(raw)) return [];

    // Group by SSID, keeping the strongest-signal entry per SSID
    const bySsid = new Map<string, WiFiNetwork>();

    for (const item of raw as Array<{
      device?: string;
      ssid?: string;
      bssid?: string;
      signal?: number;
      security?: string;
      channel?: number;
      frequency?: number;
      rate?: string;
    }>) {
      const ssid = item.ssid || '';
      const signal = item.signal || 0;
      const band = deriveBand(item.frequency || 0);

      const existing = bySsid.get(ssid);
      if (!existing || signal > existing.signal) {
        // New SSID or stronger signal — use this entry as the representative
        bySsid.set(ssid, {
          device: item.device || '',
          ssid,
          bssid: item.bssid || '',
          signal,
          security: item.security || 'open',
          channel: item.channel || 0,
          frequency: item.frequency || 0,
          band,
          bands: existing ? [...new Set([...existing.bands, band])] : [band],
          rate: item.rate || '',
        });
      } else {
        // Weaker signal — just add the band if it's new
        if (band && !existing.bands.includes(band)) {
          existing.bands.push(band);
        }
      }
    }

    return Array.from(bySsid.values());
  } catch {
    return [];
  }
}

/**
 * Signal strength indicator icon.
 */
export function signalIcon(signal: number): string {
  if (signal >= 75) return 'wifi-strength-4';
  if (signal >= 50) return 'wifi-strength-3';
  if (signal >= 25) return 'wifi-strength-2';
  return 'wifi-strength-1';
}

export default function WiFiStep({ apiClient }: WiFiStepProps) {
  const { state, updateInterface } = useWizard();
  const theme = useTheme();

  const [networks, setNetworks] = useState<WiFiNetwork[]>([]);
  const [scanning, setScanning] = useState(false);
  const [scanError, setScanError] = useState<string | null>(null);
  const [password, setPassword] = useState(state.networkInterface.wifi?.password || '');

  const scan = useCallback(async () => {
    if (!apiClient) {
      setScanError('No API client available');
      return;
    }

    setScanning(true);
    setScanError(null);

    try {
      const result = await executeCommand(apiClient, 'wifi-scan');
      if (result.exit_code !== 0) {
        setScanError(result.stderr || 'WiFi scan failed');
        setScanning(false);
        return;
      }

      const parsed = parseWifiScanOutput(result.stdout);
      setNetworks(parsed);
    } catch (err) {
      setScanError(err instanceof Error ? err.message : 'WiFi scan failed');
    } finally {
      setScanning(false);
    }
  }, [apiClient]);

  // Scan on mount
  useEffect(() => {
    scan();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSelect = useCallback(
    (network: WiFiNetwork) => {
      const needsPassword = network.security !== 'open' && network.security !== '--';
      setPassword('');
      updateInterface({
        ...state.networkInterface,
        wifi: {
          ssid: network.ssid,
          bssid: network.bssid,
          security: network.security,
          password: needsPassword ? '' : null,
        },
      });
    },
    [state.networkInterface, updateInterface]
  );

  const handlePasswordChange = useCallback(
    (text: string) => {
      setPassword(text);
      if (state.networkInterface.wifi) {
        updateInterface({
          ...state.networkInterface,
          wifi: {
            ...state.networkInterface.wifi,
            password: text || null,
          },
        });
      }
    },
    [state.networkInterface, updateInterface]
  );

  const selectedSsid = state.networkInterface.wifi ? state.networkInterface.wifi.ssid : null;
  const needsPassword =
    state.networkInterface.wifi &&
    state.networkInterface.wifi.security !== 'open' &&
    state.networkInterface.wifi.security !== '--';

  return (
    <View style={styles.container}>
      <Text variant="titleMedium" style={[styles.title, { color: theme.colors.onSurface }]}>
        Select WiFi Network
      </Text>

      <Text
        variant="bodyMedium"
        style={[styles.description, { color: theme.colors.onSurfaceVariant }]}
      >
        Choose a WiFi network to connect to.
      </Text>

      {/* Scan controls */}
      <View style={styles.scanRow}>
        <Button
          mode="outlined"
          onPress={scan}
          disabled={scanning}
          loading={scanning}
          compact
          accessibilityLabel="Rescan WiFi networks"
        >
          Rescan
        </Button>
        {scanning && (
          <Text
            variant="bodySmall"
            style={[styles.scanStatus, { color: theme.colors.onSurfaceVariant }]}
          >
            Scanning...
          </Text>
        )}
      </View>

      {/* Scan error */}
      {scanError && (
        <Text variant="bodySmall" style={[styles.errorText, { color: theme.colors.error }]}>
          {scanError}
        </Text>
      )}

      {/* Loading state */}
      {scanning && networks.length === 0 && (
        <View style={styles.emptyState}>
          <ActivityIndicator size="large" animating={true} />
          <Text
            variant="bodyMedium"
            style={[styles.emptyText, { color: theme.colors.onSurfaceVariant }]}
          >
            Scanning for WiFi networks...
          </Text>
        </View>
      )}

      {/* Empty state */}
      {!scanning && networks.length === 0 && !scanError && (
        <View style={styles.emptyState}>
          <MaterialCommunityIcons name="wifi-off" size={48} color={theme.colors.onSurfaceVariant} />
          <Text
            variant="bodyMedium"
            style={[styles.emptyText, { color: theme.colors.onSurfaceVariant }]}
          >
            No WiFi networks found. Try rescanning.
          </Text>
        </View>
      )}

      {/* Network table */}
      {networks.length > 0 && (
        <ScrollView horizontal style={styles.tableScroll}>
          <DataTable>
            <DataTable.Header>
              <DataTable.Title style={styles.radioCol}>{''}</DataTable.Title>
              <DataTable.Title style={styles.ssidCol}>SSID</DataTable.Title>
              <DataTable.Title style={styles.signalCol}>Signal</DataTable.Title>
              <DataTable.Title style={styles.securityCol}>Security</DataTable.Title>
              <DataTable.Title style={styles.bandCol}>Band</DataTable.Title>
            </DataTable.Header>

            {networks.map(network => {
              const isSelected = selectedSsid === network.ssid;

              return (
                <DataTable.Row
                  key={network.ssid || network.bssid}
                  onPress={() => handleSelect(network)}
                  style={isSelected ? { backgroundColor: theme.colors.surfaceVariant } : undefined}
                >
                  <DataTable.Cell style={styles.radioCol}>
                    <RadioButton
                      value={network.ssid}
                      status={isSelected ? 'checked' : 'unchecked'}
                      onPress={() => handleSelect(network)}
                    />
                  </DataTable.Cell>
                  <DataTable.Cell style={styles.ssidCol}>
                    <Text variant="bodySmall">{network.ssid || '(Hidden)'}</Text>
                  </DataTable.Cell>
                  <DataTable.Cell style={styles.signalCol}>
                    <MaterialCommunityIcons
                      name={signalIcon(network.signal) as 'wifi-strength-4'}
                      size={18}
                      color={theme.colors.onSurface}
                    />
                  </DataTable.Cell>
                  <DataTable.Cell style={styles.securityCol}>{network.security}</DataTable.Cell>
                  <DataTable.Cell style={styles.bandCol}>
                    {network.bands.filter(Boolean).join(', ') || '-'}
                  </DataTable.Cell>
                </DataTable.Row>
              );
            })}
          </DataTable>
        </ScrollView>
      )}

      {/* Password field for secured networks */}
      {needsPassword && (
        <View style={styles.passwordSection}>
          <TextInput
            label="WiFi Password"
            value={password}
            onChangeText={handlePasswordChange}
            mode="outlined"
            secureTextEntry
            autoCapitalize="none"
            autoCorrect={false}
            accessibilityLabel="WiFi password"
            style={styles.input}
          />
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginTop: spacing.md,
  },
  title: {
    marginBottom: spacing.sm,
  },
  description: {
    marginBottom: spacing.md,
  },
  scanRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: spacing.sm,
  },
  scanStatus: {
    marginLeft: spacing.sm,
  },
  errorText: {
    marginBottom: spacing.sm,
  },
  emptyState: {
    alignItems: 'center',
    paddingVertical: spacing.xl,
  },
  emptyText: {
    marginTop: spacing.md,
    textAlign: 'center',
  },
  tableScroll: {
    marginBottom: spacing.md,
  },
  radioCol: { width: 48 },
  ssidCol: { width: 160 },
  signalCol: { width: 60 },
  securityCol: { width: 100 },
  bandCol: { width: 100 },
  passwordSection: {
    marginTop: spacing.sm,
  },
  input: {
    marginBottom: spacing.xs,
  },
});
