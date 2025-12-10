import '@testing-library/jest-native/extend-expect';

// Mock react-native modules that don't work well in test environment
jest.mock('react-native/Libraries/Animated/NativeAnimatedHelper');
jest.mock('react-native/Libraries/EventEmitter/NativeEventEmitter');

// Mock expo modules
jest.mock('expo-secure-store', () => ({
  getItemAsync: jest.fn(),
  setItemAsync: jest.fn(),
  deleteItemAsync: jest.fn(),
}));

jest.mock('expo-crypto', () => ({
  digestStringAsync: jest.fn(),
  CryptoDigestAlgorithm: {
    SHA256: 'SHA256',
  },
}));

jest.mock('expo-camera', () => ({
  Camera: {
    requestCameraPermissionsAsync: jest.fn(),
    getCameraPermissionsAsync: jest.fn(),
  },
  BarCodeScanner: {
    requestPermissionsAsync: jest.fn(),
    getPermissionsAsync: jest.fn(),
  },
}));

jest.mock('react-native-zeroconf', () => ({
  __esModule: true,
  default: jest.fn().mockImplementation(() => ({
    scan: jest.fn(),
    stop: jest.fn(),
    addListener: jest.fn(),
    removeListener: jest.fn(),
  })),
}));

// Mock axios
jest.mock('axios');

// Mock secure-remote-password
jest.mock('secure-remote-password/client', () => ({
  __esModule: true,
  default: jest.fn().mockImplementation(() => ({
    generateEphemeral: jest.fn(),
    deriveSession: jest.fn(),
  })),
}));

// Silence console warnings in tests
global.console = {
  ...console,
  warn: jest.fn(),
  error: jest.fn(),
};
