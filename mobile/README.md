# BoardingPass Mobile App

Cross-platform React Native mobile application for iOS and Android that enables system administrators to discover, authenticate with, and onboard headless Linux devices running the BoardingPass service.

## Features

✨ **mDNS Device Discovery** - Automatically discover BoardingPass devices on the local network
🔐 **SRP-6a Authentication** - Secure passwordless authentication with FIPS 140-3 compatibility
📷 **QR Code Scanning** - Quick authentication via camera
🔒 **Certificate Pinning** - Trust-on-first-use pinning for self-signed TLS certificates
📊 **Device Information** - View system info, TPM status, and network configuration
🎨 **Material Design** - Clean, modern UI using Material Design 3

## Quick Start

See [quickstart.md](../specs/003-mobile-onboarding-app/quickstart.md) for detailed setup instructions.

### Prerequisites

- Node.js 18+ LTS
- npm 9+
- For iOS: Xcode 15+, CocoaPods
- For Android: Android Studio, JDK 17, Android SDK API 29+

### Installation

```bash
# Install dependencies
npm install

# Generate TypeScript types from OpenAPI spec
npm run generate:types

# Prebuild native projects (required for native modules)
npx expo prebuild
```

### Running

```bash
# iOS simulator
npm run ios

# Android emulator
npm run android

# Start Metro bundler
npm start
```

## Development Commands

### Testing

```bash
# Unit tests
npm test

# Integration tests
npm run test:integration

# Contract tests
npm run test:contract

# E2E tests
npm run e2e:test:ios    # or :android

# Test coverage
npm run test:coverage
```

### Code Quality

```bash
# Type checking
npm run typecheck

# Linting
npm run lint
npm run lint:fix

# Code formatting
npm run format
```

### Building

```bash
# Generate native projects
npx expo prebuild

# Build for production (EAS)
eas build --platform ios
eas build --platform android
```

## Architecture

- **Framework**: React Native 0.74+ with Expo SDK 51+
- **Language**: TypeScript 5.x (strict mode)
- **UI**: React Native Paper 5.x (Material Design 3)
- **Navigation**: Expo Router (file-based routing)
- **State**: React Context API + Custom Hooks
- **Authentication**: SRP-6a (FIPS 140-3 compatible)
- **Discovery**: mDNS via react-native-zeroconf
- **Security**: Certificate pinning, expo-secure-store
- **Testing**: Jest, React Native Testing Library, Detox

## Project Structure

```
mobile/
├── app/                      # Expo Router screens (file-based routing)
│   ├── index.tsx            # Device discovery screen
│   ├── _layout.tsx          # Root layout with providers
│   └── device/              # Device-related screens
│       ├── [id].tsx         # Device detail screen
│       └── authenticate.tsx # Authentication screen
│
├── src/
│   ├── components/          # Reusable UI components
│   │   ├── DeviceList/      # Device list and cards
│   │   ├── QRScanner/       # QR code scanner
│   │   ├── CertificateInfo/ # Certificate display
│   │   ├── DeviceInfo/      # Device information
│   │   ├── Skeleton/        # Loading skeletons
│   │   └── Errors/          # Error views
│   │
│   ├── services/            # Business logic and API
│   │   ├── discovery/       # mDNS device discovery
│   │   ├── auth/            # SRP-6a authentication
│   │   ├── api/             # BoardingPass API client
│   │   └── certificates/    # Certificate pinning
│   │
│   ├── hooks/               # Custom React hooks
│   │   ├── useDeviceDiscovery.ts
│   │   ├── useAuth.ts
│   │   ├── useDeviceInfo.ts
│   │   └── useSecureStorage.ts
│   │
│   ├── contexts/            # React Context providers
│   │   ├── DeviceContext.tsx
│   │   ├── AuthContext.tsx
│   │   └── CertificateContext.tsx
│   │
│   ├── types/               # TypeScript types
│   │   ├── device.ts
│   │   ├── auth.ts
│   │   └── api.ts           # Generated from OpenAPI
│   │
│   └── utils/               # Utilities
│       ├── validation.ts
│       ├── crypto.ts
│       ├── errors.ts
│       └── haptics.ts
│
└── tests/
    ├── unit/                # Jest unit tests
    ├── integration/         # Integration tests
    ├── contract/            # OpenAPI contract tests
    └── e2e/                 # Detox E2E tests
```

## Configuration

Create a `.env` file:

```bash
EXPO_PUBLIC_DEFAULT_DEVICE_PORT=9443
EXPO_PUBLIC_FALLBACK_IP=192.168.1.100
EXPO_PUBLIC_SESSION_TIMEOUT_MINUTES=30
EXPO_PUBLIC_MDNS_SERVICE_NAME=_boardingpass._tcp
```

## Security

- Connection codes are **NEVER** logged or persisted
- Session tokens stored in encrypted OS storage (Keychain/Keystore)
- SRP ephemeral keys cleared from memory after authentication
- Certificate pinning with Trust-On-First-Use (TOFU)
- Progressive authentication delays: 1s → 2s → 5s → 60s lockout

## FIPS Compatibility

⚠️ **CRITICAL**: The BoardingPass service uses FIPS 140-3 compliant cryptography. The mobile app MUST use compatible SRP-6a parameters:

- **Hash Algorithm**: SHA-256 (FIPS 180-4 approved)
- **SRP Group**: RFC 5054 2048-bit safe prime
- **Generator**: g = 2

Configuration: `src/services/auth/srp.ts`

## Troubleshooting

### Type errors after API changes

```bash
npm run generate:types    # Regenerate types
npm run typecheck          # Verify
```

### Native module not found

```bash
npx expo prebuild --clean  # Regenerate native projects
npm run ios                # Rebuild
```

### mDNS discovery not working

- Verify network allows multicast
- Check device is broadcasting
- Grant permissions (iOS: Local Network, Android: Location)
- Try fallback IP address

## Documentation

- [Feature Specification](../specs/003-mobile-onboarding-app/spec.md)
- [Implementation Plan](../specs/003-mobile-onboarding-app/plan.md)
- [Data Model](../specs/003-mobile-onboarding-app/data-model.md)
- [Technical Research](../specs/003-mobile-onboarding-app/research.md)
- [Quick Start Guide](../specs/003-mobile-onboarding-app/quickstart.md)

## Performance Targets

- Device Discovery: < 10 seconds
- Authentication: < 30 seconds
- Device Info Load: < 5 seconds
- UI Rendering: 60 FPS
- Bundle Size: < 50MB

## Contributing

Before submitting changes:

1. `npm run typecheck` - No TypeScript errors
2. `npm run lint` - No linting errors
3. `npm test` - All tests pass
4. Test on both iOS and Android
5. Update documentation

## License

Same as BoardingPass project (MIT/Apache)
