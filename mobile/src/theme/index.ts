/**
 * Material Design Theme Configuration
 *
 * React Native Paper theme for the BoardingPass mobile app.
 * Based on Material Design 3 (Material You) guidelines.
 */

import { MD3LightTheme as DefaultTheme, configureFonts } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';

/**
 * Color palette
 *
 * Based on Material Design color system with accessibility in mind (WCAG AA compliant)
 */
const colors = {
  // Primary colors (blue - represents trust and security)
  primary: '#1976D2', // Material Blue 700
  onPrimary: '#FFFFFF',
  primaryContainer: '#BBDEFB', // Material Blue 100
  onPrimaryContainer: '#01579B',

  // Secondary colors (teal - represents technology)
  secondary: '#00796B', // Material Teal 700
  onSecondary: '#FFFFFF',
  secondaryContainer: '#B2DFDB', // Material Teal 100
  onSecondaryContainer: '#004D40',

  // Tertiary colors (orange - for warnings/alerts)
  tertiary: '#F57C00', // Material Orange 700
  onTertiary: '#FFFFFF',
  tertiaryContainer: '#FFE0B2', // Material Orange 100
  onTertiaryContainer: '#E65100',

  // Error colors
  error: '#D32F2F', // Material Red 700
  onError: '#FFFFFF',
  errorContainer: '#FFCDD2', // Material Red 100
  onErrorContainer: '#B71C1C',

  // Background colors
  background: '#FFFFFF',
  onBackground: '#1C1B1F',

  // Surface colors
  surface: '#FFFFFF',
  onSurface: '#1C1B1F',
  surfaceVariant: '#E7E0EC',
  onSurfaceVariant: '#49454F',

  // Outline
  outline: '#79747E',
  outlineVariant: '#CAC4D0',

  // Additional colors
  surfaceDisabled: '#1C1B1F1F', // 12% opacity
  onSurfaceDisabled: '#1C1B1F61', // 38% opacity
  backdrop: '#00000066', // 40% opacity
};

/**
 * Typography configuration
 */
const fontConfig = {
  displayLarge: {
    fontFamily: 'System',
    fontSize: 57,
    fontWeight: '400' as const,
    letterSpacing: 0,
    lineHeight: 64,
  },
  displayMedium: {
    fontFamily: 'System',
    fontSize: 45,
    fontWeight: '400' as const,
    letterSpacing: 0,
    lineHeight: 52,
  },
  displaySmall: {
    fontFamily: 'System',
    fontSize: 36,
    fontWeight: '400' as const,
    letterSpacing: 0,
    lineHeight: 44,
  },
  headlineLarge: {
    fontFamily: 'System',
    fontSize: 32,
    fontWeight: '400' as const,
    letterSpacing: 0,
    lineHeight: 40,
  },
  headlineMedium: {
    fontFamily: 'System',
    fontSize: 28,
    fontWeight: '400' as const,
    letterSpacing: 0,
    lineHeight: 36,
  },
  headlineSmall: {
    fontFamily: 'System',
    fontSize: 24,
    fontWeight: '400' as const,
    letterSpacing: 0,
    lineHeight: 32,
  },
  titleLarge: {
    fontFamily: 'System',
    fontSize: 22,
    fontWeight: '400' as const,
    letterSpacing: 0,
    lineHeight: 28,
  },
  titleMedium: {
    fontFamily: 'System',
    fontSize: 16,
    fontWeight: '500' as const,
    letterSpacing: 0.15,
    lineHeight: 24,
  },
  titleSmall: {
    fontFamily: 'System',
    fontSize: 14,
    fontWeight: '500' as const,
    letterSpacing: 0.1,
    lineHeight: 20,
  },
  bodyLarge: {
    fontFamily: 'System',
    fontSize: 16,
    fontWeight: '400' as const,
    letterSpacing: 0.5,
    lineHeight: 24,
  },
  bodyMedium: {
    fontFamily: 'System',
    fontSize: 14,
    fontWeight: '400' as const,
    letterSpacing: 0.25,
    lineHeight: 20,
  },
  bodySmall: {
    fontFamily: 'System',
    fontSize: 12,
    fontWeight: '400' as const,
    letterSpacing: 0.4,
    lineHeight: 16,
  },
  labelLarge: {
    fontFamily: 'System',
    fontSize: 14,
    fontWeight: '500' as const,
    letterSpacing: 0.1,
    lineHeight: 20,
  },
  labelMedium: {
    fontFamily: 'System',
    fontSize: 12,
    fontWeight: '500' as const,
    letterSpacing: 0.5,
    lineHeight: 16,
  },
  labelSmall: {
    fontFamily: 'System',
    fontSize: 11,
    fontWeight: '500' as const,
    letterSpacing: 0.5,
    lineHeight: 16,
  },
};

/**
 * Theme configuration
 */
export const theme: MD3Theme = {
  ...DefaultTheme,
  colors: {
    ...DefaultTheme.colors,
    ...colors,
  },
  fonts: configureFonts({ config: fontConfig }),
  roundness: 12, // Rounded corners for cards and buttons
};

/**
 * Spacing scale (multiples of 4)
 */
export const spacing = {
  xs: 4,
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 48,
} as const;

/**
 * Elevation levels (shadow depth)
 */
export const elevation = {
  none: 0,
  low: 1,
  medium: 3,
  high: 6,
  highest: 12,
} as const;

/**
 * Status colors for certificate trust indicators
 */
export const statusColors = {
  trustedCA: '#4CAF50', // Green - trusted CA
  selfSignedTrusted: '#FFC107', // Yellow - self-signed but trusted
  selfSignedNew: '#FF9800', // Orange - new self-signed
  changed: '#F44336', // Red - certificate changed
} as const;

/**
 * Device status colors
 */
export const deviceStatusColors = {
  online: '#4CAF50', // Green
  offline: '#9E9E9E', // Grey
  authenticating: '#2196F3', // Blue
  authenticated: '#4CAF50', // Green
  error: '#F44336', // Red
} as const;

export default theme;
