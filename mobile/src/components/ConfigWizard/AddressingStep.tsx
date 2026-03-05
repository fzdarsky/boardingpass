/**
 * AddressingStep Component (Step 3)
 *
 * IPv4 radio (DHCP/Static) with conditional fields (address, subnet, gateway).
 * DNS auto checkbox with conditional DNS fields.
 * IPv6 radio (DHCP/Static/Disabled) with same pattern.
 */

import React, { useCallback, useState } from 'react';
import { View, StyleSheet } from 'react-native';
import {
  Text,
  TextInput,
  RadioButton,
  Switch,
  Divider,
  HelperText,
  useTheme,
} from 'react-native-paper';
import { useWizard } from '../../contexts/WizardContext';
import { validateIPv4, validateIPv6, validateSubnetMask } from '../../utils/network-validation';
import type { IPv4Config, IPv6Config } from '../../types/wizard';
import { spacing } from '../../theme';

export default function AddressingStep() {
  const { state, updateAddressing } = useWizard();
  const theme = useTheme();
  const [touched, setTouched] = useState<Record<string, boolean>>({});

  const markTouched = useCallback((field: string) => {
    setTouched(prev => ({ ...prev, [field]: true }));
  }, []);

  const updateIPv4 = useCallback(
    (updates: Partial<IPv4Config>) => {
      updateAddressing({
        ...state.addressing,
        ipv4: { ...state.addressing.ipv4, ...updates },
      });
    },
    [state.addressing, updateAddressing]
  );

  const updateIPv6 = useCallback(
    (updates: Partial<IPv6Config>) => {
      updateAddressing({
        ...state.addressing,
        ipv6: { ...state.addressing.ipv6, ...updates },
      });
    },
    [state.addressing, updateAddressing]
  );

  const ipv4 = state.addressing.ipv4;
  const ipv6 = state.addressing.ipv6;

  return (
    <View style={styles.container}>
      <Text variant="titleMedium" style={[styles.title, { color: theme.colors.onSurface }]}>
        Configure IP Addressing
      </Text>

      {/* IPv4 Section */}
      <Text variant="titleSmall" style={[styles.sectionTitle, { color: theme.colors.primary }]}>
        IPv4
      </Text>

      <RadioButton.Group
        value={ipv4.method}
        onValueChange={v => updateIPv4({ method: v as 'dhcp' | 'static' })}
      >
        <RadioButton.Item label="DHCP (Automatic)" value="dhcp" accessibilityLabel="IPv4 DHCP" />
        <RadioButton.Item label="Static" value="static" accessibilityLabel="IPv4 Static" />
      </RadioButton.Group>

      {ipv4.method === 'static' && (
        <View style={styles.staticFields}>
          <TextInput
            label="IPv4 Address"
            value={ipv4.address || ''}
            onChangeText={v => updateIPv4({ address: v })}
            onBlur={() => markTouched('ipv4Address')}
            mode="outlined"
            keyboardType="numeric"
            error={touched.ipv4Address && !!validateIPv4(ipv4.address || '')}
            placeholder="192.168.1.100"
            accessibilityLabel="IPv4 address"
            style={styles.input}
          />
          {touched.ipv4Address && validateIPv4(ipv4.address || '') && (
            <HelperText type="error" visible={true}>
              {validateIPv4(ipv4.address || '')}
            </HelperText>
          )}

          <TextInput
            label="Subnet Mask"
            value={ipv4.subnetMask || ''}
            onChangeText={v => updateIPv4({ subnetMask: v })}
            onBlur={() => markTouched('subnetMask')}
            mode="outlined"
            keyboardType="numeric"
            error={touched.subnetMask && !!validateSubnetMask(ipv4.subnetMask || '')}
            placeholder="255.255.255.0"
            accessibilityLabel="Subnet mask"
            style={styles.input}
          />
          {touched.subnetMask && validateSubnetMask(ipv4.subnetMask || '') && (
            <HelperText type="error" visible={true}>
              {validateSubnetMask(ipv4.subnetMask || '')}
            </HelperText>
          )}

          <TextInput
            label="Gateway"
            value={ipv4.gateway || ''}
            onChangeText={v => updateIPv4({ gateway: v })}
            onBlur={() => markTouched('gateway')}
            mode="outlined"
            keyboardType="numeric"
            error={touched.gateway && !!validateIPv4(ipv4.gateway || '')}
            placeholder="192.168.1.1"
            accessibilityLabel="Default gateway"
            style={styles.input}
          />
          {touched.gateway && validateIPv4(ipv4.gateway || '') && (
            <HelperText type="error" visible={true}>
              {validateIPv4(ipv4.gateway || '')}
            </HelperText>
          )}
        </View>
      )}

      {/* DNS Configuration */}
      <View style={styles.dnsRow}>
        <Text variant="bodyMedium">Automatic DNS</Text>
        <Switch
          value={ipv4.dnsAuto}
          onValueChange={v => updateIPv4({ dnsAuto: v })}
          accessibilityLabel="Automatic DNS configuration"
        />
      </View>

      {!ipv4.dnsAuto && (
        <View style={styles.staticFields}>
          <TextInput
            label="Primary DNS"
            value={ipv4.dnsPrimary || ''}
            onChangeText={v => updateIPv4({ dnsPrimary: v })}
            mode="outlined"
            keyboardType="numeric"
            placeholder="8.8.8.8"
            accessibilityLabel="Primary DNS server"
            style={styles.input}
          />
          <TextInput
            label="Secondary DNS (optional)"
            value={ipv4.dnsSecondary || ''}
            onChangeText={v => updateIPv4({ dnsSecondary: v || null })}
            mode="outlined"
            keyboardType="numeric"
            placeholder="8.8.4.4"
            accessibilityLabel="Secondary DNS server"
            style={styles.input}
          />
        </View>
      )}

      <Divider style={styles.divider} />

      {/* IPv6 Section */}
      <Text variant="titleSmall" style={[styles.sectionTitle, { color: theme.colors.primary }]}>
        IPv6
      </Text>

      <RadioButton.Group
        value={ipv6.method}
        onValueChange={v => updateIPv6({ method: v as 'dhcp' | 'static' | 'disabled' })}
      >
        <RadioButton.Item label="DHCP (Automatic)" value="dhcp" accessibilityLabel="IPv6 DHCP" />
        <RadioButton.Item label="Static" value="static" accessibilityLabel="IPv6 Static" />
        <RadioButton.Item label="Disabled" value="disabled" accessibilityLabel="IPv6 Disabled" />
      </RadioButton.Group>

      {ipv6.method === 'static' && (
        <View style={styles.staticFields}>
          <TextInput
            label="IPv6 Address (with prefix)"
            value={ipv6.address || ''}
            onChangeText={v => updateIPv6({ address: v })}
            onBlur={() => markTouched('ipv6Address')}
            mode="outlined"
            autoCapitalize="none"
            error={touched.ipv6Address && !!validateIPv6(ipv6.address || '', true)}
            placeholder="2001:db8::1/64"
            accessibilityLabel="IPv6 address with prefix"
            style={styles.input}
          />
          {touched.ipv6Address && validateIPv6(ipv6.address || '', true) && (
            <HelperText type="error" visible={true}>
              {validateIPv6(ipv6.address || '', true)}
            </HelperText>
          )}

          <TextInput
            label="IPv6 Gateway (optional)"
            value={ipv6.gateway || ''}
            onChangeText={v => updateIPv6({ gateway: v || null })}
            mode="outlined"
            autoCapitalize="none"
            placeholder="2001:db8::ffff"
            accessibilityLabel="IPv6 gateway"
            style={styles.input}
          />
        </View>
      )}

      {ipv6.method !== 'disabled' && (
        <>
          <View style={styles.dnsRow}>
            <Text variant="bodyMedium">Automatic IPv6 DNS</Text>
            <Switch
              value={ipv6.dnsAuto}
              onValueChange={v => updateIPv6({ dnsAuto: v })}
              accessibilityLabel="Automatic IPv6 DNS"
            />
          </View>

          {!ipv6.dnsAuto && (
            <View style={styles.staticFields}>
              <TextInput
                label="Primary IPv6 DNS"
                value={ipv6.dnsPrimary || ''}
                onChangeText={v => updateIPv6({ dnsPrimary: v || null })}
                mode="outlined"
                autoCapitalize="none"
                placeholder="2001:4860:4860::8888"
                accessibilityLabel="Primary IPv6 DNS server"
                style={styles.input}
              />
              <TextInput
                label="Secondary IPv6 DNS (optional)"
                value={ipv6.dnsSecondary || ''}
                onChangeText={v => updateIPv6({ dnsSecondary: v || null })}
                mode="outlined"
                autoCapitalize="none"
                placeholder="2001:4860:4860::8844"
                accessibilityLabel="Secondary IPv6 DNS server"
                style={styles.input}
              />
            </View>
          )}
        </>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    paddingHorizontal: spacing.md,
  },
  title: {
    marginBottom: spacing.md,
  },
  sectionTitle: {
    marginTop: spacing.sm,
    marginBottom: spacing.xs,
  },
  staticFields: {
    paddingLeft: spacing.md,
    marginBottom: spacing.sm,
  },
  input: {
    marginBottom: spacing.xs,
  },
  dnsRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
  },
  divider: {
    marginVertical: spacing.md,
  },
});
