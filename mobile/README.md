# BoardingPass Mobile App

Cross-platform React Native mobile application for iOS and Android that enables system administrators to discover, authenticate with, and onboard headless Linux devices running the BoardingPass service.

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
# iOS
npm run ios

# Android
npm run android
```

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
```

### Development

```bash
# Type checking
npm run typecheck

# Linting
npm run lint
npm run lint:fix

# Code formatting
npm run format
```

## Architecture

- **Framework**: React Native 0.74+ with Expo SDK 51+
- **Language**: TypeScript 5.x (strict mode)
- **UI**: React Native Paper 5.x (Material Design)
- **Navigation**: Expo Router (file-based routing)
- **State**: React Context API
- **Authentication**: SRP-6a (FIPS 140-3 compatible)
- **Discovery**: mDNS via react-native-zeroconf
- **Security**: Certificate pinning, expo-secure-store

## Project Structure

```
mobile/
├── app/                # Expo Router screens (file-based routing)
├── src/
│   ├── components/    # Reusable UI components
│   ├── services/      # Business logic and API integration
│   ├── hooks/         # Custom React hooks
│   ├── contexts/      # React Context providers
│   ├── types/         # TypeScript type definitions
│   └── utils/         # Utility functions
├── tests/
│   ├── unit/          # Jest unit tests
│   ├── integration/   # Integration tests
│   ├── contract/      # OpenAPI contract tests
│   └── e2e/           # Detox E2E tests
└── package.json
```

## Documentation

- [Feature Specification](../specs/003-mobile-onboarding-app/spec.md)
- [Implementation Plan](../specs/003-mobile-onboarding-app/plan.md)
- [Data Model](../specs/003-mobile-onboarding-app/data-model.md)
- [API Contracts](../specs/003-mobile-onboarding-app/contracts/README.md)
- [Quick Start Guide](../specs/003-mobile-onboarding-app/quickstart.md)

## FIPS Compatibility

**CRITICAL**: The BoardingPass service uses FIPS 140-3 compliant cryptography. The mobile app MUST use compatible SRP-6a parameters:

- Hash Algorithm: SHA-256 (FIPS 180-4 approved)
- SRP Group: RFC 5054 2048-bit safe prime
- Generator: g = 2

See [quickstart.md](../specs/003-mobile-onboarding-app/quickstart.md) for setup instructions.

## License

Same as BoardingPass project (MIT/Apache).
