/**
 * Skeleton Loading Components
 *
 * Reusable skeleton loading components for better UX during data fetching.
 * Implements T128: Add loading states and skeleton screens to all data-fetching screens
 */

import React, { useEffect, useRef } from 'react';
import { StyleSheet, View, Animated } from 'react-native';
import { spacing } from '@/theme';

/**
 * Skeleton Base Props
 */
interface SkeletonBaseProps {
  width?: number | string;
  height?: number;
  borderRadius?: number;
  style?: any;
}

/**
 * Base Skeleton Component with shimmer animation
 */
export function SkeletonBase({
  width = '100%',
  height = 20,
  borderRadius = 4,
  style,
}: SkeletonBaseProps) {
  const animatedValue = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    const animation = Animated.loop(
      Animated.sequence([
        Animated.timing(animatedValue, {
          toValue: 1,
          duration: 1000,
          useNativeDriver: true,
        }),
        Animated.timing(animatedValue, {
          toValue: 0,
          duration: 1000,
          useNativeDriver: true,
        }),
      ])
    );

    animation.start();

    return () => animation.stop();
  }, [animatedValue]);

  const opacity = animatedValue.interpolate({
    inputRange: [0, 1],
    outputRange: [0.3, 0.7],
  });

  return (
    <Animated.View
      style={[
        styles.skeleton,
        {
          width,
          height,
          borderRadius,
          opacity,
        },
        style,
      ]}
    />
  );
}

/**
 * Device Card Skeleton for device list
 */
export function SkeletonDeviceCard() {
  return (
    <View style={styles.deviceCard}>
      <View style={styles.deviceCardContent}>
        {/* Icon placeholder */}
        <SkeletonBase width={48} height={48} borderRadius={24} style={styles.deviceIcon} />

        {/* Text placeholders */}
        <View style={styles.deviceTextContainer}>
          <SkeletonBase width="70%" height={18} style={styles.deviceTitle} />
          <SkeletonBase width="50%" height={14} style={styles.deviceSubtitle} />
        </View>

        {/* Status indicator placeholder */}
        <SkeletonBase width={12} height={12} borderRadius={6} />
      </View>
    </View>
  );
}

/**
 * Device Info Section Skeleton
 */
export function SkeletonDeviceInfoSection() {
  return (
    <View style={styles.infoSection}>
      {/* Section title */}
      <SkeletonBase width="40%" height={20} style={styles.sectionTitle} />

      {/* Info rows */}
      {[1, 2, 3, 4].map(id => (
        <View key={`info-row-${id}`} style={styles.infoRow}>
          <SkeletonBase width="30%" height={14} style={styles.infoLabel} />
          <SkeletonBase width="60%" height={14} />
        </View>
      ))}
    </View>
  );
}

/**
 * Network Interface Card Skeleton
 */
export function SkeletonNetworkInterface() {
  return (
    <View style={styles.networkCard}>
      {/* Interface name and status */}
      <View style={styles.networkHeader}>
        <SkeletonBase width="40%" height={18} />
        <SkeletonBase width={60} height={24} borderRadius={12} />
      </View>

      {/* IP addresses */}
      <SkeletonBase width="80%" height={14} style={styles.networkAddress} />
      <SkeletonBase width="60%" height={14} style={styles.networkAddress} />

      {/* MAC address */}
      <SkeletonBase width="50%" height={12} style={styles.networkMac} />
    </View>
  );
}

/**
 * Full Device Detail Page Skeleton
 */
export function SkeletonDeviceDetail() {
  return (
    <View style={styles.detailContainer}>
      {/* System Info Section */}
      <SkeletonDeviceInfoSection />

      {/* Board Info Section */}
      <SkeletonDeviceInfoSection />

      {/* Network Interfaces */}
      <View style={styles.infoSection}>
        <SkeletonBase width="50%" height={20} style={styles.sectionTitle} />
        <SkeletonNetworkInterface />
        <SkeletonNetworkInterface />
      </View>
    </View>
  );
}

/**
 * Device List Skeleton (shows multiple cards)
 */
export function SkeletonDeviceList({ count = 3 }: { count?: number }) {
  return (
    <View style={styles.listContainer}>
      {Array.from({ length: count }).map((_, i) => (
        <SkeletonDeviceCard key={i} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  skeleton: {
    backgroundColor: '#E0E0E0',
  },

  // Device Card Skeleton
  deviceCard: {
    backgroundColor: '#fff',
    borderRadius: 8,
    padding: spacing.md,
    marginVertical: spacing.xs,
    marginHorizontal: spacing.md,
    elevation: 2,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
  },
  deviceCardContent: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  deviceIcon: {
    marginRight: spacing.md,
  },
  deviceTextContainer: {
    flex: 1,
  },
  deviceTitle: {
    marginBottom: spacing.xs,
  },
  deviceSubtitle: {
    marginBottom: 0,
  },

  // Info Section Skeleton
  infoSection: {
    backgroundColor: '#fff',
    borderRadius: 8,
    padding: spacing.md,
    marginBottom: spacing.md,
    elevation: 1,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
  },
  sectionTitle: {
    marginBottom: spacing.md,
  },
  infoRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: spacing.sm,
  },
  infoLabel: {
    marginRight: spacing.md,
  },

  // Network Interface Skeleton
  networkCard: {
    backgroundColor: '#F5F5F5',
    borderRadius: 8,
    padding: spacing.md,
    marginBottom: spacing.sm,
  },
  networkHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: spacing.sm,
  },
  networkAddress: {
    marginBottom: spacing.xs,
  },
  networkMac: {
    marginTop: spacing.xs,
  },

  // Container Styles
  detailContainer: {
    padding: spacing.md,
  },
  listContainer: {
    paddingTop: spacing.md,
  },
});
