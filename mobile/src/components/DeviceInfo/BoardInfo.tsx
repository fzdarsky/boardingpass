/**
 * BoardInfo Component
 *
 * Displays board/hardware information including manufacturer, model, and serial number.
 * Formats serial numbers for readability per FR-018 and T089.
 *
 * Related: T087 - Create BoardInfo display component
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Icon, useTheme } from 'react-native-paper';
import type { BoardInfo as BoardInfoType } from '../../services/api/info';
import { spacing } from '../../theme';

export interface BoardInfoProps {
  /** Board information data from SystemInfo */
  boardInfo: BoardInfoType;

  /** Whether to format serial number for readability (default: true) */
  formatSerial?: boolean;
}

/**
 * BoardInfo Display Component
 *
 * Renders board information in a Material Design card.
 * Shows manufacturer, model, and serial number with proper formatting.
 */
export function BoardInfo({ boardInfo, formatSerial = true }: BoardInfoProps) {
  const theme = useTheme();

  // Format serial number for readability (T089)
  const formattedSerial = formatSerial ? formatSerialNumber(boardInfo.serial) : boardInfo.serial;

  return (
    <Card style={styles.card}>
      <Card.Title
        title="Board Information"
        titleVariant="titleLarge"
        subtitle="Hardware details"
        left={props => <Icon {...props} source="chip" size={24} color={theme.colors.primary} />}
      />
      <Card.Content>
        {/* Manufacturer */}
        <View style={styles.row}>
          <Text variant="titleSmall" style={styles.label}>
            Manufacturer
          </Text>
          <Text variant="bodyLarge" style={styles.value}>
            {boardInfo.manufacturer}
          </Text>
        </View>

        {/* Model */}
        <View style={styles.row}>
          <Text variant="titleSmall" style={styles.label}>
            Model
          </Text>
          <Text variant="bodyLarge" style={styles.value}>
            {boardInfo.model}
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
