/**
 * TPMInfo Component
 *
 * Displays TPM (Trusted Platform Module) information including presence,
 * type, spec version, manufacturer, and model.
 *
 * Related: T086 - Create TPMInfo display component
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Icon, useTheme } from 'react-native-paper';
import type { TPMInfo as TPMInfoType } from '../../services/api/info';
import { spacing } from '../../theme';

export interface TPMInfoProps {
  /** TPM information data from SystemInfo */
  tpmInfo: TPMInfoType;
}

/**
 * TPMInfo Display Component
 *
 * Renders TPM information in a Material Design card.
 * Shows presence status, and optionally type, spec version, manufacturer, and model.
 */
export function TPMInfo({ tpmInfo }: TPMInfoProps) {
  const theme = useTheme();
  const hasTPM = tpmInfo.present;

  return (
    <Card style={styles.card}>
      <Card.Title
        title="TPM Information"
        titleVariant="titleLarge"
        subtitle="Trusted Platform Module"
        // eslint-disable-next-line react/no-unstable-nested-components
        left={() => (
          <Icon
            source={hasTPM ? 'shield-check' : 'shield-off'}
            size={24}
            color={hasTPM ? theme.colors.primary : theme.colors.onSurfaceVariant}
          />
        )}
      />
      <Card.Content>
        {/* TPM Present Status */}
        <View style={styles.row}>
          <Text variant="titleSmall" style={styles.label}>
            Status
          </Text>
          <Text
            variant="bodyLarge"
            style={[
              styles.value,
              { color: hasTPM ? theme.colors.primary : theme.colors.onSurfaceVariant },
            ]}
          >
            {hasTPM ? 'Present' : 'Not Present'}
          </Text>
        </View>

        {/* TPM Details (only if present) */}
        {hasTPM && (
          <>
            {(tpmInfo.type || tpmInfo.spec_version) && (
              <View style={styles.row}>
                <Text variant="titleSmall" style={styles.label}>
                  Type
                </Text>
                <Text variant="bodyLarge" style={styles.value}>
                  {formatTPMType(tpmInfo.type, tpmInfo.spec_version)}
                </Text>
              </View>
            )}

            {tpmInfo.manufacturer && (
              <View style={styles.row}>
                <Text variant="titleSmall" style={styles.label}>
                  Manufacturer
                </Text>
                <Text variant="bodyLarge" style={styles.value}>
                  {tpmInfo.manufacturer}
                </Text>
              </View>
            )}

            {tpmInfo.model && (
              <View style={styles.row}>
                <Text variant="titleSmall" style={styles.label}>
                  Model
                </Text>
                <Text variant="bodyLarge" style={styles.value}>
                  {tpmInfo.model}
                </Text>
              </View>
            )}
          </>
        )}

        {/* No TPM Message */}
        {!hasTPM && (
          <Text variant="bodyMedium" style={styles.noTPMText}>
            This device does not have a Trusted Platform Module. Hardware-based security features
            may be limited.
          </Text>
        )}
      </Card.Content>
    </Card>
  );
}

function formatTPMType(type?: string | null, specVersion?: string | null): string {
  const typeNames: Record<string, string> = {
    discrete: 'Discrete TPM',
    firmware: 'Firmware TPM',
    virtual: 'Virtual TPM',
  };
  const typePart = type ? (typeNames[type] || type) : 'TPM';
  return specVersion ? `${typePart} ${specVersion}` : typePart;
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
  noTPMText: {
    marginTop: spacing.sm,
    opacity: 0.7,
    fontStyle: 'italic',
  },
});
