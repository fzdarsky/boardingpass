/**
 * TPMInfo Component
 *
 * Displays TPM (Trusted Platform Module) information including presence,
 * manufacturer, model, and version.
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
 * Shows presence status, and optionally manufacturer, model, and version.
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
        left={props => (
          <Icon
            {...props}
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

            {tpmInfo.version && (
              <View style={styles.row}>
                <Text variant="titleSmall" style={styles.label}>
                  Version
                </Text>
                <Text variant="bodyLarge" style={styles.value}>
                  TPM {tpmInfo.version}
                </Text>
              </View>
            )}
          </>
        )}

        {/* No TPM Message */}
        {!hasTPM && (
          <Text variant="bodyMedium" style={styles.noTPMText}>
            This device does not have a Trusted Platform Module. Hardware-based security
            features may be limited.
          </Text>
        )}
      </Card.Content>
    </Card>
  );
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
