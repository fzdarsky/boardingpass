/**
 * Certificate Status Indicator Component
 *
 * Visual indicator showing certificate trust status with icon and color.
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Icon, Text, useTheme } from 'react-native-paper';
import {
  TrustStatus,
  getTrustStatusIndicator,
  getTrustStatusDescription,
} from '../../types/certificate';

export interface CertificateStatusIndicatorProps {
  status: TrustStatus;
  showLabel?: boolean;
  size?: 'small' | 'medium' | 'large';
}

/**
 * CertificateStatusIndicator
 *
 * Displays a visual indicator for certificate trust status:
 * - trusted_ca: Green checkmark (certificate from trusted CA)
 * - self_signed_trusted: Yellow shield (self-signed, user trusted)
 * - self_signed_new: Orange warning (new self-signed, needs trust)
 * - changed: Red alert (certificate changed, requires verification)
 */
export function CertificateStatusIndicator({
  status,
  showLabel = false,
  size = 'medium',
}: CertificateStatusIndicatorProps) {
  const theme = useTheme();
  const { icon, color } = getTrustStatusIndicator(status);
  const description = getTrustStatusDescription(status);

  // Size mapping
  const iconSize = size === 'small' ? 16 : size === 'medium' ? 24 : 32;
  const fontSize = size === 'small' ? 12 : size === 'medium' ? 14 : 16;

  return (
    <View style={styles.container}>
      <Icon source={icon} size={iconSize} color={color} />
      {showLabel && (
        <Text
          variant="bodySmall"
          style={[styles.label, { color: theme.colors.onSurface, fontSize }]}
        >
          {description}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  label: {
    flex: 1,
  },
});
