/**
 * DeviceCard Component
 *
 * Displays a single device in the device list with:
 * - Device name
 * - IP address (for duplicate names - FR-006)
 * - Status indicator (online, offline, authenticating, authenticated)
 * - Discovery method badge (mDNS, fallback, manual)
 * - Last seen timestamp
 */

import React from 'react';
import { StyleSheet, View } from 'react-native';
import { Card, Text, Badge, IconButton } from 'react-native-paper';
import { Device, DeviceStatus } from '@/types/device';

export interface DeviceCardProps {
  device: Device;
  onPress?: () => void;
  showDuplicateIndicator?: boolean;
}

export function DeviceCard({ device, onPress, showDuplicateIndicator = false }: DeviceCardProps) {
  const statusConfig = getStatusConfig(device.status);
  const discoveryBadgeColor = getDiscoveryMethodColor(device.discoveryMethod);

  return (
    <Card style={styles.card} onPress={onPress}>
      <Card.Content>
        {/* Header: Name and Status */}
        <View style={styles.header}>
          <View style={styles.titleContainer}>
            <Text variant="titleMedium" style={styles.name}>
              {device.name}
            </Text>
            {showDuplicateIndicator && (
              <Text variant="bodySmall" style={styles.ipAddress}>
                {device.host}
              </Text>
            )}
          </View>
          <View style={styles.statusContainer}>
            <View style={[styles.statusIndicator, { backgroundColor: statusConfig.color }]} />
            <Text variant="bodySmall" style={styles.statusText}>
              {statusConfig.label}
            </Text>
          </View>
        </View>

        {/* Device Info */}
        <View style={styles.info}>
          <View style={styles.infoRow}>
            <Text variant="bodySmall" style={styles.label}>
              Address:
            </Text>
            <Text variant="bodySmall" style={styles.value}>
              {device.host}:{device.port}
            </Text>
          </View>

          <View style={styles.infoRow}>
            <Text variant="bodySmall" style={styles.label}>
              Discovery:
            </Text>
            <Badge style={[styles.badge, { backgroundColor: discoveryBadgeColor }]}>
              {device.discoveryMethod.toUpperCase()}
            </Badge>
          </View>

          {device.addresses.length > 1 && (
            <View style={styles.infoRow}>
              <Text variant="bodySmall" style={styles.label}>
                Addresses:
              </Text>
              <Text variant="bodySmall" style={styles.value}>
                {device.addresses.length} interfaces
              </Text>
            </View>
          )}

          <View style={styles.infoRow}>
            <Text variant="bodySmall" style={styles.label}>
              Last seen:
            </Text>
            <Text variant="bodySmall" style={styles.value}>
              {formatLastSeen(device.lastSeen)}
            </Text>
          </View>
        </View>

        {/* TXT Records (if available) */}
        {device.txt && Object.keys(device.txt).length > 0 && (
          <View style={styles.txtRecords}>
            {Object.entries(device.txt).map(([key, value]) => (
              <Text key={key} variant="bodySmall" style={styles.txtRecord}>
                {key}: {value}
              </Text>
            ))}
          </View>
        )}
      </Card.Content>

      {/* Action Button */}
      {onPress && device.status === 'online' && (
        <Card.Actions>
          <IconButton icon="login" mode="contained" onPress={onPress} />
        </Card.Actions>
      )}
    </Card>
  );
}

/**
 * Get status indicator configuration
 */
function getStatusConfig(status: DeviceStatus): {
  label: string;
  color: string;
} {
  switch (status) {
    case 'online':
      return { label: 'Online', color: '#4caf50' };
    case 'offline':
      return { label: 'Offline', color: '#9e9e9e' };
    case 'authenticating':
      return { label: 'Authenticating...', color: '#ff9800' };
    case 'authenticated':
      return { label: 'Authenticated', color: '#2196f3' };
    case 'error':
      return { label: 'Error', color: '#f44336' };
    default:
      return { label: 'Unknown', color: '#9e9e9e' };
  }
}

/**
 * Get discovery method badge color
 */
function getDiscoveryMethodColor(method: string): string {
  switch (method) {
    case 'mdns':
      return '#2196f3'; // Blue
    case 'fallback':
      return '#ff9800'; // Orange
    case 'manual':
      return '#9c27b0'; // Purple
    default:
      return '#9e9e9e'; // Grey
  }
}

/**
 * Format last seen timestamp
 */
function formatLastSeen(date: Date): string {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const seconds = Math.floor(diff / 1000);

  if (seconds < 10) {
    return 'Just now';
  } else if (seconds < 60) {
    return `${seconds}s ago`;
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    return `${minutes}m ago`;
  } else {
    const hours = Math.floor(seconds / 3600);
    return `${hours}h ago`;
  }
}

const styles = StyleSheet.create({
  card: {
    marginVertical: 8,
    marginHorizontal: 16,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: 12,
  },
  titleContainer: {
    flex: 1,
  },
  name: {
    fontWeight: 'bold',
    marginBottom: 4,
  },
  ipAddress: {
    color: '#666',
    fontStyle: 'italic',
  },
  statusContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  statusIndicator: {
    width: 10,
    height: 10,
    borderRadius: 5,
  },
  statusText: {
    color: '#666',
  },
  info: {
    gap: 6,
  },
  infoRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  label: {
    color: '#666',
    minWidth: 80,
  },
  value: {
    flex: 1,
    color: '#333',
  },
  badge: {
    fontSize: 10,
  },
  txtRecords: {
    marginTop: 12,
    padding: 8,
    backgroundColor: '#f5f5f5',
    borderRadius: 4,
  },
  txtRecord: {
    color: '#666',
    fontFamily: 'monospace',
  },
});
