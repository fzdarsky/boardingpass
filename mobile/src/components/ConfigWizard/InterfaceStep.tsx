/**
 * InterfaceStep Component (Step 2)
 *
 * Displays a DataTable of network interfaces (name, type, MAC, vendor, model,
 * speed, state/carrier) with radio selection and optional VLAN ID input.
 * Highlights the service interface (the one the app is connected through).
 */

import React, { useCallback, useState } from 'react';
import { View, StyleSheet, ScrollView } from 'react-native';
import {
  Text,
  RadioButton,
  TextInput,
  HelperText,
  DataTable,
  useTheme,
  Chip,
} from 'react-native-paper';
import { useWizard } from '../../contexts/WizardContext';
import type { components } from '../../types/api';
import { spacing } from '../../theme';

type NetworkInterface = components['schemas']['NetworkInterface'];

interface InterfaceStepProps {
  interfaces: NetworkInterface[];
  serviceInterfaceName: string | null;
}

export default function InterfaceStep({ interfaces, serviceInterfaceName }: InterfaceStepProps) {
  const { state, updateInterface } = useWizard();
  const theme = useTheme();
  const [showVlan, setShowVlan] = useState(state.networkInterface.vlanId !== null);

  // Filter out loopback and virtual interfaces
  const selectableInterfaces = interfaces.filter(
    iface => iface.name !== 'lo' && iface.type !== 'virtual'
  );

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

  const handleVlanToggle = useCallback(() => {
    const newShowVlan = !showVlan;
    setShowVlan(newShowVlan);
    if (!newShowVlan) {
      updateInterface({
        ...state.networkInterface,
        vlanId: null,
      });
    }
  }, [showVlan, state.networkInterface, updateInterface]);

  const handleVlanChange = useCallback(
    (text: string) => {
      const num = parseInt(text, 10);
      updateInterface({
        ...state.networkInterface,
        vlanId: isNaN(num) ? null : num,
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
        Choose the network interface to configure for enrollment.
      </Text>

      <ScrollView horizontal style={styles.tableScroll}>
        <DataTable>
          <DataTable.Header>
            <DataTable.Title style={styles.radioCol}>{''}</DataTable.Title>
            <DataTable.Title style={styles.nameCol}>Name</DataTable.Title>
            <DataTable.Title style={styles.typeCol}>Type</DataTable.Title>
            <DataTable.Title style={styles.macCol}>MAC</DataTable.Title>
            <DataTable.Title style={styles.vendorCol}>Vendor</DataTable.Title>
            <DataTable.Title style={styles.speedCol} numeric>
              Speed
            </DataTable.Title>
            <DataTable.Title style={styles.stateCol}>State</DataTable.Title>
          </DataTable.Header>

          {selectableInterfaces.map(iface => {
            const isSelected = state.networkInterface.interfaceName === iface.name;
            const isService = iface.name === serviceInterfaceName;

            return (
              <DataTable.Row
                key={iface.name}
                onPress={() => handleSelect(iface)}
                style={[
                  isSelected && { backgroundColor: theme.colors.primaryContainer },
                  isService && styles.serviceRow,
                ]}
              >
                <DataTable.Cell style={styles.radioCol}>
                  <RadioButton
                    value={iface.name}
                    status={isSelected ? 'checked' : 'unchecked'}
                    onPress={() => handleSelect(iface)}
                  />
                </DataTable.Cell>
                <DataTable.Cell style={styles.nameCol}>
                  <View style={styles.nameCell}>
                    <Text variant="bodySmall">{iface.name}</Text>
                    {isService && (
                      <Chip compact textStyle={styles.chipText} style={styles.serviceChip}>
                        Service
                      </Chip>
                    )}
                  </View>
                </DataTable.Cell>
                <DataTable.Cell style={styles.typeCol}>{iface.type}</DataTable.Cell>
                <DataTable.Cell style={styles.macCol}>
                  <Text variant="bodySmall" style={styles.monoText}>
                    {iface.mac_address}
                  </Text>
                </DataTable.Cell>
                <DataTable.Cell style={styles.vendorCol}>{iface.vendor || '-'}</DataTable.Cell>
                <DataTable.Cell style={styles.speedCol} numeric>
                  {iface.speed > 0 ? `${iface.speed} Mbps` : '-'}
                </DataTable.Cell>
                <DataTable.Cell style={styles.stateCol}>
                  <Text
                    variant="bodySmall"
                    style={iface.carrier ? styles.stateConnected : styles.stateDisconnected}
                  >
                    {iface.carrier ? 'Connected' : 'Disconnected'}
                  </Text>
                </DataTable.Cell>
              </DataTable.Row>
            );
          })}
        </DataTable>
      </ScrollView>

      {/* VLAN Configuration */}
      <View style={styles.vlanSection}>
        <RadioButton.Item
          label="Configure VLAN"
          value="vlan"
          status={showVlan ? 'checked' : 'unchecked'}
          onPress={handleVlanToggle}
          accessibilityLabel="Enable VLAN configuration"
        />

        {showVlan && (
          <View style={styles.vlanInput}>
            <TextInput
              label="VLAN ID (1-4094)"
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
        )}
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
  stateCol: { width: 100 },
  nameCell: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  serviceRow: {
    borderLeftWidth: 3,
    borderLeftColor: '#FF9800',
  },
  serviceChip: {
    height: 20,
  },
  chipText: {
    fontSize: 9,
  },
  monoText: {
    fontFamily: 'monospace',
    fontSize: 11,
  },
  stateConnected: {
    color: '#4CAF50',
  },
  stateDisconnected: {
    color: '#9E9E9E',
  },
  vlanSection: {
    marginTop: spacing.sm,
  },
  vlanInput: {
    paddingLeft: spacing.xl,
    paddingRight: spacing.md,
  },
});
