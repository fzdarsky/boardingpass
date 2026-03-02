/**
 * Haptic Feedback Utilities
 *
 * Provides haptic feedback for button presses and important events.
 * Implements T129: Add haptic feedback for button presses and important events
 */

import * as Haptics from 'expo-haptics';

/**
 * Haptic Feedback Types
 */
export const HapticFeedback = {
  /**
   * Light impact - for button presses
   */
  light: async () => {
    try {
      await Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    } catch (error) {
      // Haptics may not be available on all devices
      if (__DEV__) {
        console.warn('[Haptics] Light feedback failed:', error);
      }
    }
  },

  /**
   * Medium impact - for important interactions (e.g., starting scan, refreshing)
   */
  medium: async () => {
    try {
      await Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    } catch (error) {
      if (__DEV__) {
        console.warn('[Haptics] Medium feedback failed:', error);
      }
    }
  },

  /**
   * Heavy impact - for critical actions (e.g., authentication, errors)
   */
  heavy: async () => {
    try {
      await Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Heavy);
    } catch (error) {
      if (__DEV__) {
        console.warn('[Haptics] Heavy feedback failed:', error);
      }
    }
  },

  /**
   * Success notification - for successful operations
   */
  success: async () => {
    try {
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    } catch (error) {
      if (__DEV__) {
        console.warn('[Haptics] Success feedback failed:', error);
      }
    }
  },

  /**
   * Warning notification - for warnings
   */
  warning: async () => {
    try {
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
    } catch (error) {
      if (__DEV__) {
        console.warn('[Haptics] Warning feedback failed:', error);
      }
    }
  },

  /**
   * Error notification - for errors
   */
  error: async () => {
    try {
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    } catch (error) {
      if (__DEV__) {
        console.warn('[Haptics] Error feedback failed:', error);
      }
    }
  },

  /**
   * Selection changed - for picking items
   */
  selection: async () => {
    try {
      await Haptics.selectionAsync();
    } catch (error) {
      if (__DEV__) {
        console.warn('[Haptics] Selection feedback failed:', error);
      }
    }
  },
};

/**
 * Haptic-enabled button press handler
 * Wraps a button onPress handler with haptic feedback
 */
export function withHapticFeedback<T extends unknown[]>(
  handler: (...args: T) => void | Promise<void>,
  feedbackType: keyof typeof HapticFeedback = 'light'
): (...args: T) => Promise<void> {
  return async (...args: T) => {
    await HapticFeedback[feedbackType]();
    await handler(...args);
  };
}

/**
 * Haptic feedback for device selection
 */
export const deviceSelectionFeedback = () => HapticFeedback.light();

/**
 * Haptic feedback for authentication start
 */
export const authenticationStartFeedback = () => HapticFeedback.medium();

/**
 * Haptic feedback for authentication success
 */
export const authenticationSuccessFeedback = () => HapticFeedback.success();

/**
 * Haptic feedback for authentication failure
 */
export const authenticationFailureFeedback = () => HapticFeedback.error();

/**
 * Haptic feedback for refresh/scan
 */
export const refreshFeedback = () => HapticFeedback.medium();

/**
 * Haptic feedback for errors
 */
export const errorFeedback = () => HapticFeedback.error();

/**
 * Haptic feedback for QR code scanned
 */
export const qrCodeScannedFeedback = () => HapticFeedback.success();
