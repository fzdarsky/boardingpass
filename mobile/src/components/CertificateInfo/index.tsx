/**
 * Certificate Info Display Component
 *
 * Displays detailed certificate information including issuer, subject,
 * validity period, fingerprint, and trust status.
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Divider, useTheme, Chip } from 'react-native-paper';
import { CertificateInfo } from '../../types/certificate';
import { CertificateStatusIndicator } from './StatusIndicator';
import { formatFingerprint, truncateFingerprint } from '../../utils/crypto';
import {
  extractCommonName,
  formatValidityPeriod,
  getDaysUntilExpiration,
} from '../../services/certificates/utils';

export interface CertificateInfoProps {
  certificate: CertificateInfo;
  compact?: boolean;
}

/**
 * CertificateInfo Component
 *
 * Displays certificate details in a card format with:
 * - Trust status indicator
 * - Common name and issuer
 * - Validity period with expiration warning
 * - Certificate fingerprint
 * - Certificate type (self-signed vs CA-signed)
 */
export function CertificateInfoDisplay({ certificate, compact = false }: CertificateInfoProps) {
  const theme = useTheme();

  const commonName = extractCommonName(certificate.subject);
  const issuerName = extractCommonName(certificate.issuer);
  const validityPeriod = formatValidityPeriod(certificate);
  const daysUntilExpiry = getDaysUntilExpiration(certificate);
  const isExpiringSoon = daysUntilExpiry >= 0 && daysUntilExpiry <= 30;
  const isExpired = daysUntilExpiry < 0;

  return (
    <Card style={styles.card}>
      <Card.Content>
        {/* Status Indicator */}
        <View style={styles.statusRow}>
          <CertificateStatusIndicator status={certificate.trustStatus} showLabel size="medium" />
        </View>

        <Divider style={styles.divider} />

        {/* Certificate Type */}
        <View style={styles.row}>
          <Text variant="labelMedium" style={styles.label}>
            Type:
          </Text>
          <Chip mode="outlined" compact style={styles.chip} textStyle={styles.chipText}>
            {certificate.isSelfSigned ? 'Self-Signed' : 'CA-Signed'}
          </Chip>
        </View>

        {/* Common Name */}
        {!compact && (
          <View style={styles.row}>
            <Text variant="labelMedium" style={styles.label}>
              Name:
            </Text>
            <Text variant="bodyMedium" style={styles.value}>
              {commonName}
            </Text>
          </View>
        )}

        {/* Issuer */}
        <View style={styles.row}>
          <Text variant="labelMedium" style={styles.label}>
            Issuer:
          </Text>
          <Text variant="bodyMedium" style={styles.value}>
            {issuerName}
          </Text>
        </View>

        {/* Validity Period */}
        <View style={styles.row}>
          <Text variant="labelMedium" style={styles.label}>
            Valid:
          </Text>
          <View style={styles.validityContainer}>
            <Text variant="bodyMedium" style={styles.value}>
              {validityPeriod}
            </Text>
            {isExpired && (
              <Chip
                mode="flat"
                compact
                style={[styles.chip, { backgroundColor: theme.colors.errorContainer }]}
                textStyle={[styles.chipText, { color: theme.colors.error }]}
              >
                Expired
              </Chip>
            )}
            {!isExpired && isExpiringSoon && (
              <Chip
                mode="flat"
                compact
                style={[styles.chip, { backgroundColor: theme.colors.secondaryContainer }]}
                textStyle={styles.chipText}
              >
                Expires in {daysUntilExpiry} days
              </Chip>
            )}
          </View>
        </View>

        {/* Fingerprint */}
        {!compact && (
          <>
            <Divider style={styles.divider} />
            <View style={styles.column}>
              <Text variant="labelMedium" style={styles.label}>
                Fingerprint (SHA-256):
              </Text>
              <Text variant="bodySmall" style={styles.fingerprint} selectable>
                {formatFingerprint(certificate.fingerprint)}
              </Text>
            </View>
          </>
        )}

        {compact && (
          <View style={styles.row}>
            <Text variant="labelMedium" style={styles.label}>
              Fingerprint:
            </Text>
            <Text variant="bodySmall" style={styles.valueMonospace} selectable>
              {truncateFingerprint(certificate.fingerprint, 4)}
            </Text>
          </View>
        )}

        {/* Pinned Status */}
        {!compact && certificate.pinnedAt && (
          <>
            <Divider style={styles.divider} />
            <View style={styles.row}>
              <Text variant="labelMedium" style={styles.label}>
                Pinned:
              </Text>
              <Text variant="bodyMedium" style={styles.value}>
                {certificate.pinnedAt.toLocaleDateString()}
              </Text>
            </View>
          </>
        )}
      </Card.Content>
    </Card>
  );
}

const styles = StyleSheet.create({
  card: {
    marginVertical: 8,
  },
  statusRow: {
    marginBottom: 12,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    marginVertical: 6,
    gap: 8,
  },
  column: {
    marginVertical: 6,
  },
  label: {
    minWidth: 80,
    fontWeight: '600',
  },
  value: {
    flex: 1,
  },
  validityContainer: {
    flex: 1,
    flexDirection: 'column',
    gap: 4,
  },
  chip: {
    alignSelf: 'flex-start',
  },
  chipText: {
    fontSize: 12,
  },
  fingerprint: {
    marginTop: 4,
    lineHeight: 20,
    fontFamily: 'monospace',
  },
  valueMonospace: {
    flex: 1,
    fontFamily: 'monospace',
  },
  divider: {
    marginVertical: 12,
  },
});
