/**
 * DeviceList Component
 *
 * Displays list of discovered devices with:
 * - Device cards with status indicators
 * - Pull-to-refresh functionality
 * - Empty state ("no devices found")
 * - Scanning state (loading indicator)
 * - Auto-refresh on device changes
 * - Duplicate name handling (FR-006)
 */

import React from 'react';
import { StyleSheet, FlatList, RefreshControl, View } from 'react-native';
import { Text, ActivityIndicator, Button } from 'react-native-paper';
import { Device } from '@/types/device';
import { DeviceCard } from './DeviceCard';

export interface DeviceListProps {
  devices: Device[];
  isScanning: boolean;
  onRefresh?: () => void;
  onDevicePress?: (device: Device) => void;
  onStartScan?: () => void;
}

export function DeviceList({
  devices,
  isScanning,
  onRefresh,
  onDevicePress,
  onStartScan,
}: DeviceListProps) {
  // Detect duplicate device names
  const deviceNamesCount = devices.reduce(
    (acc, device) => {
      acc[device.name] = (acc[device.name] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const hasDuplicateNames = Object.values(deviceNamesCount).some(count => count > 1);

  // Render empty state
  if (devices.length === 0 && !isScanning) {
    return (
      <View style={styles.emptyContainer}>
        <Text variant="titleLarge" style={styles.emptyTitle}>
          No Devices Found
        </Text>
        <Text variant="bodyMedium" style={styles.emptyText}>
          Make sure your device is on the same network
        </Text>
        <View style={styles.emptyHints}>
          <Text variant="bodySmall" style={styles.emptyHint}>
            • Device must be running BoardingPass service
          </Text>
          <Text variant="bodySmall" style={styles.emptyHint}>
            • mDNS must be enabled on your network
          </Text>
          <Text variant="bodySmall" style={styles.emptyHint}>
            • Check firewall settings
          </Text>
        </View>
        {onStartScan && (
          <Button mode="contained" onPress={onStartScan} icon="radar" style={styles.scanButton}>
            Start Scan
          </Button>
        )}
      </View>
    );
  }

  // Render scanning state
  if (isScanning && devices.length === 0) {
    return (
      <View style={styles.scanningContainer}>
        <ActivityIndicator size="large" />
        <Text variant="titleMedium" style={styles.scanningTitle}>
          Scanning for devices...
        </Text>
        <Text variant="bodySmall" style={styles.scanningText}>
          Looking for _boardingpass._tcp services
        </Text>
      </View>
    );
  }

  // Render device list
  return (
    <FlatList
      data={devices}
      keyExtractor={item => item.id}
      renderItem={({ item }) => (
        <DeviceCard
          device={item}
          onPress={() => onDevicePress?.(item)}
          showDuplicateIndicator={hasDuplicateNames && deviceNamesCount[item.name] > 1}
        />
      )}
      contentContainerStyle={styles.listContent}
      refreshControl={
        onRefresh ? <RefreshControl refreshing={isScanning} onRefresh={onRefresh} /> : undefined
      }
      ListHeaderComponent={
        isScanning ? (
          <View style={styles.scanningHeader}>
            <ActivityIndicator size="small" />
            <Text variant="bodySmall" style={styles.scanningHeaderText}>
              Scanning...
            </Text>
          </View>
        ) : null
      }
      ListEmptyComponent={
        <View style={styles.emptyContainer}>
          <Text variant="bodyMedium" style={styles.emptyText}>
            No devices found
          </Text>
        </View>
      }
    />
  );
}

const styles = StyleSheet.create({
  listContent: {
    paddingVertical: 8,
  },
  emptyContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
    minHeight: 400,
  },
  emptyTitle: {
    fontWeight: 'bold',
    marginBottom: 12,
    textAlign: 'center',
  },
  emptyText: {
    color: '#666',
    marginBottom: 16,
    textAlign: 'center',
  },
  emptyHints: {
    alignItems: 'flex-start',
    marginBottom: 24,
  },
  emptyHint: {
    color: '#999',
    marginTop: 4,
  },
  scanButton: {
    paddingVertical: 4,
    paddingHorizontal: 16,
  },
  scanningContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
    minHeight: 400,
  },
  scanningTitle: {
    marginTop: 16,
    fontWeight: '600',
  },
  scanningText: {
    marginTop: 4,
    color: '#666',
  },
  scanningHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 12,
    gap: 8,
  },
  scanningHeaderText: {
    color: '#666',
  },
});
