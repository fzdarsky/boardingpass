/**
 * ReviewPage Component
 *
 * Deferred-mode summary of all queued configuration changes.
 * Shows a human-readable list of what will be applied on reboot.
 * "Confirm & Reboot" sends the atomic bundle and triggers device reboot.
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, Button, Divider, useTheme } from 'react-native-paper';
import MaterialCommunityIcons from '@expo/vector-icons/MaterialCommunityIcons';
import { useWizard } from '../../contexts/WizardContext';
import { STEP_LABELS, WIZARD_STEPS } from '../../types/wizard';
import { spacing } from '../../theme';

interface ReviewPageProps {
  onConfirm: () => void;
  onBack: () => void;
  applying?: boolean;
}

function ReviewSection({ title, children }: { title: string; children: React.ReactNode }) {
  const theme = useTheme();
  return (
    <View style={styles.section}>
      <Text variant="titleSmall" style={[styles.sectionTitle, { color: theme.colors.primary }]}>
        {title}
      </Text>
      {children}
    </View>
  );
}

function ReviewItem({ label, value }: { label: string; value: string }) {
  const theme = useTheme();
  return (
    <View style={styles.item}>
      <Text variant="bodySmall" style={{ color: theme.colors.onSurfaceVariant }}>
        {label}
      </Text>
      <Text variant="bodyMedium" style={{ color: theme.colors.onSurface }}>
        {value}
      </Text>
    </View>
  );
}

const RebootIcon = ({ size, color }: { size: number; color: string }) => (
  <MaterialCommunityIcons name="restart" size={size} color={color} />
);

export default function ReviewPage({ onConfirm, onBack, applying }: ReviewPageProps) {
  const { state } = useWizard();
  const theme = useTheme();

  const ipv4Method = state.addressing.ipv4.method === 'dhcp' ? 'DHCP' : 'Static';
  const ipv6Method =
    state.addressing.ipv6.method === 'disabled'
      ? 'Disabled'
      : state.addressing.ipv6.method === 'dhcp'
        ? 'DHCP'
        : 'Static';

  return (
    <View style={styles.container}>
      <Text variant="titleMedium" style={[styles.title, { color: theme.colors.onSurface }]}>
        Review Configuration
      </Text>
      <Text
        variant="bodyMedium"
        style={[styles.description, { color: theme.colors.onSurfaceVariant }]}
      >
        The following changes will be applied when the device reboots.
      </Text>

      {/* Step 1: Hostname */}
      <ReviewSection title={STEP_LABELS[WIZARD_STEPS.HOSTNAME]}>
        <ReviewItem label="Hostname" value={state.hostname.hostname} />
      </ReviewSection>

      <Divider />

      {/* Step 2: Interface */}
      <ReviewSection title={STEP_LABELS[WIZARD_STEPS.INTERFACE]}>
        <ReviewItem label="Interface" value={state.networkInterface.interfaceName} />
        <ReviewItem label="Type" value={state.networkInterface.interfaceType} />
        {state.networkInterface.vlanId !== null && (
          <ReviewItem label="VLAN ID" value={String(state.networkInterface.vlanId)} />
        )}
        {state.networkInterface.wifi && (
          <ReviewItem label="WiFi SSID" value={state.networkInterface.wifi.ssid} />
        )}
      </ReviewSection>

      <Divider />

      {/* Step 3: Addressing */}
      <ReviewSection title={STEP_LABELS[WIZARD_STEPS.ADDRESSING]}>
        <ReviewItem label="IPv4" value={ipv4Method} />
        {state.addressing.ipv4.method === 'static' && (
          <>
            {state.addressing.ipv4.address && (
              <ReviewItem label="Address" value={state.addressing.ipv4.address} />
            )}
            {state.addressing.ipv4.subnetMask && (
              <ReviewItem label="Subnet Mask" value={state.addressing.ipv4.subnetMask} />
            )}
            {state.addressing.ipv4.gateway && (
              <ReviewItem label="Gateway" value={state.addressing.ipv4.gateway} />
            )}
          </>
        )}
        {!state.addressing.ipv4.dnsAuto && state.addressing.ipv4.dnsPrimary && (
          <ReviewItem label="DNS Primary" value={state.addressing.ipv4.dnsPrimary} />
        )}
        <ReviewItem label="IPv6" value={ipv6Method} />
        {state.addressing.ipv6.method === 'static' && state.addressing.ipv6.address && (
          <ReviewItem label="IPv6 Address" value={state.addressing.ipv6.address} />
        )}
      </ReviewSection>

      <Divider />

      {/* Step 4: Services */}
      <ReviewSection title={STEP_LABELS[WIZARD_STEPS.SERVICES]}>
        <ReviewItem
          label="NTP"
          value={
            state.services.ntp.mode === 'automatic'
              ? 'Automatic'
              : state.services.ntp.servers.join(', ')
          }
        />
        <ReviewItem
          label="Proxy"
          value={
            state.services.proxy
              ? `${state.services.proxy.hostname}:${state.services.proxy.port}`
              : 'None'
          }
        />
      </ReviewSection>

      <Divider />

      {/* Step 5: Enrollment */}
      <ReviewSection title={STEP_LABELS[WIZARD_STEPS.ENROLLMENT]}>
        <ReviewItem
          label="Red Hat Insights"
          value={
            state.enrollment.insights ? `Org ${state.enrollment.insights.orgId}` : 'Not configured'
          }
        />
        <ReviewItem
          label="Flight Control"
          value={
            state.enrollment.flightControl
              ? state.enrollment.flightControl.endpoint
              : 'Not configured'
          }
        />
      </ReviewSection>

      {/* Actions */}
      <View style={styles.actions}>
        <Button
          mode="outlined"
          onPress={onBack}
          disabled={applying}
          style={styles.actionButton}
          accessibilityLabel="Go back to edit configuration"
        >
          Back
        </Button>
        <Button
          mode="contained"
          onPress={onConfirm}
          loading={applying}
          disabled={applying}
          style={styles.actionButton}
          icon={RebootIcon}
          accessibilityLabel="Confirm configuration and reboot device"
        >
          Confirm & Reboot
        </Button>
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
    marginBottom: spacing.lg,
  },
  section: {
    paddingVertical: spacing.sm,
  },
  sectionTitle: {
    marginBottom: spacing.xs,
  },
  item: {
    marginLeft: spacing.md,
    marginBottom: spacing.xs,
  },
  actions: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: spacing.lg,
    paddingBottom: spacing.md,
  },
  actionButton: {
    minWidth: 120,
  },
});
