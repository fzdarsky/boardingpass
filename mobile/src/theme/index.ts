/**
 * Material Design Theme Configuration
 *
 * React Native Paper theme for the BoardingPass mobile app.
 * Colors generated with Material Theme Builder.
 */

import { configureFonts } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';

/**
 * Light color palette — exported from Material Theme Builder
 */
const lightColors = {
  primary: '#2D628B',
  onPrimary: '#FFFFFF',
  primaryContainer: '#CDE5FF',
  onPrimaryContainer: '#0A4A72',

  secondary: '#2D628C',
  onSecondary: '#FFFFFF',
  secondaryContainer: '#CDE5FF',
  onSecondaryContainer: '#0A4A72',

  tertiary: '#715188',
  onTertiary: '#FFFFFF',
  tertiaryContainer: '#F3DAFF',
  onTertiaryContainer: '#583A6F',

  error: '#BA1A1A',
  onError: '#FFFFFF',
  errorContainer: '#FFDAD6',
  onErrorContainer: '#93000A',

  background: '#F7F9FF',
  onBackground: '#181C20',

  surface: '#FDF9EC',
  onSurface: '#1C1C14',
  surfaceVariant: '#E7E3D1',
  onSurfaceVariant: '#49473A',

  outline: '#7A7768',
  outlineVariant: '#CAC7B5',

  shadow: '#000000',
  scrim: '#000000',
  inverseSurface: '#323128',
  inverseOnSurface: '#F5F1E3',
  inversePrimary: '#9ACBFA',

  surfaceDisabled: '#1C1C141F',
  onSurfaceDisabled: '#1C1C1461',
  backdrop: '#00000066',

  elevation: {
    level0: 'transparent',
    level1: '#F8F4E6',
    level2: '#F2EEE0',
    level3: '#ECE8DA',
    level4: '#EAE6D7',
    level5: '#E6E2D3',
  },
};

/**
 * Dark color palette — exported from Material Theme Builder
 */
const darkColors = {
  primary: '#9ACBFA',
  onPrimary: '#003352',
  primaryContainer: '#0A4A72',
  onPrimaryContainer: '#CDE5FF',

  secondary: '#9ACBFA',
  onSecondary: '#003352',
  secondaryContainer: '#0A4A72',
  onSecondaryContainer: '#CDE5FF',

  tertiary: '#DEB8F7',
  onTertiary: '#412357',
  tertiaryContainer: '#583A6F',
  onTertiaryContainer: '#F3DAFF',

  error: '#FFB4AB',
  onError: '#690005',
  errorContainer: '#93000A',
  onErrorContainer: '#FFDAD6',

  background: '#101418',
  onBackground: '#E0E2E8',

  surface: '#14140C',
  onSurface: '#E6E2D5',
  surfaceVariant: '#49473A',
  onSurfaceVariant: '#CAC7B5',

  outline: '#949181',
  outlineVariant: '#49473A',

  shadow: '#000000',
  scrim: '#000000',
  inverseSurface: '#E6E2D5',
  inverseOnSurface: '#323128',
  inversePrimary: '#2D628B',

  surfaceDisabled: '#E6E2D51F',
  onSurfaceDisabled: '#E6E2D561',
  backdrop: '#00000066',

  elevation: {
    level0: 'transparent',
    level1: '#1E1E15',
    level2: '#23231C',
    level3: '#282822',
    level4: '#2A2A24',
    level5: '#2D2D27',
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
  authenticating: '#2196F3', // Blue
  authenticated: '#4CAF50', // Green
  error: '#F44336', // Red
} as const;

export default theme;
