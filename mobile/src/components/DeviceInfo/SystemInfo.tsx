/**
 * SystemInfo Component
 *
 * Displays device system information including TPM, board, CPU, OS, and FIPS status.
 * Formats data for readability per FR-018 and T089.
 *
 * Related: T084 - Create SystemInfo display component
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Chip, Divider, useTheme } from 'react-native-paper';
import type { SystemInfo as SystemInfoType } from '../../services/api/info';
import { getArchitectureDisplayName, isFIPSEnabled } from '../../services/api/info';
import { spacing } from '../../theme';

export interface SystemInfoProps {
  /** System information data from /info endpoint */
  systemInfo: SystemInfoType;

  /** Whether to show FIPS status indicator (T094) */
  showFIPSIndicator?: boolean;
}

/**
 * SystemInfo Display Component
 *
 * Renders system information in a Material Design card with sections for
 * CPU, OS, and FIPS status. Includes FIPS status badge indicator.
 */
export function SystemInfo({ systemInfo, showFIPSIndicator = true }: SystemInfoProps) {
  const theme = useTheme();
  const fipsEnabled = isFIPSEnabled(systemInfo);

  return (
    <Card style={styles.card}>
      <Card.Title
        title="System Information"
        titleVariant="titleLarge"
        subtitle="Hardware and software details"
      />
      <Card.Content>
        {/* CPU Information */}
        <View style={styles.section}>
          <Text variant="titleMedium" style={styles.sectionTitle}>
            CPU Architecture
          </Text>
          <Text variant="bodyLarge" style={styles.value}>
            {getArchitectureDisplayName(systemInfo.cpu.architecture)}
          </Text>
        </View>

        <Divider style={styles.divider} />

        {/* Operating System Information */}
        <View style={styles.section}>
          <Text variant="titleMedium" style={styles.sectionTitle}>
            Operating System
          </Text>
          <Text variant="bodyLarge" style={styles.value}>
            {systemInfo.os.distribution}
          </Text>
          <Text variant="bodyMedium" style={[styles.value, styles.subValue]}>
            Version {systemInfo.os.version}
          </Text>
        </View>

        <Divider style={styles.divider} />

        {/* FIPS Mode Status */}
        {showFIPSIndicator && (
          <View style={styles.section}>
            <Text variant="titleMedium" style={styles.sectionTitle}>
              Security Compliance
            </Text>
            <View style={styles.chipContainer}>
              <Chip
                mode="flat"
                selected={fipsEnabled}
                selectedColor={fipsEnabled ? theme.colors.primary : theme.colors.surfaceVariant}
                style={[
                  styles.chip,
                  fipsEnabled && {
                    backgroundColor: theme.colors.primaryContainer,
                  },
                ]}
                icon={fipsEnabled ? 'shield-check' : 'shield-off'}
                textStyle={styles.chipText}
              >
                FIPS {fipsEnabled ? 'Enabled' : 'Disabled'}
              </Chip>
              {fipsEnabled && (
                <Text variant="bodySmall" style={[styles.value, styles.fipsNote]}>
                  Device is operating in FIPS 140-3 mode
                </Text>
              )}
            </View>
          </View>
        )}
      </Card.Content>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: {
    marginVertical: spacing.sm,
  },
  section: {
    marginVertical: spacing.sm,
  },
  sectionTitle: {
    marginBottom: spacing.xs,
    fontWeight: '600',
  },
  value: {
    marginTop: spacing.xs,
  },
  subValue: {
    opacity: 0.7,
  },
  divider: {
    marginVertical: spacing.md,
  },
  chipContainer: {
    marginTop: spacing.sm,
  },
  chip: {
    alignSelf: 'flex-start',
  },
  chipText: {
    fontWeight: '600',
  },
  fipsNote: {
    marginTop: spacing.sm,
    fontStyle: 'italic',
    opacity: 0.7,
  },
});
