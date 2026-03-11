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
import { Device, DeviceStatus, DiscoveryMethod } from '@/types/device';
import { deviceStatusColors } from '@/theme';
import { CertificateStatusIndicator } from '../CertificateInfo/StatusIndicator';
import { CertificateInfoDisplay } from '../CertificateInfo';
import { deviceSelectionFeedback } from '@/utils/haptics';

export interface DeviceCardProps {
  device: Device;
  onPress?: () => void;
  showDuplicateIndicator?: boolean;
  showCertificateStatus?: boolean;
}

export function DeviceCard({
  device,
  onPress,
  showDuplicateIndicator = false,
  showCertificateStatus = true,
}: DeviceCardProps) {
  const theme = useTheme();
  const [showCertModal, setShowCertModal] = useState(false);
  const statusConfig = getStatusConfig(device.status);
  const transportInfo = getTransportInfo(device.discoveryMethod);
  const discoveryBadgeColors = {
    backgroundColor: transportInfo.isManual ? theme.colors.primary : theme.colors.tertiary,
    color: transportInfo.isManual ? theme.colors.onPrimary : theme.colors.onTertiary,
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
      <Card style={styles.card}>
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
                Transport:
              </Text>
              <Badge style={[styles.badge, discoveryBadgeColors]}>{transportInfo.label}</Badge>
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

            {device.status === 'enrolled' && device.enrolledAt && (
              <View style={styles.infoRow}>
                <Text variant="bodySmall" style={[styles.label, dynamicStyles.secondaryText]}>
                  Enrolled:
                </Text>
                <Text variant="bodySmall" style={[styles.value, dynamicStyles.primaryText]}>
                  {formatLastSeen(device.enrolledAt)}
                </Text>
              </View>
            )}
          </View>

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

        {/* Action Button — always visible, disabled for non-actionable states */}
        {onPress && (
          <Card.Actions>
            <Button
              icon={getActionButton(device.status).icon}
              mode="contained"
              onPress={handlePress}
              disabled={!getActionButton(device.status).enabled}
            >
              {getActionButton(device.status).label}
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
 * Get transport display info (icon and label) for a discovery method.
 */
function getTransportInfo(method: DiscoveryMethod): {
  label: string;
  isManual: boolean;
} {
  switch (method) {
    case 'wifi':
      return { label: 'WIFI', isManual: false };
    case 'bluetooth':
      return { label: 'BLE', isManual: false };
    case 'usb':
      return { label: 'USB', isManual: false };
    case 'mdns':
      return { label: 'MDNS', isManual: false };
    case 'scan':
      return { label: 'SCAN', isManual: false };
    case 'manual':
      return { label: 'MANUAL', isManual: true };
    default:
      return { label: String(method).toUpperCase(), isManual: false };
  }
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
    case 'unavailable':
      return { label: 'Unavailable', color: deviceStatusColors.unavailable };
    case 'authenticating':
      return { label: 'Authenticating...', color: deviceStatusColors.authenticating };
    case 'authenticated':
      return { label: 'Connected', color: deviceStatusColors.authenticated };
    case 'enrolled':
      return { label: 'Enrolled', color: deviceStatusColors.enrolled };
    case 'error':
      return { label: 'Error', color: deviceStatusColors.error };
    default:
      return { label: 'Unknown', color: deviceStatusColors.offline };
  }
}

/**
 * Get action button config for a given status.
 * Always returns a config; non-actionable states get a disabled "Connect" button.
 */
function getActionButton(status: DeviceStatus): { label: string; icon: string; enabled: boolean } {
  switch (status) {
    case 'online':
      return { label: 'Connect', icon: 'login', enabled: true };
    case 'authenticated':
      return { label: 'Details', icon: 'information-outline', enabled: true };
    case 'error':
      return { label: 'Retry', icon: 'refresh', enabled: true };
    default:
      return { label: 'Connect', icon: 'login', enabled: false };
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
