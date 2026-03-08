/**
 * ProductInfo Component
 *
 * Displays product identity information including vendor, name, and serial number.
 * Formats serial numbers for readability per FR-018 and T089.
 *
 * Related: T087 - Create ProductInfo display component
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Icon, useTheme } from 'react-native-paper';
import type { ProductInfo as ProductInfoType } from '../../services/api/info';
import { spacing } from '../../theme';

export interface ProductInfoProps {
  /** Product information data from SystemInfo */
  productInfo: ProductInfoType;

  /** Whether to format serial number for readability (default: true) */
  formatSerial?: boolean;
}

/**
 * ProductInfo Display Component
 *
 * Renders product information in a Material Design card.
 * Shows vendor, name, and serial number with proper formatting.
 */
export function ProductInfo({ productInfo, formatSerial = true }: ProductInfoProps) {
  const theme = useTheme();

  // Format serial number for readability (T089)
  const formattedSerial = formatSerial
    ? formatSerialNumber(productInfo.serial)
    : productInfo.serial;

  return (
    <Card style={styles.card}>
      <Card.Title
        title="Product Information"
        titleVariant="titleLarge"
        subtitle="Hardware details"
        // eslint-disable-next-line react/no-unstable-nested-components
        left={() => <Icon source="chip" size={24} color={theme.colors.primary} />}
      />
      <Card.Content>
        {/* Vendor */}
        <View style={styles.row}>
          <Text variant="titleSmall" style={styles.label}>
            Vendor
          </Text>
          <Text variant="bodyLarge" style={styles.value}>
            {productInfo.vendor}
          </Text>
        </View>

        {/* Name */}
        <View style={styles.row}>
          <Text variant="titleSmall" style={styles.label}>
            Name
          </Text>
          <Text variant="bodyLarge" style={styles.value}>
            {productInfo.name}
          </Text>
        </View>

        {/* Serial Number */}
        <View style={styles.row}>
          <Text variant="titleSmall" style={styles.label}>
            Serial Number
          </Text>
          <Text variant="bodyMedium" style={[styles.value, styles.monospace]}>
            {formattedSerial}
          </Text>
        </View>
      </Card.Content>
    </Card>
  );
}

/**
 * Format serial number for readability
 *
 * Adds spacing/grouping to long serial numbers for easier reading (T089).
 * Example: "10000000abcdef01" -> "1000-0000-abcd-ef01"
 *
 * @param serial - Raw serial number string
 * @returns Formatted serial number
 */
function formatSerialNumber(serial: string): string {
  // If serial is too short, return as-is
  if (serial.length <= 8) {
    return serial;
  }

  // Group in chunks of 4 characters for long serials
  const chunks: string[] = [];
  for (let i = 0; i < serial.length; i += 4) {
    chunks.push(serial.slice(i, i + 4));
  }

  return chunks.join('-');
}

const styles = StyleSheet.create({
  card: {
    marginVertical: spacing.sm,
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginVertical: spacing.sm,
  },
  label: {
    fontWeight: '600',
    opacity: 0.7,
  },
  value: {
    textAlign: 'right',
    flex: 1,
    marginLeft: spacing.md,
  },
  monospace: {
    fontFamily: 'monospace',
    fontSize: 12,
  },
});
