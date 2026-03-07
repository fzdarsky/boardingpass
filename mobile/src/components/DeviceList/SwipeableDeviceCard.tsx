/**
 * SwipeableDeviceCard Component
 *
 * Wraps DeviceCard with swipe-to-delete functionality.
 * Uses react-native-gesture-handler's Swipeable for cross-platform support.
 */

import React, { useRef, useCallback } from 'react';
import { StyleSheet, View, Animated } from 'react-native';
import { Swipeable } from 'react-native-gesture-handler';
import { IconButton, Text, useTheme } from 'react-native-paper';
import { DeviceCard, DeviceCardProps } from './DeviceCard';
import { HapticFeedback } from '@/utils/haptics';

export interface SwipeableDeviceCardProps extends DeviceCardProps {
  onDelete?: () => void;
}

export function SwipeableDeviceCard({ onDelete, ...deviceCardProps }: SwipeableDeviceCardProps) {
  const theme = useTheme();
  const swipeableRef = useRef<Swipeable>(null);

  const handleDelete = useCallback(async () => {
    await HapticFeedback.warning();
    swipeableRef.current?.close();
    onDelete?.();
  }, [onDelete]);

  const renderRightActions = useCallback(
    (progress: Animated.AnimatedInterpolation<number>) => {
      const translateX = progress.interpolate({
        inputRange: [0, 1],
        outputRange: [80, 0],
      });

      return (
        <Animated.View
          style={[
            styles.deleteAction,
            { backgroundColor: theme.colors.error, transform: [{ translateX }] },
          ]}
        >
          <View style={styles.deleteContent}>
            <IconButton
              icon="delete"
              iconColor={theme.colors.onError}
              size={24}
              onPress={handleDelete}
            />
            <Text style={[styles.deleteText, { color: theme.colors.onError }]}>Delete</Text>
          </View>
        </Animated.View>
      );
    },
    [theme, handleDelete]
  );

  if (!onDelete) {
    return <DeviceCard {...deviceCardProps} />;
  }

  return (
    <Swipeable
      ref={swipeableRef}
      renderRightActions={renderRightActions}
      rightThreshold={40}
      overshootRight={false}
      friction={2}
    >
      <DeviceCard {...deviceCardProps} />
    </Swipeable>
  );
}

const styles = StyleSheet.create({
  deleteAction: {
    justifyContent: 'center',
    alignItems: 'center',
    width: 80,
    marginVertical: 8,
    marginRight: 16,
    borderRadius: 4,
  },
  deleteContent: {
    alignItems: 'center',
  },
  deleteText: {
    fontSize: 12,
    fontWeight: '500',
    marginTop: -8,
  },
});
