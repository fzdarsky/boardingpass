/**
 * SystemInformationCard Component
 *
 * Consolidated single-card display of all device information:
 * Operating System, CPU, TPM, Board, and Network Interfaces.
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Badge, Divider, useTheme } from 'react-native-paper';
import type { SystemInfo } from '../../services/api/info';
import { getArchitectureDisplayName, isFIPSEnabled } from '../../services/api/info';
import type {
  NetworkConfig as NetworkConfigType,
  NetworkInterface,
} from '../../services/api/network';
import {
  formatMACAddress,
  formatIPAddress,
  getInterfaceTypeHint,
  sortInterfacesByPriority,
  getIPv4Addresses,
  getIPv6Addresses,
} from '../../services/api/network';
import { spacing } from '../../theme';

export interface SystemInformationCardProps {
  systemInfo: SystemInfo;
  networkConfig?: NetworkConfigType | null;
  connectionHost?: string | null;
}

export function SystemInformationCard({
  systemInfo,
  networkConfig,
  connectionHost,
}: SystemInformationCardProps) {
  const theme = useTheme();
  const fipsEnabled = isFIPSEnabled(systemInfo);
  const hasTPM = systemInfo.tpm.present;

  // Match connection host IP against interface addresses
  const serviceIfaceName = connectionHost
    ? (networkConfig?.interfaces.find(iface =>
        iface.ip_addresses.some(addr => addr.ip === connectionHost)
      )?.name ?? null)
    : null;

  const dynamicStyles = {
    label: { color: theme.colors.onSurfaceVariant },
    value: { color: theme.colors.onSurface },
    sectionHeader: { color: theme.colors.onSurface },
    fipsBadgeColor: {
      backgroundColor: fipsEnabled ? theme.colors.tertiary : theme.colors.surfaceVariant,
      color: fipsEnabled ? theme.colors.onTertiary : theme.colors.onSurfaceVariant,
    },
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
        {systemInfo.os.system_time && (
          <InfoRow
            label="System time"
            value={formatSystemTime(systemInfo.os.system_time)}
            labelStyle={dynamicStyles.label}
            valueStyle={dynamicStyles.value}
          />
        )}
        {systemInfo.os.clock_synchronized !== undefined && (
          <View style={styles.row}>
            <Text variant="bodySmall" style={[styles.label, dynamicStyles.label]}>
              Clock sync
            </Text>
            <Badge
              style={[
                styles.fipsBadge,
                {
                  backgroundColor: systemInfo.os.clock_synchronized
                    ? theme.colors.tertiary
                    : theme.colors.surfaceVariant,
                  color: systemInfo.os.clock_synchronized
                    ? theme.colors.onTertiary
                    : theme.colors.onSurfaceVariant,
                },
              ]}
            >
              {systemInfo.os.clock_synchronized ? 'SYNCED' : 'NOT SYNCED'}
            </Badge>
          </View>
        )}

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
            {(systemInfo.tpm.type || systemInfo.tpm.spec_version) && (
              <InfoRow
                label="Type"
                value={formatTPMType(systemInfo.tpm.type, systemInfo.tpm.spec_version)}
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

        {/* Firmware */}
        <Text variant="titleSmall" style={[styles.sectionHeader, dynamicStyles.sectionHeader]}>
          Firmware:
        </Text>
        <InfoRow
          label="Vendor"
          value={systemInfo.firmware.vendor}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
        />
        <InfoRow
          label="Version"
          value={systemInfo.firmware.version}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
        />
        <InfoRow
          label="Date"
          value={systemInfo.firmware.date}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
        />

        <Divider style={styles.sectionDivider} />

        {/* Product */}
        <Text variant="titleSmall" style={[styles.sectionHeader, dynamicStyles.sectionHeader]}>
          Product:
        </Text>
        <InfoRow
          label="Vendor"
          value={systemInfo.product.vendor}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
        />
        <InfoRow
          label="Name"
          value={systemInfo.product.name}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
        />
        <InfoRow
          label="Serial"
          value={formatSerialNumber(systemInfo.product.serial)}
          labelStyle={dynamicStyles.label}
          valueStyle={dynamicStyles.value}
          monospace
        />

        {/* Network Interfaces */}
        {networkConfig && networkConfig.interfaces.length > 0 && (
          <>
            <Divider style={styles.sectionDivider} />
            <Text variant="titleSmall" style={[styles.sectionHeader, dynamicStyles.sectionHeader]}>
              Network Interfaces ({networkConfig.interfaces.length}):
            </Text>
            {sortInterfacesByPriority(networkConfig.interfaces).map((iface, idx) => (
              <View key={`iface-${idx}-${iface.name}`}>
                {idx > 0 && <Divider style={styles.ifaceDivider} />}
                <CompactInterface iface={iface} isServiceIface={iface.name === serviceIfaceName} />
              </View>
            ))}
            {serviceIfaceName && (
              <Text variant="bodySmall" style={styles.footnote}>
                <Text style={{ color: theme.colors.error }}>{'* '}</Text>= interface currently used
                for provisioning
              </Text>
            )}
          </>
        )}
      </Card.Content>
    </Card>
  );
}

/**
 * Compact network interface display with status badge, MAC, and IPs
 */
function CompactInterface({
  iface,
  isServiceIface,
}: {
  iface: NetworkInterface;
  isServiceIface: boolean;
}) {
  const theme = useTheme();
  const isUp = iface.carrier;
  const typeHint = getInterfaceTypeHint(iface.name);
  const ipv4Addresses = getIPv4Addresses(iface);
  const ipv6Addresses = getIPv6Addresses(iface);

  return (
    <View style={styles.ifaceContainer}>
      <View style={styles.ifaceHeader}>
        <View
          style={[
            styles.statusDot,
            { backgroundColor: isUp ? theme.colors.tertiary : theme.colors.surfaceVariant },
          ]}
        />
        <Text variant="bodyMedium" style={[styles.ifaceName, { color: theme.colors.onSurface }]}>
          {iface.name}
          {isServiceIface && <Text style={{ color: theme.colors.error }}>{' *'}</Text>}
        </Text>
        <Text variant="bodySmall" style={{ color: theme.colors.onSurfaceVariant }}>
          ({typeHint})
        </Text>
        <Badge
          style={[
            styles.statusBadge,
            {
              backgroundColor: isUp ? theme.colors.tertiary : theme.colors.surfaceVariant,
              color: isUp ? theme.colors.onTertiary : theme.colors.onSurfaceVariant,
            },
          ]}
        >
          {isUp ? 'UP' : 'DOWN'}
        </Badge>
      </View>
      <View style={styles.ifaceDetails}>
        <Text variant="bodySmall" style={[styles.mono, { color: theme.colors.onSurfaceVariant }]}>
          {formatMACAddress(iface.mac_address)}
        </Text>
        {ipv4Addresses.map(addr => (
          <Text
            key={`v4-${addr.ip}-${addr.prefix}`}
            variant="bodySmall"
            style={[styles.mono, { color: theme.colors.onSurface }]}
          >
            {formatIPAddress(addr)}
          </Text>
        ))}
        {ipv6Addresses.map(addr => (
          <Text
            key={`v6-${addr.ip}-${addr.prefix}`}
            variant="bodySmall"
            style={[styles.mono, { color: theme.colors.onSurface }]}
          >
            {formatIPAddress(addr)}
          </Text>
        ))}
      </View>
    </View>
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
 * Format TPM type and spec version into a combined string, e.g. "Discrete TPM 2.0"
 */
function formatTPMType(type?: string | null, specVersion?: string | null): string {
  const typeNames: Record<string, string> = {
    discrete: 'Discrete TPM',
    firmware: 'Firmware TPM',
    virtual: 'Virtual TPM',
  };
  const typePart = type ? typeNames[type] || type : 'TPM';
  return specVersion ? `${typePart} ${specVersion}` : typePart;
}

/**
 * Format ISO 8601 system time to a human-readable locale string
 */
function formatSystemTime(isoTime: string): string {
  try {
    const date = new Date(isoTime);
    return date.toLocaleString(undefined, {
      dateStyle: 'medium',
      timeStyle: 'medium',
    });
  } catch {
    return isoTime;
  }
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
    paddingHorizontal: 8,
  },
  notAvailable: {
    paddingLeft: spacing.sm,
    paddingVertical: 3,
    fontStyle: 'italic',
  },
  ifaceDivider: {
    marginVertical: spacing.xs,
    marginLeft: spacing.sm,
  },
  ifaceContainer: {
    marginVertical: spacing.xs,
    paddingLeft: spacing.sm,
  },
  ifaceHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
  },
  ifaceName: {
    fontWeight: '600',
  },
  statusBadge: {
    marginLeft: 'auto',
    fontSize: 10,
    paddingHorizontal: 8,
  },
  ifaceDetails: {
    paddingLeft: 14,
    marginTop: 2,
  },
  mono: {
    fontFamily: 'monospace',
    fontSize: 12,
    lineHeight: 18,
  },
  footnote: {
    marginTop: spacing.sm,
    paddingLeft: spacing.sm,
    fontStyle: 'italic',
    opacity: 0.7,
  },
});
