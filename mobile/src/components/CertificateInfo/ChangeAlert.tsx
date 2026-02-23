/**
 * Certificate Change Alert Component
 *
 * Alert component for notifying users when a certificate has changed.
 * Implements FR-035: Certificate change detection and user notification.
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Banner, Icon, Text, useTheme } from 'react-native-paper';
import { CertificateInfo } from '../../types/certificate';

export interface CertificateChangeAlertProps {
  visible: boolean;
  oldCertificate?: CertificateInfo;
  newCertificate?: CertificateInfo;
  onViewDetails: () => void;
  onDismiss: () => void;
}

/**
 * Alert icon component for certificate change banner
 */
const AlertIcon = ({ size, color }: { size: number; color: string }) => (
  <Icon source="alert-circle" size={size} color={color} />
);

/**
 * CertificateChangeAlert
 *
 * Displays a prominent banner alert when a device's certificate has changed.
 * Provides actions to:
 * - View certificate details
 * - Dismiss the alert (but still blocks connection until user decides)
 */
export function CertificateChangeAlert({
  visible,
  onViewDetails,
  onDismiss,
}: CertificateChangeAlertProps) {
  const theme = useTheme();

  if (!visible) {
    return null;
  }

  return (
    <Banner
      visible={visible}
      icon={({ size }) => <AlertIcon size={size} color={theme.colors.error} />}
      actions={[
        {
          label: 'View Details',
          onPress: onViewDetails,
        },
        {
          label: 'Dismiss',
          onPress: onDismiss,
        },
      ]}
      style={[styles.banner, { backgroundColor: theme.colors.errorContainer }]}
    >
      <View style={styles.content}>
        <View style={styles.titleRow}>
          <Icon source="shield-alert" size={20} color={theme.colors.error} />
          <Text variant="titleMedium" style={styles.titleText}>
            Security Alert: Certificate Changed
          </Text>
        </View>
        <Text variant="bodyMedium" style={styles.messageText}>
          This device&apos;s certificate has changed since your last connection. This could indicate
          a security issue or device reconfiguration. Verify the new certificate before proceeding.
        </Text>
      </View>
    </Banner>
  );
}

const styles = StyleSheet.create({
  banner: {
    marginVertical: 8,
    elevation: 4,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.25,
    shadowRadius: 3.84,
  },
  content: {
    flex: 1,
  },
  titleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 8,
  },
  titleText: {
    fontWeight: 'bold',
    fontSize: 16,
  },
  messageText: {
    fontSize: 14,
    lineHeight: 20,
  },
});
