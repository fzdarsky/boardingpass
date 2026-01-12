# Quick Start: BoardingPass Mobile App

**Feature Branch**: `003-mobile-onboarding-app`
**Created**: 2025-12-10
**Status**: Developer Guide Complete

This guide will help you set up, run, and develop the BoardingPass mobile onboarding application for iOS and Android.

---

## Prerequisites

Before you begin, ensure you have the following installed:

### Required

- **Node.js**: Version 18+ LTS ([Download](https://nodejs.org/))
  ```bash
  node --version  # Should be 18.x or higher
  ```

- **npm** or **yarn**: Comes with Node.js
  ```bash
  npm --version   # Should be 9.x or higher
  ```

- **Git**: For version control
  ```bash
  git --version
  ```

### Platform-Specific

#### iOS Development

- **macOS**: Required for iOS development
- **Xcode**: Version 15.0 or later ([Mac App Store](https://apps.apple.com/app/xcode/id497799835))
- **Xcode Command Line Tools**:
  ```bash
  xcode-select --install
  ```
- **CocoaPods**: For iOS dependencies
  ```bash
  sudo gem install cocoapods
  ```

#### Android Development

- **Android Studio**: Latest stable version ([Download](https://developer.android.com/studio))
- **Android SDK**: API Level 29+ (Android 10+)
- **Java Development Kit (JDK)**: Version 17
  ```bash
  java --version  # Should be 17.x
  ```

### Recommended

- **Expo CLI**: Global installation (optional, can use npx)
  ```bash
  npm install -g expo-cli
  ```

- **EAS CLI**: For cloud builds
  ```bash
  npm install -g eas-cli
  ```

---

## Repository Setup

### 1. Clone Repository

```bash
git clone https://github.com/yourcompany/boardingpass.git
cd boardingpass
```

### 2. Checkout Feature Branch

```bash
git checkout 003-mobile-onboarding-app
```

### 3. Navigate to Mobile Directory

```bash
cd mobile
```

The mobile app lives in a `mobile/` directory within the BoardingPass monorepo:

```
boardingpass/
├── cmd/                  # Go service
├── internal/             # Go service internals
├── pkg/                  # Shared Go packages
├── mobile/               # React Native app ← You are here
│   ├── app/
│   ├── src/
│   ├── tests/
│   ├── app.json
│   └── package.json
└── specs/                # Feature specifications
```

---

## Installation

### 1. Install Dependencies

```bash
npm install
```

This installs:
- React Native and Expo SDK
- React Native Paper (Material Design UI)
- react-native-zeroconf (mDNS discovery)
- expo-camera, expo-barcode-scanner (QR scanning)
- expo-secure-store (secure storage)
- Axios (HTTP client)
- TypeScript and development tools

### 2. Generate TypeScript Types

Generate TypeScript types from the BoardingPass OpenAPI specification:

```bash
npm run generate:types
```

This creates `src/types/api.ts` with type-safe API interfaces.

### 3. Prebuild Native Projects

Since the app uses native modules (mDNS, camera), you need to generate iOS and Android projects:

```bash
npx expo prebuild
```

This creates:
- `ios/` directory with Xcode project
- `android/` directory with Android Studio project

**Note**: These directories are generated and not committed to git. Run `npx expo prebuild` again if you pull changes to `app.json` or install new native modules.

---

## FIPS Compatibility Setup

⚠️ **CRITICAL**: The BoardingPass service uses FIPS 140-3 compliant cryptography. The mobile app MUST use compatible SRP-6a parameters.

### Required Configuration

The app MUST configure SRP-6a authentication with:

1. **Hash Algorithm**: SHA-256 (FIPS 180-4 approved)
2. **SRP Group**: RFC 5054 2048-bit safe prime
3. **Generator**: g = 2

### Verification Steps

1. **Review Library Documentation**:
   ```bash
   # Check secure-remote-password library configuration
   npm info secure-remote-password
   ```
   Verify the library supports SHA-256 and RFC 5054 2048-bit groups.

2. **Implementation Check**:
   ```typescript
   // In src/services/auth/srp.ts
   import SRP from 'secure-remote-password/client';

   // Verify configuration matches server:
   const srpClient = new SRP({
     hash: 'sha256',           // MUST be SHA-256 (not SHA-1)
     group: 'rfc5054-2048',    // MUST be 2048-bit (not 1024 or 1536)
   });
   ```

3. **Integration Testing**:
   Test authentication against actual BoardingPass service (not mocks) to verify parameter compatibility.

**See Also**: [`specs/003-mobile-onboarding-app/research.md`](research.md) Section 1 "FIPS Compatibility Requirements" for detailed requirements.

---

## Running the App

### Development Mode

#### iOS Simulator

```bash
npx expo run:ios
```

Or specify a specific simulator:
```bash
npx expo run:ios --simulator "iPhone 15 Pro"
```

#### Android Emulator

```bash
npx expo run:android
```

Or specify a device:
```bash
# List available devices
adb devices

# Run on specific device
npx expo run:android --device <device-id>
```

#### Physical Devices

**iOS**:
```bash
npx expo run:ios --device
```
Requires device connected via USB and registered in Xcode.

**Android**:
```bash
npx expo run:android --device
```
Requires USB debugging enabled on device.

### Development Builds

Since the app uses native modules, you can't use Expo Go. Instead, create development builds:

```bash
# Install development client
npx expo install expo-dev-client

# Create development build
npx expo run:ios    # iOS
npx expo run:android # Android
```

This builds a custom development client with your native modules that supports fast refresh and hot reload.

---

## Project Structure

```
mobile/
├── app/                      # Expo Router app directory (file-based routing)
│   ├── (tabs)/              # Tab-based navigation
│   │   ├── index.tsx        # Device discovery screen (home)
│   │   └── settings.tsx     # Settings screen
│   ├── device/              # Device-related screens
│   │   ├── [id].tsx         # Device detail screen (dynamic route)
│   │   └── authenticate.tsx # Authentication screen
│   └── _layout.tsx          # Root layout with providers
│
├── src/
│   ├── components/          # Reusable UI components
│   │   ├── DeviceList/      # Device list component
│   │   ├── QRScanner/       # QR code scanner
│   │   ├── CertificateInfo/ # Certificate display
│   │   └── ErrorBoundary/   # Error handling
│   │
│   ├── services/            # Business logic and API integration
│   │   ├── discovery/       # mDNS device discovery
│   │   ├── auth/            # SRP-6a authentication
│   │   ├── api/             # BoardingPass API client
│   │   └── certificates/    # Certificate pinning
│   │
│   ├── hooks/               # Custom React hooks
│   │   ├── useDeviceDiscovery.ts
│   │   ├── useAuth.ts
│   │   └── useSecureStorage.ts
│   │
│   ├── contexts/            # React Context providers
│   │   ├── DeviceContext.tsx
│   │   ├── AuthContext.tsx
│   │   └── CertificateContext.tsx
│   │
│   ├── types/               # TypeScript type definitions
│   │   ├── device.ts
│   │   ├── auth.ts
│   │   └── api.ts           # Generated from OpenAPI spec
│   │
│   └── utils/               # Utility functions
│       ├── validation.ts
│       ├── crypto.ts
│       └── errors.ts
│
├── tests/
│   ├── unit/               # Jest unit tests
│   ├── integration/        # Integration tests
│   ├── e2e/               # Detox E2E tests
│   └── contract/          # OpenAPI contract tests
│
├── app.json               # Expo configuration
├── package.json           # NPM dependencies and scripts
├── tsconfig.json          # TypeScript configuration
├── metro.config.js        # Metro bundler config
└── babel.config.js        # Babel configuration
```

---

## Configuration

### Environment Variables

Create a `.env` file in the `mobile/` directory:

```bash
# .env
EXPO_PUBLIC_DEFAULT_DEVICE_PORT=9443
EXPO_PUBLIC_FALLBACK_IP=192.168.1.100
EXPO_PUBLIC_SESSION_TIMEOUT_MINUTES=30
EXPO_PUBLIC_MDNS_SERVICE_NAME=_boardingpass._tcp
```

**Note**: Expo only exposes environment variables prefixed with `EXPO_PUBLIC_`.

### App Configuration

Edit `app.json` to customize app metadata, permissions, and build settings:

```json
{
  "expo": {
    "name": "BoardingPass",
    "slug": "boardingpass-mobile",
    "version": "1.0.0",
    "ios": {
      "bundleIdentifier": "com.yourcompany.boardingpass",
      "infoPlist": {
        "NSCameraUsageDescription": "Camera access required for QR code scanning",
        "NSLocalNetworkUsageDescription": "Network access required to discover devices",
        "NSBonjourServices": ["_boardingpass._tcp"]
      }
    },
    "android": {
      "package": "com.yourcompany.boardingpass",
      "permissions": [
        "CAMERA",
        "ACCESS_FINE_LOCATION",
        "CHANGE_WIFI_MULTICAST_STATE"
      ]
    }
  }
}
```

---

## Testing

### Unit Tests

Run Jest unit tests:

```bash
npm test
```

Run with coverage:
```bash
npm run test:coverage
```

Run specific test file:
```bash
npm test -- src/services/auth/__tests__/srp.test.ts
```

### Integration Tests

Run integration tests against API:

```bash
npm run test:integration
```

**Note**: Integration tests require a running BoardingPass service (see "Testing Against BoardingPass Service" below).

### End-to-End Tests

Run Detox E2E tests:

```bash
# Build E2E test app
npm run e2e:build:ios       # or :android

# Run E2E tests
npm run e2e:test:ios        # or :android
```

### Contract Tests

Validate API responses against OpenAPI specification:

```bash
npm run test:contract
```

---

## Testing Against BoardingPass Service

For authentication and API integration testing, you need a running BoardingPass service.

### Option 1: Local Development Server

```bash
# In repository root (not mobile/ directory)
cd ..
make build
./_output/bin/boardingpass --config path/to/config.yaml
```

### Option 2: Container

```bash
# Run BoardingPass in container
podman run -p 9443:9443 \
  -e BOARDINGPASS_PASSWORD="test-connection-code" \
  boardingpass:latest
```

### Option 3: Physical Device

If you have a physical device running BoardingPass on your local network, ensure:

1. Device is broadcasting via mDNS
2. Mobile device is on the same network
3. Firewall allows HTTPS traffic on port 9443

### Obtaining Connection Codes

Connection codes are generated by the BoardingPass service. For testing:

1. Check device logs for generated connection code
2. Use QR code displayed on device console (if available)
3. For development, configure service with known connection code

---

## Building for Production

### EAS Build (Recommended)

Expo Application Services (EAS) provides cloud-based builds:

```bash
# Login to Expo account
eas login

# Configure project
eas build:configure

# Build for iOS
eas build --platform ios --profile production

# Build for Android
eas build --platform android --profile production
```

### Local Builds

#### iOS

```bash
npx expo run:ios --configuration Release
```

This creates an `.ipa` file in `ios/build/`.

#### Android

```bash
npx expo run:android --variant release
```

This creates an `.apk` or `.aab` file in `android/app/build/outputs/`.

---

## Common Issues and Troubleshooting

### Issue: "Cannot find module 'expo'"

**Solution**: Install dependencies
```bash
npm install
```

### Issue: "Native module not found"

**Solution**: Prebuild native projects
```bash
npx expo prebuild --clean
```

### Issue: "iOS build fails with CocoaPods error"

**Solution**: Update CocoaPods and reinstall
```bash
cd ios
pod repo update
pod install --repo-update
cd ..
```

### Issue: "Android build fails with SDK error"

**Solution**: Verify Android SDK installation
```bash
# In Android Studio: Tools → SDK Manager
# Ensure API Level 29+ is installed
```

### Issue: "mDNS discovery not working"

**Checklist**:
- ✅ Network allows multicast traffic
- ✅ BoardingPass device is broadcasting
- ✅ Mobile device on same network
- ✅ Permissions granted (iOS: Local Network, Android: Location)
- ✅ Try fallback IP address (192.168.1.100:9443)

### Issue: "Authentication fails with SRP error"

**Solution**: Verify FIPS compatibility
```typescript
// Check SRP configuration in src/services/auth/srp.ts
// Ensure SHA-256 and 2048-bit group are configured
```

### Issue: "QR scanner not working"

**Checklist**:
- ✅ Camera permission granted
- ✅ Good lighting conditions
- ✅ QR code is valid BoardingPass connection code format
- ✅ Try manual entry as fallback

### Issue: "Certificate pinning error"

**Solution**: Clear stored certificate pins (development only)
```bash
# Reset app data
npx expo start --clear

# Or manually clear via device settings
# iOS: Settings → General → iPhone Storage → BoardingPass → Delete App
# Android: Settings → Apps → BoardingPass → Clear Data
```

---

## Development Workflow

### 1. Start Development Server

```bash
npm start
```

This starts Metro bundler and shows QR code for Expo Go (note: Expo Go won't work with native modules; use development build).

### 2. Make Code Changes

Edit files in `src/`, `app/`, or `components/`. Metro will hot reload changes automatically.

### 3. Type Check

```bash
npm run typecheck
```

Run this before committing to catch TypeScript errors.

### 4. Lint Code

```bash
npm run lint
```

Fix linting issues:
```bash
npm run lint:fix
```

### 5. Format Code

```bash
npm run format
```

Uses Prettier to format code consistently.

### 6. Run Tests

```bash
npm test
```

Ensure tests pass before committing.

### 7. Commit Changes

```bash
git add .
git commit -m "feat: add device discovery with mDNS"
```

Follow conventional commit format.

---

## Useful Commands

### Development

```bash
npm start                    # Start Metro bundler
npm run ios                  # Run on iOS
npm run android              # Run on Android
npm run web                  # Run in browser (limited functionality)
```

### Type Generation

```bash
npm run generate:types       # Generate types from OpenAPI spec
npm run validate:spec        # Validate OpenAPI specification
```

### Testing

```bash
npm test                     # Run unit tests
npm run test:watch          # Run tests in watch mode
npm run test:coverage       # Run tests with coverage report
npm run test:integration    # Run integration tests
npm run e2e:test:ios        # Run E2E tests on iOS
npm run e2e:test:android    # Run E2E tests on Android
```

### Building

```bash
npx expo prebuild            # Generate native projects
eas build --platform ios     # Build iOS via EAS
eas build --platform android # Build Android via EAS
```

### Utilities

```bash
npm run lint                 # Lint code
npm run lint:fix             # Lint and auto-fix issues
npm run format               # Format code with Prettier
npm run typecheck            # TypeScript type checking
npm run clean                # Clean build artifacts
```

---

## Additional Resources

### Documentation

- **Expo Documentation**: [docs.expo.dev](https://docs.expo.dev)
- **React Native Documentation**: [reactnative.dev](https://reactnative.dev)
- **React Native Paper**: [callstack.github.io/react-native-paper](https://callstack.github.io/react-native-paper)
- **TypeScript Documentation**: [typescriptlang.org/docs](https://www.typescriptlang.org/docs)

### BoardingPass-Specific

- **Feature Specification**: [`specs/003-mobile-onboarding-app/spec.md`](spec.md)
- **Implementation Plan**: [`specs/003-mobile-onboarding-app/plan.md`](plan.md)
- **Research Documentation**: [`specs/003-mobile-onboarding-app/research.md`](research.md)
- **Data Model**: [`specs/003-mobile-onboarding-app/data-model.md`](data-model.md)
- **API Contracts**: [`specs/003-mobile-onboarding-app/contracts/README.md`](contracts/README.md)

### Tools

- **Expo Snack**: Test React Native code in browser - [snack.expo.dev](https://snack.expo.dev)
- **React DevTools**: Debug React components - [reactnative.dev/docs/debugging](https://reactnative.dev/docs/debugging)
- **Flipper**: Mobile app debugger - [fbflipper.com](https://fbflipper.com)

---

## Getting Help

### Community Support

- **Expo Discord**: [chat.expo.dev](https://chat.expo.dev)
- **React Native Community**: [reactnative.dev/community/overview](https://reactnative.dev/community/overview)
- **Stack Overflow**: Tag questions with `react-native`, `expo`, `typescript`

### Project-Specific

- Open an issue on the BoardingPass repository
- Review existing issues and documentation
- Contact project maintainers

---

**Version**: 1.0.0
**Last Updated**: 2025-12-10

**Happy coding! 🚀**
