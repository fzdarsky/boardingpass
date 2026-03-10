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

- Node.js 20+ LTS
- npm 10+
- For iOS: Xcode 15-26 (tested with 26.2), CocoaPods
- For Android: Android Studio, JDK 17, Android SDK API 29+

### Installation

**Using Makefile (recommended from project root):**

```bash
make install-deps-app     # Install dependencies
make generate-app         # Generate TypeScript types from OpenAPI spec
make build-app-ios        # Generate iOS native project
```

**Or using npm directly:**

```bash
# Install dependencies
npm install

# Generate TypeScript types from OpenAPI spec
npm run generate:types

# Prebuild native projects (required for native modules)
npx expo prebuild
```

### Running

**Using Makefile (from project root):**

```bash
make run-app-ios         # iOS simulator
make run-app-android     # Android emulator
make run-app             # Default platform (macOS→iOS, Linux→Android)
```

**Or using npm:**

```bash
# iOS simulator
npm run ios

# Android emulator
npm run android

# Start Metro bundler
npm start
```

## Development Commands

The BoardingPass project uses a consistent Makefile naming scheme. All targets follow the pattern: `{action}-app[-{platform}]`

### Makefile Targets (run from project root)

```bash
# Installation & Setup
make install-deps-app        # Install npm dependencies
make generate-app            # Generate TypeScript types from OpenAPI

# Building
make build-app-ios           # Generate iOS native project (expo prebuild)
make build-app-android       # Generate Android native project
make build-app               # Generate both iOS and Android

# Testing
make test-unit-app           # Run unit tests
make test-integration-app    # Run integration tests
make test-e2e-app-ios        # Run E2E tests on iOS
make test-e2e-app-android    # Run E2E tests on Android
make test-contract-app       # Run contract tests
make test-app                # Run all tests

# Code Quality
make lint-app                # Run ESLint

# Running
make run-app-ios             # Run on iOS simulator
make run-app-android         # Run on Android emulator
make run-app                 # Run on default platform

# Cleaning
make clean-cache-app         # Clear Metro cache (.expo, node_modules/.cache)
make clean-native-app        # Remove native projects (ios/, android/)
make clean-app               # Clean cache + native projects
make clean-app-full          # Deep clean including node_modules

# Troubleshooting Workflows
make fix-app                 # Fix common issues (Xcode 26, deps, cache)
make rebuild-app-ios         # Full rebuild: clean + build + run (iOS)
make rebuild-app-android     # Full rebuild: clean + build + run (Android)
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

```text
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
EXPO_PUBLIC_DEFAULT_DEVICE_PORT=8443
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

### Xcode 26+ build errors (iOS)

If you see `cannot find 'TARGET_IPHONE_SIMULATOR' in scope` in `expo-dev-menu`:

**Using Makefile (from project root):**

```bash
make fix-app         # Runs expo install --fix + clean + prebuild
make run-app-ios     # Run on iOS
```

**Or manually:**

```bash
# This is a known compatibility issue with Xcode 26+
# First, ensure you have the latest Expo SDK 51 packages
cd mobile
npx expo install --fix

# Then rebuild the iOS project
rm -rf ios && npx expo prebuild --platform ios
npm run ios
```

If the error persists, you may need to manually patch `node_modules/expo-dev-menu/ios/DevMenuViewController.swift` line 66:

```swift
# Replace:
let isSimulator = TARGET_IPHONE_SIMULATOR > 0

# With:
#if targetEnvironment(simulator)
let isSimulator = true
#else
let isSimulator = false
#endif
```

### "Problem loading the project" error

This can be caused by two issues:

**1. Missing native dependencies:**

**Using Makefile:**

```bash
cd mobile && npm run typecheck    # Look for "Cannot find module 'expo-*'" errors
cd mobile && npx expo install expo-haptics  # Install any missing modules
make build-app-ios                # Rebuild native projects (from project root)
make run-app-ios                  # Run again
```

**Or manually:**

```bash
cd mobile
npm run typecheck                 # Look for "Cannot find module 'expo-*'" errors
npx expo install expo-haptics     # Install any missing modules
rm -rf ios && npx expo prebuild --platform ios  # Rebuild native projects
npm run ios                       # Run again
```

**Note**: Always run `make build-app-ios` (or `npx expo prebuild`) after installing new Expo native modules (expo-haptics, expo-camera, etc.) to regenerate native code.

**2. Metro bundler cache issues (shows "Unable to resolve module ./index"):**

**Using Makefile:**

```bash
make clean-cache-app       # Clear Metro cache (from project root)
make run-app-ios           # Restart with clear cache
```

**Or manually:**

```bash
cd mobile
# Clear Metro cache and restart
rm -rf .expo node_modules/.cache
npx expo start --clear

# In another terminal, rebuild the app
npm run ios
```

This is common after installing native modules, running prebuild, or switching git branches.

### "Unable to resolve module crypto" error

This occurs when packages (like axios) try to load Node.js-specific modules instead of browser/React Native versions.

**Solution**: Ensure [metro.config.js](metro.config.js) has proper package resolution:

```javascript
// Configure resolver to use React Native and browser-compatible modules
config.resolver.resolverMainFields = ['react-native', 'browser', 'main'];
config.resolver.unstable_enablePackageExports = true;
```

Then clear cache and rebuild:

**Using Makefile:**

```bash
make clean-app         # Clear cache + native projects (from project root)
make build-app-ios     # Regenerate iOS project
make run-app-ios       # Run
```

**Or manually:**

```bash
cd mobile
rm -rf .expo node_modules/.cache ios android
npx expo prebuild --platform ios
npm run ios
```

### Type errors after API changes

**Using Makefile:**

```bash
make generate-app          # Regenerate types from OpenAPI (from project root)
cd mobile && npm run typecheck  # Verify
```

**Or manually:**

```bash
cd mobile
npm run generate:types     # Regenerate types
npm run typecheck          # Verify
```

### Native module not found

**Using Makefile:**

```bash
make clean-native-app      # Remove ios/android (from project root)
make build-app-ios         # Regenerate native projects
make run-app-ios           # Rebuild
```

**Or manually:**

```bash
cd mobile
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

## Releasing to Testers (TestFlight)

The iOS app is distributed to testers via TestFlight using [EAS Build](https://docs.expo.dev/build/introduction/). Build configuration is in [eas.json](eas.json).

### First-Time Setup

1. Install EAS CLI: `npm install -g eas-cli`
2. Log in: `eas login`
3. Initialize project: `eas init` (sets the `projectId` in `app.config.js`)
4. Configure Apple credentials: `eas credentials` (EAS manages certificates and provisioning profiles)
5. Create the app in [App Store Connect](https://appstoreconnect.apple.com/) with bundle ID `org.boardingpass-project.mobile`
6. Update `ascAppId` in `eas.json` with the App Store Connect app ID

### Build & Submit

**Manual release:**

```bash
eas build --platform ios --profile production    # Build on EAS cloud (~15 min)
eas submit --platform ios --latest               # Upload to TestFlight
```

Or from the project root: `make release-app-ios`

**Automated release:** Push a tag matching `app-v*.*.*`:

```bash
git tag app-v1.0.0 && git push origin app-v1.0.0
```

The [release-app](../.github/workflows/release-app.yaml) workflow builds and submits automatically.

### Adding Testers

In [App Store Connect](https://appstoreconnect.apple.com/) > Your App > TestFlight:

- **Internal testers** (up to 100 team members): add by Apple ID email, access is immediate
- **External testers** (up to 10,000): add by email or share a public link, first build requires beta review (~24 hours)

Testers install the free [TestFlight](https://apps.apple.com/app/testflight/id899247664) app, open the invitation link, and tap Install. Builds expire after 90 days.

### Build Profiles

| Profile        | Distribution           | Use Case                    |
| -------------- | ---------------------- | --------------------------- |
| `development`  | Internal (simulator)   | Day-to-day development      |
| `preview`      | Internal (ad-hoc)      | Quick QA without TestFlight |
| `production`   | App Store / TestFlight | Tester and release builds   |

## Contributing

Before submitting changes:

**Using Makefile (from project root):**

1. `make generate-app` - Regenerate types if OpenAPI changed
2. `cd mobile && npm run typecheck` - No TypeScript errors
3. `make lint-app` - No linting errors
4. `make test-app` - All tests pass
5. Test on both iOS and Android (`make run-app-ios`, `make run-app-android`)
6. Update documentation

**Or using npm:**

1. `npm run typecheck` - No TypeScript errors
2. `npm run lint` - No linting errors
3. `npm test` - All tests pass
4. Test on both iOS and Android
5. Update documentation

## License

Same as BoardingPass project (MIT/Apache)
