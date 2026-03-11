/**
 * Material Design Theme Configuration
 *
 * React Native Paper theme for the BoardingPass mobile app.
 * Colors generated with Material Theme Builder.
 */

import { configureFonts } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';

/**
 * Light color palette — Material Theme Builder export 2026-03-06
 * Seed: #2C4154, Tertiary: #968E9B, Neutral: #F3EFE3
 */
const lightColors = {
  primary: '#2C4154',
  onPrimary: '#FFFFFF',
  primaryContainer: '#2C4154',
  onPrimaryContainer: '#97ADC3',

  secondary: '#575F68',
  onSecondary: '#FFFFFF',
  secondaryContainer: '#D9E0EB',
  onSecondaryContainer: '#5C636C',

  tertiary: '#635C68',
  onTertiary: '#FFFFFF',
  tertiaryContainer: '#968E9B',
  onTertiaryContainer: '#2D2832',

  error: '#BA1A1A',
  onError: '#FFFFFF',
  errorContainer: '#FFDAD6',
  onErrorContainer: '#93000A',

  background: '#FBF9FA',
  onBackground: '#1B1C1D',

  surface: '#FDF8F6',
  onSurface: '#1C1B1B',
  surfaceVariant: '#DFE3E9',
  onSurfaceVariant: '#43474C',

  outline: '#73777D',
  outlineVariant: '#C3C7CD',

  shadow: '#000000',
  scrim: '#000000',
  inverseSurface: '#31302F',
  inverseOnSurface: '#F4F0EE',
  inversePrimary: '#B3C9E0',

  surfaceDisabled: '#1C1B1B1F',
  onSurfaceDisabled: '#1C1B1B61',
  backdrop: '#00000066',

  elevation: {
    level0: 'transparent',
    level1: '#F7F3F1', // surfaceContainerLow
    level2: '#F1EDEB', // surfaceContainer
    level3: '#EBE7E5', // surfaceContainerHigh
    level4: '#E9E5E3',
    level5: '#E6E2E0', // surfaceContainerHighest
  },
};

/**
 * Dark color palette — Material Theme Builder export 2026-03-06
 * Seed: #2C4154, Tertiary: #968E9B, Neutral: #F3EFE3
 */
const darkColors = {
  primary: '#B3C9E0',
  onPrimary: '#1D3245',
  primaryContainer: '#2C4154',
  onPrimaryContainer: '#97ADC3',

  secondary: '#BFC7D1',
  onSecondary: '#293139',
  secondaryContainer: '#424A52',
  onSecondaryContainer: '#B1B9C3',

  tertiary: '#CDC3D1',
  onTertiary: '#342E39',
  tertiaryContainer: '#968E9B',
  onTertiaryContainer: '#2D2832',

  error: '#FFB4AB',
  onError: '#690005',
  errorContainer: '#93000A',
  onErrorContainer: '#FFDAD6',

  background: '#121315',
  onBackground: '#E3E2E4',

  surface: '#141312',
  onSurface: '#E6E2E0',
  surfaceVariant: '#43474C',
  onSurfaceVariant: '#C3C7CD',

  outline: '#8D9197',
  outlineVariant: '#43474C',

  shadow: '#000000',
  scrim: '#000000',
  inverseSurface: '#E6E2E0',
  inverseOnSurface: '#31302F',
  inversePrimary: '#4C6075',

  surfaceDisabled: '#E6E2E01F',
  onSurfaceDisabled: '#E6E2E061',
  backdrop: '#00000066',

  elevation: {
    level0: 'transparent',
    level1: '#1C1B1B', // surfaceContainerLow
    level2: '#201F1F', // surfaceContainer
    level3: '#2B2A29', // surfaceContainerHigh
    level4: '#302F2E',
    level5: '#363433', // surfaceContainerHighest
  },
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

const fonts = configureFonts({ config: fontConfig });

const baseTheme = {
  version: 3 as const,
  isV3: true,
  fonts,
  roundness: 12,
  animation: { scale: 1.0, defaultAnimationDuration: 220 },
};

export const lightTheme: MD3Theme = {
  ...baseTheme,
  dark: false,
  mode: 'exact',
  colors: lightColors,
} as MD3Theme;

export const darkTheme: MD3Theme = {
  ...baseTheme,
  dark: true,
  mode: 'adaptive',
  colors: darkColors,
} as MD3Theme;

/** Default theme (light) for static style references */
export const theme = lightTheme;

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
  unavailable: '#FB8C00', // Amber
  authenticating: '#2196F3', // Blue
  authenticated: '#4CAF50', // Green
  enrolled: '#7E57C2', // Purple
  warning: '#FB8C00', // Amber
  error: '#F44336', // Red
} as const;

export default theme;
