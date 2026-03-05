/**
 * SystemInformationCard Component
 *
 * Consolidated single-card display of all system information:
 * Operating System, CPU, TPM, and Board details.
 * Replaces the separate SystemInfo, TPMInfo, and BoardInfo cards
 * on the device details page for a more compact layout.
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Badge, Divider, useTheme } from 'react-native-paper';
import type { SystemInfo } from '../../services/api/info';
import { getArchitectureDisplayName, isFIPSEnabled } from '../../services/api/info';
import { spacing } from '../../theme';

export interface SystemInformationCardProps {
  systemInfo: SystemInfo;
}

export function SystemInformationCard({ systemInfo }: SystemInformationCardProps) {
  const theme = useTheme();
  const fipsEnabled = isFIPSEnabled(systemInfo);
  const hasTPM = systemInfo.tpm.present;

  const dynamicStyles = {
    label: { color: theme.colors.onSurfaceVariant },
    value: { color: theme.colors.onSurface },
    sectionHeader: { color: theme.colors.onSurface },
    fipsBadgeColor: { backgroundColor: fipsEnabled ? '#2D628B' : '#9e9e9e' },
  };

  return (
    <Card style={styles.card}>
      <Card.Title title="System Information" titleVariant="titleLarge" />
      <Card.Content>
        {/* Operating System */}
        <Text variant="titleSmall" style={[styles.sectionHeader, dynamicStyles.sectionHeader]}>
          Operating System:
        </Text>
        <InfoRow
          label="Distribution"
          value={`${systemInfo.os.distribution}`}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
        />
        <View style={styles.row}>
          <Text variant="bodySmall" style={[styles.label, dynamicStyles.label]}>
            FIPS mode
          </Text>
          <Badge style={[styles.fipsBadge, dynamicStyles.fipsBadgeColor]}>
            {fipsEnabled ? 'ENABLED' : 'DISABLED'}
          </Badge>
        </View>

        <Divider style={styles.sectionDivider} />

        {/* CPU */}
        <Text variant="titleSmall" style={[styles.sectionHeader, dynamicStyles.sectionHeader]}>
          CPU:
        </Text>
        <InfoRow
          label="Architecture"
          value={getArchitectureDisplayName(systemInfo.cpu.architecture)}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
        />

        <Divider style={styles.sectionDivider} />

        {/* TPM */}
        <Text variant="titleSmall" style={[styles.sectionHeader, dynamicStyles.sectionHeader]}>
          TPM:
        </Text>
        {hasTPM ? (
          <>
            {systemInfo.tpm.version && (
              <InfoRow
                label="Version"
                value={systemInfo.tpm.version}
                labelStyle={dynamicStyles.label}
                valueStyle={dynamicStyles.value}
              />
            )}
            {systemInfo.tpm.manufacturer && (
              <InfoRow
                label="Manufacturer"
                value={systemInfo.tpm.manufacturer}
                labelStyle={dynamicStyles.label}
                valueStyle={dynamicStyles.value}
              />
            )}
            {systemInfo.tpm.model && (
              <InfoRow
                label="Model"
                value={systemInfo.tpm.model}
                labelStyle={dynamicStyles.label}
                valueStyle={dynamicStyles.value}
              />
            )}
          </>
        ) : (
          <Text variant="bodySmall" style={[styles.notAvailable, dynamicStyles.label]}>
            Not available
          </Text>
        )}

        <Divider style={styles.sectionDivider} />

        {/* Board */}
        <Text variant="titleSmall" style={[styles.sectionHeader, dynamicStyles.sectionHeader]}>
          Board:
        </Text>
        <InfoRow
          label="Manufacturer"
          value={systemInfo.board.manufacturer}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
        />
        <InfoRow
          label="Model"
          value={systemInfo.board.model}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
        />
        <InfoRow
          label="Serial"
          value={formatSerialNumber(systemInfo.board.serial)}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
          monospace
        />
      </Card.Content>
    </Card>
  );
}

/**
 * Two-column label:value row
 */
function InfoRow({
  label,
  value,
  labelStyle,
  valueStyle,
  monospace = false,
}: {
  label: string;
  value: string;
  labelStyle: { color: string };
  valueStyle: { color: string };
  monospace?: boolean;
}) {
  return (
    <View style={styles.row}>
      <Text variant="bodySmall" style={[styles.label, labelStyle]}>
        {label}
      </Text>
      <Text variant="bodySmall" style={[styles.value, valueStyle, monospace && styles.monospace]}>
        {value}
      </Text>
    </View>
  );
}

/**
 * Format serial number for readability (groups of 4 with dashes)
 */
function formatSerialNumber(serial: string): string {
  if (serial.length <= 8) {
    return serial;
  }
  const chunks: string[] = [];
  for (let i = 0; i < serial.length; i += 4) {
    chunks.push(serial.slice(i, i + 4));
  }
  return chunks.join('-');
}

const styles = StyleSheet.create({
  card: {
    marginVertical: spacing.sm,
    borderRadius: 4,
  },
  sectionHeader: {
    fontWeight: 'bold',
    marginTop: spacing.sm,
    marginBottom: spacing.xs,
  },
  sectionDivider: {
    marginVertical: spacing.sm,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 3,
    paddingLeft: spacing.sm,
  },
  label: {
    minWidth: 100,
  },
  value: {
    flex: 1,
  },
  monospace: {
    fontFamily: 'monospace',
    fontSize: 12,
  },
  fipsBadge: {
    fontSize: 10,
  },
  notAvailable: {
    paddingLeft: spacing.sm,
    paddingVertical: 3,
    fontStyle: 'italic',
  },
});
