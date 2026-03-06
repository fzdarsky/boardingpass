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

import React, { useState } from 'react';
import { StyleSheet, View, TouchableOpacity } from 'react-native';
import { Card, Text, Badge, Button, IconButton, Portal, Modal, useTheme } from 'react-native-paper';
import { Device, DeviceStatus } from '@/types/device';
import { deviceStatusColors } from '@/theme';
import { CertificateStatusIndicator } from '../CertificateInfo/StatusIndicator';
import { CertificateInfoDisplay } from '../CertificateInfo';
import { deviceSelectionFeedback } from '@/utils/haptics';

export interface DeviceCardProps {
  device: Device;
  onPress?: () => void;
  showDuplicateIndicator?: boolean;
  showCertificateStatus?: boolean;
  isAuthenticated?: boolean;
}

export function DeviceCard({
  device,
  onPress,
  showDuplicateIndicator = false,
  showCertificateStatus = true,
  isAuthenticated = false,
}: DeviceCardProps) {
  const theme = useTheme();
  const [showCertModal, setShowCertModal] = useState(false);
  const displayStatus =
    isAuthenticated && device.status === 'online' ? 'authenticated' : device.status;
  const statusConfig = getStatusConfig(displayStatus);
  const isManualDiscovery = device.discoveryMethod === 'manual';
  const discoveryBadgeColors = {
    backgroundColor: isManualDiscovery ? theme.colors.primary : theme.colors.tertiary,
    color: isManualDiscovery ? theme.colors.onPrimary : theme.colors.onTertiary,
  };

  // Dynamic styles that depend on theme
  const dynamicStyles = {
    secondaryText: { color: theme.colors.onSurfaceVariant },
    primaryText: { color: theme.colors.onSurface },
    subtleBackground: { backgroundColor: theme.colors.surfaceVariant },
    modalBackground: { backgroundColor: theme.colors.surface },
  };

  // Wrap onPress with haptic feedback (T129)
  const handlePress = async () => {
    if (onPress) {
      await deviceSelectionFeedback();
      onPress();
    }
  };

  return (
    <>
      <Card style={styles.card} onPress={handlePress}>
        <Card.Content>
          {/* Header: Name and Status */}
          <View style={styles.header}>
            <View style={styles.titleContainer}>
              <Text variant="titleMedium" style={styles.name}>
                {device.name}
              </Text>
              {showDuplicateIndicator && (
                <Text variant="bodySmall" style={[styles.ipAddress, dynamicStyles.secondaryText]}>
                  {device.host}
                </Text>
              )}
            </View>
            <View style={styles.statusContainer}>
              <View style={[styles.statusIndicator, { backgroundColor: statusConfig.color }]} />
              <Text variant="bodySmall" style={[styles.statusText, dynamicStyles.secondaryText]}>
                {statusConfig.label}
              </Text>
            </View>
          </View>

          {/* Certificate Status (FR-032, FR-033) */}
          {showCertificateStatus && device.certificateInfo && (
            <View style={[styles.certificateRow, dynamicStyles.subtleBackground]}>
              <CertificateStatusIndicator
                status={device.certificateInfo.trustStatus}
                showLabel={false}
                size="small"
              />
              <Text
                variant="bodySmall"
                style={[styles.certificateLabel, dynamicStyles.secondaryText]}
              >
                {device.certificateInfo.isSelfSigned ? 'Self-Signed' : 'CA-Signed'}
              </Text>
              <TouchableOpacity
                onPress={() => setShowCertModal(true)}
                style={styles.certificateInfoButton}
              >
                <IconButton
                  icon="information-outline"
                  size={16}
                  onPress={() => setShowCertModal(true)}
                />
              </TouchableOpacity>
            </View>
          )}

          {/* Device Info */}
          <View style={styles.info}>
            <View style={styles.infoRow}>
              <Text variant="bodySmall" style={[styles.label, dynamicStyles.secondaryText]}>
                Address:
              </Text>
              <Text variant="bodySmall" style={[styles.value, dynamicStyles.primaryText]}>
                {device.host}:{device.port}
              </Text>
            </View>

            <View style={styles.infoRow}>
              <Text variant="bodySmall" style={[styles.label, dynamicStyles.secondaryText]}>
                Discovery:
              </Text>
              <Badge style={[styles.badge, discoveryBadgeColors]}>
                {device.discoveryMethod.toUpperCase()}
              </Badge>
            </View>

            {device.addresses.length > 1 && (
              <View style={styles.infoRow}>
                <Text variant="bodySmall" style={[styles.label, dynamicStyles.secondaryText]}>
                  Addresses:
                </Text>
                <Text variant="bodySmall" style={[styles.value, dynamicStyles.primaryText]}>
                  {device.addresses.length} interfaces
                </Text>
              </View>
            )}

            <View style={styles.infoRow}>
              <Text variant="bodySmall" style={[styles.label, dynamicStyles.secondaryText]}>
                Last seen:
              </Text>
              <Text variant="bodySmall" style={[styles.value, dynamicStyles.primaryText]}>
                {formatLastSeen(device.lastSeen)}
              </Text>
            </View>
          </View>

          {/* TXT Records (if available) */}
          {device.txt && Object.keys(device.txt).length > 0 && (
            <View style={[styles.txtRecords, dynamicStyles.subtleBackground]}>
              {Object.entries(device.txt).map(([key, value]) => (
                <Text
                  key={key}
                  variant="bodySmall"
                  style={[styles.txtRecord, dynamicStyles.secondaryText]}
                >
                  {key}: {value}
                </Text>
              ))}
            </View>
          )}
        </Card.Content>

        {/* Action Button */}
        {onPress && (device.status === 'online' || device.status === 'authenticated') && (
          <Card.Actions>
            <Button
              icon={isAuthenticated ? 'information-outline' : 'login'}
              mode="contained"
              onPress={onPress}
            >
              {isAuthenticated ? 'Details' : 'Connect'}
            </Button>
          </Card.Actions>
        )}
      </Card>

      {/* Certificate Info Modal (FR-033) */}
      {device.certificateInfo && (
        <Portal>
          <Modal
            visible={showCertModal}
            onDismiss={() => setShowCertModal(false)}
            contentContainerStyle={[styles.modal, dynamicStyles.modalBackground]}
          >
            <CertificateInfoDisplay certificate={device.certificateInfo} compact={false} />
            <View style={styles.modalActions}>
              <IconButton icon="close" mode="contained" onPress={() => setShowCertModal(false)} />
            </View>
          </Modal>
        </Portal>
      )}
    </>
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
      return { label: 'Online', color: deviceStatusColors.online };
    case 'offline':
      return { label: 'Offline', color: deviceStatusColors.offline };
    case 'authenticating':
      return { label: 'Authenticating...', color: deviceStatusColors.authenticating };
    case 'authenticated':
      return { label: 'Authenticated', color: deviceStatusColors.authenticated };
    case 'error':
      return { label: 'Error', color: deviceStatusColors.error };
    default:
      return { label: 'Unknown', color: deviceStatusColors.offline };
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
    borderRadius: 4,
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
  statusText: {},
  info: {
    gap: 6,
  },
  infoRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  label: {
    minWidth: 80,
  },
  value: {
    flex: 1,
  },
  badge: {
    fontSize: 10,
    paddingHorizontal: 8,
  },
  txtRecords: {
    marginTop: 12,
    padding: 8,
    borderRadius: 4,
  },
  txtRecord: {
    fontFamily: 'monospace',
  },
  certificateRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: 8,
    marginBottom: 4,
    paddingVertical: 4,
    paddingHorizontal: 8,
    borderRadius: 4,
    gap: 8,
  },
  certificateLabel: {
    flex: 1,
  },
  certificateInfoButton: {
    marginLeft: 'auto',
  },
  modal: {
    padding: 20,
    margin: 20,
    borderRadius: 8,
  },
  modalActions: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    marginTop: 16,
  },
});
