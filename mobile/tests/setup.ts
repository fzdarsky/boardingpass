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
  digestStringAsync: jest.fn().mockResolvedValue('a'.repeat(64)),
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

// Mock axios with a factory that returns a working instance
jest.mock('axios', () => {
  const mockAxiosInstance = {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    head: jest.fn(),
    patch: jest.fn(),
    request: jest.fn(),
    interceptors: {
      request: { use: jest.fn(), eject: jest.fn() },
      response: { use: jest.fn(), eject: jest.fn() },
    },
    defaults: { headers: { common: {} } },
  };

  return {
    __esModule: true,
    default: {
      ...mockAxiosInstance,
      create: jest.fn(() => ({ ...mockAxiosInstance })),
      isAxiosError: jest.fn((err: unknown) => !!(err as Record<string, unknown>)?.isAxiosError),
    },
  };
});

// Mock secure-remote-password/client (used as: import * as srp from 'secure-remote-password/client')
jest.mock('secure-remote-password/client', () => ({
  generateEphemeral: jest.fn(() => ({
    secret: 'mock-secret',
    public: 'mock-public-A',
  })),
  derivePrivateKey: jest.fn(() => 'mock-private-key'),
  deriveSession: jest.fn(() => ({
    key: 'mock-session-key',
    proof: 'mock-client-proof-M1',
  })),
  verifySession: jest.fn(),
}));

// Silence console warnings in tests
global.console = {
  ...console,
  warn: jest.fn(),
  error: jest.fn(),
};
