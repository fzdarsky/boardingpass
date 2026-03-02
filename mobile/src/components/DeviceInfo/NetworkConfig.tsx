/**
 * NetworkConfig Component
 *
 * Displays network interface configuration including interface names, MAC addresses,
 * link states, and IP addresses. Implements status indicators per T095.
 *
 * Related: T085 - Create NetworkConfig display component
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Card, Text, Chip, Divider, useTheme } from 'react-native-paper';
import type {
  NetworkConfig as NetworkConfigType,
  NetworkInterface,
} from '../../services/api/network';
import {
  isInterfaceUp,
  formatMACAddress,
  formatIPAddress,
  getInterfaceTypeHint,
  sortInterfacesByPriority,
  hasIPAddresses,
  getIPv4Addresses,
  getIPv6Addresses,
} from '../../services/api/network';
import { spacing } from '../../theme';

export interface NetworkConfigProps {
  /** Network configuration data from /network endpoint */
  networkConfig: NetworkConfigType;

  /** Whether to show interface status indicators (T095) */
  showStatusIndicators?: boolean;
}

/**
 * NetworkConfig Display Component
 *
 * Renders network interfaces in a Material Design card with expandable sections.
 * Shows link state indicators, MAC addresses, and IP addresses formatted for readability.
 */
export function NetworkConfig({ networkConfig, showStatusIndicators = true }: NetworkConfigProps) {
  // Sort interfaces by priority (physical first, loopback last, up before down)
  const sortedInterfaces = sortInterfacesByPriority(networkConfig.interfaces);

  return (
    <Card style={styles.card}>
      <Card.Title
        title="Network Configuration"
        titleVariant="titleLarge"
        subtitle={`${networkConfig.interfaces.length} interface${networkConfig.interfaces.length !== 1 ? 's' : ''}`}
      />
      <Card.Content>
        {sortedInterfaces.map((iface, idx) => (
          <View key={`iface-${idx}-${iface.name}`}>
            {idx > 0 && <Divider style={styles.divider} />}
            <NetworkInterfaceItem interface={iface} showStatusIndicator={showStatusIndicators} />
          </View>
        ))}

        {sortedInterfaces.length === 0 && (
          <Text variant="bodyMedium" style={styles.emptyText}>
            No network interfaces found
          </Text>
        )}
      </Card.Content>
    </Card>
  );
}

interface NetworkInterfaceItemProps {
  interface: NetworkInterface;
  showStatusIndicator: boolean;
}

/**
 * Individual Network Interface Item
 *
 * Displays a single network interface with status indicator, MAC address, and IP addresses.
 */
function NetworkInterfaceItem({
  interface: iface,
  showStatusIndicator,
}: NetworkInterfaceItemProps) {
  const theme = useTheme();
  const isUp = isInterfaceUp(iface);
  const statusColor = isUp ? '#4CAF50' : '#F44336'; // Green : Red (T095)
  const typeHint = getInterfaceTypeHint(iface.name);
  const hasIPs = hasIPAddresses(iface);

  const ipv4Addresses = getIPv4Addresses(iface);
  const ipv6Addresses = getIPv6Addresses(iface);

  return (
    <View style={styles.interfaceContainer}>
      {/* Interface header with status indicator */}
      <View style={styles.interfaceHeader}>
        {showStatusIndicator && (
          <View style={[styles.statusIndicator, { backgroundColor: statusColor }]} />
        )}
        <View style={styles.interfaceTitle}>
          <Text variant="titleMedium" style={styles.interfaceName}>
            {iface.name}
          </Text>
          <Text variant="bodySmall" style={styles.interfaceType}>
            {typeHint}
          </Text>
        </View>
        <Chip
          mode="outlined"
          compact
          style={[styles.statusChip, isUp && { borderColor: statusColor }]}
          textStyle={[
            styles.statusChipText,
            { color: isUp ? statusColor : theme.colors.onSurfaceVariant },
          ]}
        >
          {iface.link_state.toUpperCase()}
        </Chip>
      </View>

      {/* MAC Address */}
      <View style={styles.detailRow}>
        <Text variant="bodySmall" style={styles.label}>
          MAC Address
        </Text>
        <Text variant="bodyMedium" style={styles.monospace}>
          {formatMACAddress(iface.mac_address)}
        </Text>
      </View>

      {/* IP Addresses */}
      {hasIPs ? (
        <View style={styles.ipSection}>
          <Text variant="bodySmall" style={styles.label}>
            IP Addresses
          </Text>

          {/* IPv4 Addresses */}
          {ipv4Addresses.length > 0 && (
            <View style={styles.ipGroup}>
              <Text variant="bodySmall" style={styles.ipFamilyLabel}>
                IPv4
              </Text>
              {ipv4Addresses.map(addr => (
                <Text
                  key={`ipv4-${addr.ip}-${addr.prefix}`}
                  variant="bodyMedium"
                  style={styles.monospace}
                >
                  {formatIPAddress(addr)}
                </Text>
              ))}
            </View>
          )}

          {/* IPv6 Addresses */}
          {ipv6Addresses.length > 0 && (
            <View style={styles.ipGroup}>
              <Text variant="bodySmall" style={styles.ipFamilyLabel}>
                IPv6
              </Text>
              {ipv6Addresses.map(addr => (
                <Text
                  key={`ipv6-${addr.ip}-${addr.prefix}`}
                  variant="bodyMedium"
                  style={styles.monospace}
                >
                  {formatIPAddress(addr)}
                </Text>
              ))}
            </View>
          )}
        </View>
      ) : (
        <View style={styles.detailRow}>
          <Text variant="bodySmall" style={[styles.label, styles.noIPsText]}>
            No IP addresses assigned
          </Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    marginVertical: spacing.sm,
  },
  interfaceContainer: {
    marginVertical: spacing.sm,
  },
  interfaceHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: spacing.sm,
  },
  statusIndicator: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: spacing.sm,
  },
  interfaceTitle: {
    flex: 1,
  },
  interfaceName: {
    fontWeight: '600',
  },
  interfaceType: {
    marginTop: 2,
    opacity: 0.7,
  },
  statusChip: {
    height: 24,
  },
  statusChipText: {
    fontSize: 11,
    fontWeight: '600',
  },
  detailRow: {
    marginVertical: spacing.xs,
  },
  label: {
    opacity: 0.7,
    marginBottom: 2,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  monospace: {
    fontFamily: 'monospace',
    fontSize: 13,
  },
  ipSection: {
    marginTop: spacing.sm,
  },
  ipGroup: {
    marginTop: spacing.xs,
  },
  ipFamilyLabel: {
    opacity: 0.7,
    marginBottom: 2,
    fontWeight: '600',
  },
  noIPsText: {
    fontStyle: 'italic',
  },
  divider: {
    marginVertical: spacing.md,
  },
  emptyText: {
    textAlign: 'center',
    marginVertical: spacing.lg,
    opacity: 0.5,
  },
});
