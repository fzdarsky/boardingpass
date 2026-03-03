/**
 * NetworkConfig Component
 *
 * Compact display of network interface configuration.
 * Shows interfaces as a flat list with status, MAC, and IP addresses.
 * No Card wrapper — rendered directly under a section heading.
 *
 * Related: T085 - Create NetworkConfig display component
 */

import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, Divider, Badge, useTheme } from 'react-native-paper';
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
  getIPv4Addresses,
  getIPv6Addresses,
} from '../../services/api/network';
import { spacing } from '../../theme';

export interface NetworkConfigProps {
  networkConfig: NetworkConfigType;
  showStatusIndicators?: boolean;
}

export function NetworkConfig({ networkConfig, showStatusIndicators = true }: NetworkConfigProps) {
  const theme = useTheme();
  const sortedInterfaces = sortInterfacesByPriority(networkConfig.interfaces);
  const count = networkConfig.interfaces.length;

  return (
    <View style={styles.container}>
      <Text variant="titleMedium" style={[styles.heading, { color: theme.colors.onSurface }]}>
        Network Interfaces ({count})
      </Text>

      {sortedInterfaces.length === 0 && (
        <Text
          variant="bodySmall"
          style={[styles.emptyText, { color: theme.colors.onSurfaceVariant }]}
        >
          No network interfaces found
        </Text>
      )}

      {sortedInterfaces.map((iface, idx) => (
        <View key={`iface-${idx}-${iface.name}`}>
          {idx > 0 && <Divider style={styles.divider} />}
          <CompactInterface iface={iface} showStatus={showStatusIndicators} />
        </View>
      ))}
    </View>
  );
}

function CompactInterface({ iface, showStatus }: { iface: NetworkInterface; showStatus: boolean }) {
  const theme = useTheme();
  const isUp = isInterfaceUp(iface);
  const statusColor = isUp ? '#4CAF50' : '#F44336';
  const typeHint = getInterfaceTypeHint(iface.name);

  const ipv4Addresses = getIPv4Addresses(iface);
  const ipv6Addresses = getIPv6Addresses(iface);

  return (
    <View style={styles.ifaceContainer}>
      {/* Header: status dot + name (type) ... UP/DOWN badge */}
      <View style={styles.ifaceHeader}>
        {showStatus && <View style={[styles.statusDot, { backgroundColor: statusColor }]} />}
        <Text variant="bodyMedium" style={[styles.ifaceName, { color: theme.colors.onSurface }]}>
          {iface.name}
        </Text>
        <Text variant="bodySmall" style={{ color: theme.colors.onSurfaceVariant }}>
          ({typeHint})
        </Text>
        <Badge style={[styles.statusBadge, { backgroundColor: statusColor }]}>
          {iface.link_state.toUpperCase()}
        </Badge>
      </View>

      {/* Details: MAC and IPs as indented monospace lines */}
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

const styles = StyleSheet.create({
  container: {
    marginVertical: spacing.sm,
  },
  heading: {
    fontWeight: '600',
    marginBottom: spacing.sm,
  },
  emptyText: {
    fontStyle: 'italic',
    marginTop: spacing.xs,
  },
  divider: {
    marginVertical: spacing.sm,
  },
  ifaceContainer: {
    marginVertical: spacing.xs,
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
});
