/**
 * InterfaceStep Component (Step 2)
 *
 * Displays a DataTable of network interfaces (name, type, MAC, vendor, model,
 * speed, state/carrier) with radio selection and optional VLAN ID input.
 */

import React, { useCallback, useEffect, useMemo } from 'react';
import { View, StyleSheet, ScrollView } from 'react-native';
import {
  Text,
  RadioButton,
  TextInput,
  HelperText,
  DataTable,
  Badge,
  useTheme,
} from 'react-native-paper';
import { useWizard } from '../../contexts/WizardContext';
import type { components } from '../../types/api';
import { spacing } from '../../theme';

type NetworkInterface = components['schemas']['NetworkInterface'];

/** Interface types that can be used as enrollment targets. */
const ENROLLABLE_TYPES: ReadonlySet<string> = new Set(['ethernet', 'wifi']);

interface InterfaceStepProps {
  interfaces: NetworkInterface[];
  children?: React.ReactNode;
}

export default function InterfaceStep({ interfaces, children }: InterfaceStepProps) {
  const { state, updateInterface } = useWizard();
  const theme = useTheme();

  // Only show interfaces that can be used for enrollment
  const selectableInterfaces = useMemo(
    () => interfaces.filter(iface => ENROLLABLE_TYPES.has(iface.type)),
    [interfaces]
  );

  // Auto-select when only one enrollable interface exists
  useEffect(() => {
    if (selectableInterfaces.length === 1 && !state.networkInterface.interfaceName) {
      const iface = selectableInterfaces[0];
      updateInterface({
        interfaceName: iface.name,
        interfaceType: iface.type,
        vlanId: state.networkInterface.vlanId,
        wifi: null,
      });
    }
  }, [
    selectableInterfaces,
    state.networkInterface.interfaceName,
    state.networkInterface.vlanId,
    updateInterface,
  ]);

  const handleSelect = useCallback(
    (iface: NetworkInterface) => {
      updateInterface({
        interfaceName: iface.name,
        interfaceType: iface.type,
        vlanId: state.networkInterface.vlanId,
        wifi: null,
      });
    },
    [updateInterface, state.networkInterface.vlanId]
  );

  const handleVlanChange = useCallback(
    (text: string) => {
      const num = parseInt(text, 10);
      updateInterface({
        ...state.networkInterface,
        vlanId: text === '' ? null : isNaN(num) ? null : num,
      });
    },
    [state.networkInterface, updateInterface]
  );

  const vlanError =
    state.networkInterface.vlanId !== null &&
    (state.networkInterface.vlanId < 1 || state.networkInterface.vlanId > 4094)
      ? 'VLAN ID must be between 1 and 4094'
      : null;

  return (
    <View style={styles.container}>
      <Text variant="titleMedium" style={[styles.title, { color: theme.colors.onSurface }]}>
        Select Network Interface
      </Text>

      <Text
        variant="bodyMedium"
        style={[styles.description, { color: theme.colors.onSurfaceVariant }]}
      >
        Choose the network interface to use for enrollment.
      </Text>

      <ScrollView horizontal style={styles.tableScroll}>
        <DataTable>
          <DataTable.Header>
            <DataTable.Title style={styles.radioCol}>{''}</DataTable.Title>
            <DataTable.Title style={styles.nameCol}>Name</DataTable.Title>
            <DataTable.Title style={styles.typeCol}>Type</DataTable.Title>
            <DataTable.Title style={styles.macCol}>MAC</DataTable.Title>
            <DataTable.Title style={styles.vendorCol}>Vendor</DataTable.Title>
            <DataTable.Title style={styles.speedCol}>Speed</DataTable.Title>
            <DataTable.Title style={styles.stateCol}>State</DataTable.Title>
          </DataTable.Header>

          {selectableInterfaces.map(iface => {
            const isSelected = state.networkInterface.interfaceName === iface.name;

            return (
              <DataTable.Row
                key={iface.name}
                onPress={() => handleSelect(iface)}
                style={isSelected ? { backgroundColor: theme.colors.surfaceVariant } : undefined}
              >
                <DataTable.Cell style={styles.radioCol}>
                  <RadioButton
                    value={iface.name}
                    status={isSelected ? 'checked' : 'unchecked'}
                    onPress={() => handleSelect(iface)}
                  />
                </DataTable.Cell>
                <DataTable.Cell style={styles.nameCol}>
                  <Text variant="bodySmall">
                    {iface.name}
                    {iface.name === state.serviceInterfaceName && (
                      <Text style={{ color: theme.colors.error }}>{' *'}</Text>
                    )}
                  </Text>
                </DataTable.Cell>
                <DataTable.Cell style={styles.typeCol}>{iface.type}</DataTable.Cell>
                <DataTable.Cell style={styles.macCol}>
                  <Text variant="bodySmall" style={styles.monoText}>
                    {iface.mac_address}
                  </Text>
                </DataTable.Cell>
                <DataTable.Cell style={styles.vendorCol}>{iface.vendor || '-'}</DataTable.Cell>
                <DataTable.Cell style={styles.speedCol}>
                  {iface.speed > 0 ? `${iface.speed} Mbps` : '-'}
                </DataTable.Cell>
                <DataTable.Cell style={styles.stateCol}>
                  <Badge
                    style={[
                      styles.badgeFontSize,
                      {
                        backgroundColor: iface.carrier
                          ? theme.colors.tertiary
                          : theme.colors.surfaceVariant,
                        color: iface.carrier
                          ? theme.colors.onTertiary
                          : theme.colors.onSurfaceVariant,
                      },
                    ]}
                  >
                    {iface.carrier ? 'CONNECTED' : 'DISCONNECTED'}
                  </Badge>
                </DataTable.Cell>
              </DataTable.Row>
            );
          })}
        </DataTable>
      </ScrollView>

      {state.serviceInterfaceName && (
        <Text variant="bodySmall" style={styles.footnote}>
          <Text style={{ color: theme.colors.error }}>{'* '}</Text>
          = interface currently used for provisioning
        </Text>
      )}

      {/* Injected sub-step (e.g. WiFi network selection) */}
      {children}

      {/* VLAN Configuration — always visible, empty means no VLAN */}
      <View style={styles.vlanSection}>
        <Text variant="titleMedium" style={[styles.title, { color: theme.colors.onSurface }]}>
          Configure a VLAN ID (optional)
        </Text>

        <Text
          variant="bodyMedium"
          style={[styles.description, { color: theme.colors.onSurfaceVariant }]}
        >
          Enter a VLAN ID if required by your network.
        </Text>
        <TextInput
          label="VLAN ID (1–4094) or empty for none"
          value={state.networkInterface.vlanId?.toString() || ''}
          onChangeText={handleVlanChange}
          mode="outlined"
          keyboardType="number-pad"
          error={!!vlanError}
          accessibilityLabel="VLAN ID"
        />
        <HelperText type="error" visible={!!vlanError}>
          {vlanError}
        </HelperText>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    paddingHorizontal: spacing.md,
  },
  title: {
    marginBottom: spacing.sm,
  },
  description: {
    marginBottom: spacing.md,
  },
  tableScroll: {
    marginBottom: spacing.md,
  },
  radioCol: { width: 48 },
  nameCol: { width: 120 },
  typeCol: { width: 80 },
  macCol: { width: 140 },
  vendorCol: { width: 140 },
  speedCol: { width: 90 },
  stateCol: { width: 120 },
  monoText: {
    fontFamily: 'monospace',
    fontSize: 11,
  },
  badgeFontSize: {
    fontSize: 10,
    paddingHorizontal: 8,
  },
  vlanSection: {
    marginTop: spacing.md,
  },
  footnote: {
    fontStyle: 'italic',
    opacity: 0.7,
  },
});
