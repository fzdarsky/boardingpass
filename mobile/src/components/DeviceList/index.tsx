/**
 * DeviceList Component
 *
 * Displays list of discovered devices with:
 * - Device cards with status indicators
 * - Pull-to-refresh functionality
 * - Swipe-to-delete functionality
 * - Empty state ("no devices found")
 * - Scanning state (loading indicator)
 * - Auto-refresh on device changes
 * - Duplicate name handling (FR-006)
 */

import React, { useCallback } from 'react';
import { StyleSheet, FlatList, RefreshControl, View } from 'react-native';
import { Text, ActivityIndicator, Button, useTheme } from 'react-native-paper';
import { Device } from '@/types/device';
import { SwipeableDeviceCard } from './SwipeableDeviceCard';
import { SkeletonDeviceList } from '../Skeleton';

export interface DeviceListProps {
  devices: Device[];
  isScanning: boolean;
  onRefresh?: () => void;
  onDevicePress?: (device: Device) => void;
  onDeleteDevice?: (device: Device) => void;
  onStartScan?: () => void;
  onAddDevice?: () => void;
}

export function DeviceList({
  devices,
  isScanning,
  onRefresh,
  onDevicePress,
  onDeleteDevice,
  onStartScan,
  onAddDevice,
}: DeviceListProps) {
  const theme = useTheme();

  // Dynamic styles that depend on theme
  const dynamicStyles = {
    secondaryText: { color: theme.colors.onSurfaceVariant },
  };

  // Detect duplicate device names
  const deviceNamesCount = devices.reduce(
    (acc, device) => {
      acc[device.name] = (acc[device.name] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const hasDuplicateNames = Object.values(deviceNamesCount).some(count => count > 1);

  // Render item callback (memoized to avoid re-creation)
  const renderDeviceCard = useCallback(
    ({ item }: { item: Device }) => (
      <SwipeableDeviceCard
        device={item}
        onPress={() => onDevicePress?.(item)}
        onDelete={onDeleteDevice ? () => onDeleteDevice(item) : undefined}
        showDuplicateIndicator={hasDuplicateNames && deviceNamesCount[item.name] > 1}
      />
    ),
    [onDevicePress, onDeleteDevice, hasDuplicateNames, deviceNamesCount]
  );

  // Render empty state
  if (devices.length === 0 && !isScanning) {
    return (
      <View style={styles.emptyContainer}>
        <Text variant="titleLarge" style={styles.emptyTitle}>
          No Devices Found
        </Text>
        <Text variant="bodyMedium" style={[styles.emptyText, dynamicStyles.secondaryText]}>
          Make sure your device is on the same network
        </Text>
        <View style={styles.emptyHints}>
          <Text variant="bodySmall" style={[styles.emptyHint, dynamicStyles.secondaryText]}>
            • Device must be running BoardingPass service
          </Text>
          <Text variant="bodySmall" style={[styles.emptyHint, dynamicStyles.secondaryText]}>
            • mDNS must be enabled on your network
          </Text>
          <Text variant="bodySmall" style={[styles.emptyHint, dynamicStyles.secondaryText]}>
            • Check firewall settings
          </Text>
        </View>
        {onStartScan && (
          <Button mode="contained" onPress={onStartScan} icon="radar" style={styles.scanButton}>
            Scan Network
          </Button>
        )}
        {onAddDevice && (
          <Button mode="outlined" onPress={onAddDevice} icon="plus" style={styles.addButton}>
            Add Device
          </Button>
        )}
      </View>
    );
  }

  // Render scanning state with skeleton screens (T128)
  if (isScanning && devices.length === 0) {
    return (
      <View style={styles.container}>
        <View style={styles.scanningHeader}>
          <ActivityIndicator size="small" />
          <Text
            variant="bodyMedium"
            style={[styles.scanningHeaderText, dynamicStyles.secondaryText]}
          >
            Scanning for devices...
          </Text>
        </View>
        <SkeletonDeviceList count={3} />
      </View>
    );
  }

  // Render device list
  return (
    <FlatList
      data={devices}
      keyExtractor={item => item.id}
      renderItem={renderDeviceCard}
      contentContainerStyle={styles.listContent}
      refreshControl={
        onRefresh ? <RefreshControl refreshing={isScanning} onRefresh={onRefresh} /> : undefined
      }
      ListHeaderComponent={
        isScanning ? (
          <View style={styles.scanningHeader}>
            <ActivityIndicator size="small" />
            <Text
              variant="bodySmall"
              style={[styles.scanningHeaderText, dynamicStyles.secondaryText]}
            >
              Scanning...
            </Text>
          </View>
        ) : null
      }
      ListEmptyComponent={
        <View style={styles.emptyContainer}>
          <Text variant="bodyMedium" style={[styles.emptyText, dynamicStyles.secondaryText]}>
            No devices found
          </Text>
        </View>
      }
    />
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
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
    marginBottom: 16,
    textAlign: 'center',
  },
  emptyHints: {
    alignItems: 'flex-start',
    marginBottom: 24,
  },
  emptyHint: {
    marginTop: 4,
  },
  scanButton: {
    paddingVertical: 4,
    paddingHorizontal: 16,
  },
  addButton: {
    marginTop: 12,
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
  },
  scanningHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 12,
    gap: 8,
  },
  scanningHeaderText: {},
});
