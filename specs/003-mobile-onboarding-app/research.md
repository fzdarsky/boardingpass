# Mobile Onboarding App Technical Research

**Feature Branch**: `003-mobile-onboarding-app`
**Created**: 2025-12-10
**Status**: Research Complete

This document provides comprehensive technical research for key architectural decisions in the BoardingPass mobile onboarding application. Each section evaluates options, makes recommendations, and provides implementation guidance aligned with the feature specification and BoardingPass constitution principles.

---

## 1. SRP-6a Implementation in React Native/TypeScript

### Context

The BoardingPass service uses SRP-6a (Secure Remote Password) authentication protocol for passwordless authentication. The mobile app must implement the client side of this protocol to authenticate with devices securely. This is a critical security component as it forms the foundation of the entire authentication flow (FR-010, FR-011).

### Options Evaluated

#### 1.1 **thinbus-srp (npm: thinbus-srp)**

- **Description**: Pure JavaScript implementation of SRP-6a protocol, browser and Node.js compatible
- **Pros**:
  - Pure JavaScript, no native dependencies
  - Well-documented API
  - Supports multiple groups (1024, 1536, 2048-bit)
  - Used in production environments
  - Clear separation of client/server implementations
- **Cons**:
  - Last major update several years ago (maintenance concerns)
  - Limited TypeScript type definitions
  - Not specifically tested with React Native
  - No official mobile-specific documentation
- **Security**:
  - Implements standard SRP-6a protocol correctly
  - Uses crypto-browserify for cryptographic operations
  - Requires careful review of random number generation in React Native context

#### 1.2 **secure-remote-password (npm: secure-remote-password)**

- **Description**: Modern JavaScript/TypeScript SRP-6a implementation with focus on simplicity and security
- **Pros**:
  - Modern codebase with TypeScript support
  - Clean, minimal API surface
  - Good test coverage
  - Works in both browser and Node.js environments
  - Active maintenance as of 2023-2024
  - Explicitly designed for zero-knowledge password proofs
- **Cons**:
  - Smaller community compared to thinbus-srp
  - May require polyfills for React Native crypto operations
  - Documentation could be more comprehensive
- **Security**:
  - Uses Web Crypto API or Node crypto module
  - Proper handling of ephemeral keys
  - Implements RFC 5054 SRP-6a specification

#### 1.3 **Custom Implementation from Scratch**

- **Description**: Implement SRP-6a protocol directly using React Native crypto libraries
- **Pros**:
  - Full control over implementation details
  - Can optimize for specific React Native requirements
  - No external dependency for critical security component
  - Aligns with "minimal dependencies" constitution principle
- **Cons**:
  - High risk of implementation errors in cryptographic protocol
  - Significant development and testing effort required
  - Requires deep expertise in SRP-6a protocol
  - Maintenance burden entirely on team
  - Violates "don't roll your own crypto" security principle
- **Security**:
  - High risk - cryptographic protocols are notoriously difficult to implement correctly
  - Requires extensive security review and testing
  - Must handle edge cases, timing attacks, and protocol validation

#### 1.4 **Use React Native-specific fork or wrapper**

- **Description**: Wrap existing SRP library with React Native crypto polyfills
- **Pros**:
  - Leverages proven SRP implementations
  - Can integrate with react-native-crypto or expo-crypto
  - Maintains compatibility with React Native's JavaScript engine
- **Cons**:
  - Adds another layer of dependencies
  - Polyfills may have performance implications
  - Increased complexity in build configuration
- **Security**:
  - Depends on both underlying SRP library and crypto polyfill security
  - Additional attack surface from polyfill layer

### Decision

**Chosen**: **secure-remote-password** with React Native crypto polyfills (expo-crypto or react-native-crypto)

### Rationale

1. **Modern TypeScript Support**: Native TypeScript support provides type safety for the complex SRP protocol flow, reducing integration errors
2. **Active Maintenance**: More recent maintenance activity compared to thinbus-srp reduces security vulnerability risk
3. **Clean API**: Simple, focused API reduces likelihood of misuse compared to more complex alternatives
4. **Balanced Approach**: Avoids "rolling our own crypto" while maintaining reasonable dependency minimization
5. **Testing**: Good test coverage provides confidence in protocol implementation correctness
6. **Expo Compatibility**: Can leverage expo-crypto for cryptographic primitives without ejecting from Expo managed workflow

While this adds an external dependency (counter to constitution's minimal dependencies principle), implementing SRP-6a from scratch poses unacceptable security risks. The library is small, focused, and addresses a critical security requirement.

### FIPS Compatibility Requirements (CRITICAL)

**BLOCKING ISSUE**: The BoardingPass service operates in FIPS 140-3 mode with strict cryptographic requirements. The mobile app MUST use the exact same SRP-6a parameters as the server, or authentication will fail.

**Required SRP-6a Parameters** (from `internal/auth/srp.go`):

1. **Hash Algorithm**: SHA-256 (FIPS 180-4 approved)
   - Server uses `crypto/sha256` (lines 6, 44 in `internal/auth/srp.go`)
   - Client MUST configure `secure-remote-password` library to use SHA-256
   - ⚠️ **NEVER use SHA-1** - not FIPS-compliant and will cause authentication failure

2. **SRP Group**: RFC 5054 2048-bit safe prime (FIPS 186-4 compliant)
   - Server uses RFC 5054 2048-bit group (lines 13-40 in `internal/auth/srp.go`)
   - Client MUST use the same 2048-bit group
   - ⚠️ **NEVER use 1024-bit or 1536-bit groups** - insufficient security and incompatible

3. **Group Generator**: g = 2
   - Server uses generator g=2 (line 21 in `internal/auth/srp.go`)
   - Client MUST use the same generator

**Library Configuration**:

```typescript
// Ensure secure-remote-password library uses FIPS-compatible parameters
import SRP from 'secure-remote-password/client';

// CRITICAL: Verify library is configured for:
// - Hash: SHA-256 (not SHA-1 or other algorithms)
// - Group: 2048-bit RFC 5054 prime
// - Generator: g = 2

// Example initialization (verify with library documentation):
const srp = new SRP({
  hash: 'sha256',          // MUST be SHA-256
  group: 'rfc5054-2048',   // MUST be 2048-bit RFC 5054 group
  // Verify library defaults match server requirements
});
```

**Verification Steps**:

1. **Before implementation**: Review `secure-remote-password` library documentation to confirm:
   - Default or configurable hash algorithm is SHA-256
   - Supports RFC 5054 2048-bit group
   - Generator g=2 is used

2. **During implementation**: Add logging (development only) to verify:
   ```typescript
   console.log('SRP Config:', {
     hash: 'SHA-256',  // Verify this matches library actual config
     group: '2048-bit RFC 5054',
     generator: 2
   });
   ```

3. **During testing**: Authentication failures with error messages about invalid proofs indicate parameter mismatch. Verify:
   - Client and server use identical hash algorithm
   - Client and server use identical group parameters
   - Both use same generator

**Common Pitfalls**:

- ❌ Using library defaults without verification (may default to SHA-1 or 1024-bit groups)
- ❌ Assuming all SRP-6a implementations are compatible (they're not - parameters must match exactly)
- ❌ Testing against mock server with different crypto parameters (masks incompatibility until production)

**Testing Strategy**:

- Integration tests MUST authenticate against actual BoardingPass service (not mocks)
- Test authentication with known good connection codes to verify parameter compatibility
- Monitor for authentication failures that could indicate crypto parameter mismatch

### Implementation Notes

1. **Crypto Polyfill Strategy**:
   ```typescript
   // Use expo-crypto for managed workflow compatibility
   import * as Crypto from 'expo-crypto';

   // Wrap secure-remote-password with React Native crypto primitives
   ```

2. **Random Number Generation**: Verify that expo-crypto provides cryptographically secure random bytes for ephemeral key generation

3. **BigInteger Operations**: Ensure JavaScript BigInt support or use the library's built-in big number handling

4. **Protocol Flow**:
   ```typescript
   // Step 1: Client generates ephemeral key pair (A = g^a)
   const client = new SRPClient();
   const A = client.generateEphemeral();

   // Step 2: Send username + A to server, receive salt + B
   const { salt, B } = await api.srpInit({ username, A });

   // Step 3: Client derives session key and generates proof M1
   const M1 = client.generateProof(password, salt, B);

   // Step 4: Send M1, receive M2 and session token
   const { M2, sessionToken } = await api.srpVerify({ M1 });

   // Step 5: Verify server proof M2
   client.verifyServerProof(M2);
   ```

5. **Error Handling**: Implement robust error handling for:
   - Invalid server responses (malformed B, salt)
   - Server proof verification failures
   - Network timeouts during multi-step protocol
   - Progressive delay on authentication failures (FR-038)

6. **Security Considerations**:
   - Clear ephemeral keys and proofs from memory after authentication
   - Never log password, A, M1, or other sensitive protocol values
   - Validate server's ephemeral public key B is in valid range
   - Implement timeout for authentication flow to prevent hanging sessions

7. **Testing Strategy**:
   - Unit tests with known SRP test vectors
   - Integration tests against actual BoardingPass service
   - Negative testing with invalid server responses
   - Performance testing to ensure sub-30-second authentication (SC-002)

---

## 2. Certificate Pinning with Self-Signed Certificates

### Context

BoardingPass devices may use self-signed TLS certificates. The app must establish trust on first use (TOFU), pin the certificate to prevent MITM attacks on subsequent connections, and alert users if certificates change (FR-031, FR-032, FR-033, FR-034, FR-035). This is critical for security in environments where devices cannot obtain CA-signed certificates.

### Options Evaluated

#### 2.1 **Axios with Custom HTTPS Agent (Node.js style)**

- **Description**: Configure Axios with custom HTTPS agent for certificate validation
- **Pros**:
  - Familiar pattern from Node.js development
  - Direct control over TLS validation logic
  - Can inspect certificate details before accepting
- **Cons**:
  - React Native doesn't use Node.js https module
  - Requires platform-specific native modules for actual TLS control
  - Not directly compatible with React Native's networking layer
- **Security**:
  - Would provide strong security if it worked in React Native
  - Unfortunately not applicable to mobile environment

#### 2.2 **react-native-ssl-pinning**

- **Description**: Native module for certificate pinning on iOS and Android
- **Pros**:
  - Designed specifically for React Native certificate pinning
  - Supports both public key pinning and certificate pinning
  - Native implementation provides strong security guarantees
  - Can specify trusted certificates at runtime
- **Cons**:
  - Requires ejecting from Expo managed workflow
  - Adds native code maintenance burden
  - May conflict with Expo's OTA updates
  - Complexity in managing pinned certificates across app updates
- **Security**:
  - Strong security through native TLS implementation
  - Prevents MITM attacks effectively once pins are established
  - Requires careful pin management to avoid lockout scenarios

#### 2.3 **Custom Certificate Validation with expo-crypto + Manual Verification**

- **Description**: Fetch certificate info, compute fingerprint, store and validate manually
- **Pros**:
  - Stays within Expo managed workflow
  - Full control over trust decision flow
  - Can implement custom TOFU UI/UX
  - No native module dependencies
- **Cons**:
  - Doesn't actually pin at TLS layer - validation happens after connection
  - Vulnerable to MITM on very first connection before fingerprint stored
  - More complex to implement correctly
  - May miss sophisticated MITM attacks
- **Security**:
  - Weaker than true certificate pinning
  - Vulnerable during initial connection establishment
  - Relies on application layer validation rather than transport layer

#### 2.4 **Trust First Certificate + Warn on Change Strategy**

- **Description**: Accept first certificate blindly, store fingerprint, warn user if it changes
- **Pros**:
  - Simple to implement
  - Works within Expo managed workflow
  - Good UX for expected use case (device's certificate doesn't change)
  - Clear user communication on certificate changes
- **Cons**:
  - Vulnerable to MITM on very first connection
  - Relies on user to make correct trust decision
  - No cryptographic validation of initial certificate
- **Security**:
  - Weak on first connection (TOFU vulnerability window)
  - Reasonable security after initial trust establishment
  - User education critical for security

#### 2.5 **Hybrid Approach: Certificate Inspection + Fingerprint Pinning**

- **Description**: Fetch and display certificate details on first connection, compute fingerprint with expo-crypto, prompt user to verify, pin fingerprint for future connections
- **Pros**:
  - Balances security with Expo managed workflow compatibility
  - User can inspect certificate details before trusting (FR-033)
  - Stores fingerprint for subsequent validation
  - Clear warning if certificate changes (FR-035)
  - Implements TOFU with user confirmation (FR-034)
- **Cons**:
  - Still vulnerable during actual first TLS handshake
  - Requires fetching certificate separately from main API calls
  - More complex state management
  - User must understand certificate concepts
- **Security**:
  - Better than blind trust, weaker than native pinning
  - Provides defense-in-depth through user verification
  - Detects certificate changes reliably

### Decision

**Chosen**: **Hybrid Approach: Certificate Inspection + Fingerprint Pinning** with user confirmation on first connection

### Rationale

1. **Expo Compatibility**: Maintains Expo managed workflow compatibility, critical for development velocity and OTA updates
2. **User-Centric Security**: Aligns with FR-034 requirement for explicit user trust confirmation on first connection
3. **Balance**: Provides reasonable security without requiring native modules or bare workflow
4. **Transparency**: Enables FR-033 requirement to display certificate information (issuer, subject, validity, fingerprint)
5. **Change Detection**: Satisfies FR-035 requirement to alert on certificate changes
6. **Practical**: Acknowledges that TOFU is acceptable for this use case (local network, physical device access)

The TOFU vulnerability is acceptable because:
- Users have physical access to devices (can verify device identity)
- Devices are on local network (reduced MITM risk)
- First connection typically happens during initial setup in controlled environment
- Certificate change detection provides ongoing protection

### Implementation Notes

1. **Certificate Fetching**:
   ```typescript
   // Use expo-crypto to fetch and parse certificate
   import * as Crypto from 'expo-crypto';

   async function fetchCertificate(deviceUrl: string): Promise<CertificateInfo> {
     // Attempt TLS connection, extract certificate
     // Parse X.509 certificate details
     // Return issuer, subject, validity dates, public key
   }
   ```

2. **Fingerprint Computation**:
   ```typescript
   async function computeFingerprint(certificate: string): Promise<string> {
     // Use SHA-256 hash of certificate DER encoding
     const hash = await Crypto.digestStringAsync(
       Crypto.CryptoDigestAlgorithm.SHA256,
       certificate,
       { encoding: Crypto.CryptoEncoding.BASE64 }
     );
     return hash;
   }
   ```

3. **Storage Strategy**:
   ```typescript
   import * as SecureStore from 'expo-secure-store';

   interface PinnedCertificate {
     deviceId: string;
     fingerprint: string;
     issuer: string;
     subject: string;
     validFrom: string;
     validTo: string;
     trustedAt: string;
   }

   async function pinCertificate(deviceId: string, cert: PinnedCertificate) {
     await SecureStore.setItemAsync(
       `cert_pin_${deviceId}`,
       JSON.stringify(cert)
     );
   }
   ```

4. **First Connection Flow**:
   ```
   1. Attempt connection to device
   2. Fetch certificate details
   3. Display certificate info UI:
      - Issuer (with CA vs self-signed indicator)
      - Subject (device name/IP)
      - Validity period
      - SHA-256 fingerprint (formatted)
      - Trust status indicator
   4. User reviews and confirms trust
   5. Compute and store fingerprint
   6. Proceed with authentication
   ```

5. **Subsequent Connection Flow**:
   ```
   1. Attempt connection to device
   2. Fetch current certificate
   3. Compute fingerprint
   4. Compare with stored fingerprint
   5. If match: proceed silently
   6. If mismatch: show warning UI with details of both certificates
   7. User decides to trust new certificate or abort
   ```

6. **Certificate Information Display** (FR-033):
   ```typescript
   interface CertificateDisplayInfo {
     isSelfSigned: boolean;
     status: 'trusted_ca' | 'self_signed_trusted' | 'self_signed_new' | 'changed';
     issuer: string;
     subject: string;
     validFrom: Date;
     validTo: Date;
     fingerprint: string;  // SHA-256, formatted as hex with colons
   }

   // Show in UI with appropriate icons/colors:
   // - Green checkmark for trusted CA
   // - Yellow shield for self-signed but pinned
   // - Orange warning for self-signed new device
   // - Red alert for certificate change
   ```

7. **Security Considerations**:
   - Display fingerprint in easily verifiable format (hex with colons or groups)
   - Warn users about expired certificates
   - Consider requiring re-authentication after certificate change
   - Clear old pins for devices not seen in extended period
   - Log certificate changes for troubleshooting

8. **User Experience**:
   - Use simple language ("This device uses its own security certificate")
   - Provide clear actions ("Trust This Device" vs "Cancel")
   - Show certificate status icon in device list
   - Tooltip or info button for certificate details
   - Clear warning dialogs for certificate changes

9. **Limitations and Mitigations**:
   - **Limitation**: Vulnerable to MITM on first connection
   - **Mitigation**: Guide users to perform first connection in secure environment
   - **Limitation**: Users may not understand certificate concepts
   - **Mitigation**: Use clear, non-technical language and visual indicators
   - **Limitation**: No protection during actual TLS handshake
   - **Mitigation**: Accept this limitation for managed workflow benefits

---

## 3. mDNS/Bonjour Service Discovery

### Context

BoardingPass devices broadcast their availability via mDNS service name `_boardingpass._tcp`. The app must discover these devices automatically on the local network (FR-001, FR-005) to provide seamless device onboarding without requiring users to know IP addresses.

### Options Evaluated

#### 3.1 **react-native-zeroconf**

- **Description**: React Native module wrapping native mDNS/Bonjour implementations
- **Pros**:
  - Most popular React Native mDNS library
  - Native implementation (NSNetService on iOS, NsdManager on Android)
  - Active community and maintenance
  - Supports service browsing, resolving, and publishing
  - Event-driven API for service discovery lifecycle
- **Cons**:
  - Requires ejecting from Expo managed workflow
  - Behavioral differences between iOS and Android
  - Some reports of reliability issues on Android
  - Manual lifecycle management required
- **Security**:
  - Uses platform native APIs (secure)
  - No additional security considerations beyond network access

#### 3.2 **expo-network + Manual Service Discovery**

- **Description**: Use Expo's network APIs to manually discover devices
- **Pros**:
  - Stays within Expo managed workflow
  - Full control over discovery logic
- **Cons**:
  - Expo doesn't provide mDNS APIs
  - Would require IP scanning (unreliable, slow)
  - Cannot discover service-specific metadata
  - Poor user experience (long scan times)
- **Security**:
  - IP scanning may be flagged by network security tools
  - Less reliable device identification

#### 3.3 **Expo Config Plugin for react-native-zeroconf**

- **Description**: Use react-native-zeroconf with Expo config plugin for managed workflow
- **Pros**:
  - Maintains most Expo benefits (EAS Build, OTA updates)
  - Uses proven mDNS implementation
  - Simpler than full bare workflow
  - Config plugin handles native configuration
- **Cons**:
  - Requires Expo prebuild (not pure managed workflow)
  - Adds build complexity
  - Depends on community plugin maintenance
  - May lag behind expo-sdk updates
- **Security**:
  - Same as react-native-zeroconf option

#### 3.4 **Fallback to Well-Known IP Only**

- **Description**: Skip mDNS entirely, rely only on well-known IP address (192.168.1.100:9443)
- **Pros**:
  - Simple implementation
  - No native dependencies
  - Works in all network environments
- **Cons**:
  - Poor user experience (manual IP entry)
  - Doesn't meet FR-001 requirement
  - Requires network knowledge from users
  - Only works if device has expected IP
- **Security**:
  - No security concerns, but limited functionality

### Decision

**Chosen**: **Expo Config Plugin for react-native-zeroconf** with fallback to well-known IP

### Rationale

1. **Requirement Compliance**: Necessary to meet FR-001 (mDNS scanning) and FR-002 (fallback IP)
2. **Expo Benefits**: Maintains EAS Build, OTA updates, and most managed workflow benefits
3. **Native Performance**: Leverages platform-native mDNS implementations for reliability
4. **User Experience**: Automatic discovery provides best UX for primary use case (SC-001: discover within 10 seconds)
5. **Pragmatic**: Config plugin approach balances Expo benefits with native functionality needs
6. **Fallback**: Well-known IP provides safety net when mDNS unavailable (corporate networks)

This requires prebuild but maintains most Expo advantages while meeting core requirements.

### Implementation Notes

1. **Installation**:
   ```bash
   npx expo install react-native-zeroconf
   # Add config plugin to app.json
   ```

2. **Expo Config** (app.json):
   ```json
   {
     "expo": {
       "plugins": [
         [
           "react-native-zeroconf",
           {
             "nsdServiceType": "_boardingpass._tcp"
           }
         ]
       ]
     }
   }
   ```

3. **Service Discovery Hook**:
   ```typescript
   import Zeroconf from 'react-native-zeroconf';
   import { useEffect, useState } from 'react';

   interface DiscoveredDevice {
     name: string;
     host: string;
     port: number;
     addresses: string[];
     txt?: Record<string, string>;
   }

   function useDeviceDiscovery() {
     const [devices, setDevices] = useState<DiscoveredDevice[]>([]);
     const [scanning, setScanning] = useState(false);

     useEffect(() => {
       const zeroconf = new Zeroconf();

       zeroconf.on('start', () => setScanning(true));
       zeroconf.on('stop', () => setScanning(false));

       zeroconf.on('resolved', (service) => {
         setDevices(prev => {
           const exists = prev.find(d => d.name === service.name);
           if (exists) return prev;
           return [...prev, {
             name: service.name,
             host: service.host,
             port: service.port,
             addresses: service.addresses,
             txt: service.txt
           }];
         });
       });

       zeroconf.on('remove', (service) => {
         setDevices(prev => prev.filter(d => d.name !== service.name));
       });

       zeroconf.scan('_boardingpass._tcp', 'local.');

       return () => {
         zeroconf.stop();
       };
     }, []);

     return { devices, scanning };
   }
   ```

4. **Fallback IP Strategy**:
   ```typescript
   const FALLBACK_IP = '192.168.1.100';
   const FALLBACK_PORT = 9443;

   async function checkFallbackDevice(): Promise<DiscoveredDevice | null> {
     try {
       // Attempt to connect to well-known IP
       const response = await axios.get(
         `https://${FALLBACK_IP}:${FALLBACK_PORT}/health`,
         { timeout: 5000 }
       );

       if (response.status === 200) {
         return {
           name: 'Device (fallback)',
           host: FALLBACK_IP,
           port: FALLBACK_PORT,
           addresses: [FALLBACK_IP],
         };
       }
     } catch (error) {
       // Fallback device not available
     }
     return null;
   }
   ```

5. **Platform-Specific Considerations**:

   **iOS**:
   - Requires NSBonjourServices in Info.plist
   - Local network permission prompt on iOS 14+
   - Handle permission denial gracefully

   **Android**:
   - Requires ACCESS_FINE_LOCATION permission (for mDNS on Android 13+)
   - NsdManager can be unreliable on some devices
   - May need retry logic for failed discoveries

6. **Lifecycle Management**:
   ```typescript
   // Start scanning when app comes to foreground
   useEffect(() => {
     const subscription = AppState.addEventListener('change', nextAppState => {
       if (nextAppState === 'active') {
         // Resume scanning
       } else {
         // Pause scanning to save battery
       }
     });

     return () => subscription.remove();
   }, []);
   ```

7. **Error Handling**:
   - Network permission denied
   - mDNS not supported on network
   - Service resolution failures
   - Duplicate device names (use host/IP as secondary identifier)
   - Transient network issues (implement retry with backoff)

8. **Performance Considerations**:
   - Scan timeout (e.g., 30 seconds for initial scan)
   - Periodic refresh vs continuous scanning (balance battery vs freshness)
   - Debounce rapid service appearances/disappearances
   - Limit number of concurrent resolution attempts

9. **User Experience**:
   - Show scanning indicator while discovering
   - Display "No devices found" after scan timeout
   - Auto-refresh device list when new devices appear (FR-005)
   - Remove stale devices promptly when they go offline
   - Clearly distinguish mDNS discovered vs fallback devices

10. **Testing Strategy**:
    - Test on both iOS and Android physical devices
    - Test on WiFi networks with mDNS enabled/disabled
    - Test with multiple devices broadcasting simultaneously
    - Test device removal detection
    - Test fallback IP when mDNS unavailable
    - Test permission denial scenarios

---

## 4. QR Code Scanning

### Context

The app must support scanning connection codes via QR codes or barcodes (FR-008, FR-009) to simplify authentication. QR scanning is a critical UX feature that reduces user error and speeds up onboarding (SC-007: scan in under 3 seconds).

### Options Evaluated

#### 4.1 **react-native-vision-camera v4**

- **Description**: Modern camera library with built-in barcode scanning via ML Kit/Vision
- **Pros**:
  - Latest version (v4) with frame processors for real-time scanning
  - Native performance using platform ML frameworks
  - Supports QR codes, barcodes, and many formats
  - Excellent documentation and active maintenance
  - Frame processor API allows custom processing
  - Good TypeScript support
  - Handles camera permissions elegantly
- **Cons**:
  - Requires ejecting from Expo managed workflow (needs native modules)
  - Larger library size (includes full camera functionality)
  - Learning curve for frame processors API
  - Requires worklets for frame processing
- **Security**:
  - No inherent security concerns
  - Validates scanned data in JavaScript layer
  - Permission prompts handled by OS

#### 4.2 **expo-camera with expo-barcode-scanner**

- **Description**: Expo's built-in camera and barcode scanning modules
- **Pros**:
  - Native Expo integration (managed workflow compatible)
  - Simple API for barcode scanning
  - Good documentation within Expo ecosystem
  - Handles permissions through Expo APIs
  - Lighter weight than vision-camera
- **Cons**:
  - Less performant than vision-camera
  - Fewer advanced features
  - expo-barcode-scanner may be deprecated in future
  - Less active development compared to vision-camera
- **Security**:
  - Similar to vision-camera
  - Expo's permission handling is well-tested

#### 4.3 **expo-camera with vision-camera-code-scanner plugin**

- **Description**: Hybrid approach using vision-camera with Expo config plugin
- **Pros**:
  - Best performance from vision-camera
  - Maintain some Expo benefits via config plugin
  - Modern scanning capabilities
- **Cons**:
  - Requires prebuild (not pure managed workflow)
  - More complex setup than pure Expo
  - Adds build complexity
- **Security**:
  - Same as vision-camera

#### 4.4 **Manual Entry Only (No Scanning)**

- **Description**: Only allow manual connection code entry
- **Pros**:
  - No camera dependencies
  - Works on devices without cameras
  - Simpler implementation
- **Cons**:
  - Poor user experience (typing long codes)
  - Higher error rate (typos)
  - Doesn't meet FR-008, FR-009 requirements
  - Fails SC-007 success criteria
- **Security**:
  - No security implications

### Decision

**Chosen**: **expo-camera with expo-barcode-scanner** for initial implementation, with path to upgrade to vision-camera if needed

### Rationale

1. **Expo Compatibility**: Maintains managed workflow benefits for initial development velocity
2. **Requirement Compliance**: Meets FR-008 and FR-009 requirements for QR/barcode scanning
3. **Simplicity**: Simpler API reduces implementation time and complexity
4. **Sufficient Performance**: For QR code scanning (static codes), expo-barcode-scanner performance is adequate
5. **Upgrade Path**: Can migrate to vision-camera if performance becomes an issue
6. **Lower Risk**: Proven within Expo ecosystem, less chance of configuration issues

The decision prioritizes development velocity and Expo benefits while meeting requirements. If SC-007 (3-second scan time) cannot be met, we can migrate to vision-camera.

### Implementation Notes

1. **Installation**:
   ```bash
   npx expo install expo-camera expo-barcode-scanner
   ```

2. **Permissions**:
   ```typescript
   import { Camera } from 'expo-camera';

   async function requestCameraPermission(): Promise<boolean> {
     const { status } = await Camera.requestCameraPermissionsAsync();
     if (status !== 'granted') {
       // Show permission rationale dialog (FR-026)
       return false;
     }
     return true;
   }
   ```

3. **Scanner Component**:
   ```typescript
   import { BarCodeScanner } from 'expo-barcode-scanner';
   import { Camera, CameraType } from 'expo-camera';

   interface QRScannerProps {
     onScanned: (code: string) => void;
     onCancel: () => void;
   }

   function QRScanner({ onScanned, onCancel }: QRScannerProps) {
     const [hasPermission, setHasPermission] = useState<boolean | null>(null);
     const [scanned, setScanned] = useState(false);

     useEffect(() => {
       requestCameraPermission().then(setHasPermission);
     }, []);

     const handleBarCodeScanned = ({ type, data }: BarCodeScannedEvent) => {
       setScanned(true);
       // Validate QR code format (FR-027)
       if (isValidConnectionCode(data)) {
         onScanned(data);
       } else {
         // Show error: invalid QR code format
         setScanned(false);
       }
     };

     if (hasPermission === null) {
       return <LoadingView />;
     }

     if (hasPermission === false) {
       return <PermissionDeniedView onOpenSettings={openSettings} />;
     }

     return (
       <View style={styles.container}>
         <Camera
           type={CameraType.back}
           onBarCodeScanned={scanned ? undefined : handleBarCodeScanned}
           barCodeScannerSettings={{
             barCodeTypes: [BarCodeScanner.Constants.BarCodeType.qr],
           }}
           style={StyleSheet.absoluteFillObject}
         />
         <ScannerOverlay />
         <Button title="Cancel" onPress={onCancel} />
       </View>
     );
   }
   ```

4. **QR Code Format Validation**:
   ```typescript
   function isValidConnectionCode(data: string): boolean {
     // Define expected format (e.g., base64 string, specific length)
     // Match against BoardingPass connection code format
     const pattern = /^[A-Za-z0-9+/=]{32,}$/;  // Example: base64
     return pattern.test(data);
   }
   ```

5. **Scanner UI/UX**:
   - Display scanning frame overlay (guides user where to point camera)
   - Show scanning instructions ("Point camera at QR code")
   - Provide visual/haptic feedback when code detected
   - Cancel button to return to manual entry
   - Auto-close scanner on successful scan
   - Error message for invalid QR codes with retry option

6. **Permission Handling** (FR-026):
   ```typescript
   function PermissionDeniedView({ onOpenSettings }: Props) {
     return (
       <View style={styles.center}>
         <Icon name="camera-off" size={64} />
         <Text style={styles.title}>Camera Access Required</Text>
         <Text style={styles.body}>
           To scan QR codes, please allow camera access in your device settings.
         </Text>
         <Button title="Open Settings" onPress={onOpenSettings} />
         <Button title="Enter Code Manually" onPress={onManualEntry} />
       </View>
     );
   }

   function openSettings() {
     if (Platform.OS === 'ios') {
       Linking.openURL('app-settings:');
     } else {
       Linking.openSettings();
     }
   }
   ```

7. **Error Scenarios**:
   - Permission denied: Show rationale and link to settings
   - Camera unavailable: Fallback to manual entry
   - Invalid QR code: Show error, allow retry
   - Malformed data: Validate before using (FR-027)
   - Low light: Show message to improve lighting

8. **Performance Optimization**:
   - Limit barcode types to QR only (faster detection)
   - Debounce scan events to prevent multiple triggers
   - Release camera resources when scanner closed
   - Consider throttling frame processing if needed

9. **Accessibility**:
   - Provide manual entry alternative (required for users without cameras)
   - Voice-over support for scanner instructions
   - Sufficient contrast for overlay UI
   - Clear button labels and actions

10. **Testing**:
    - Test with various QR code sizes and distances
    - Test in different lighting conditions
    - Test with damaged/partial QR codes
    - Test permission denial flows
    - Measure scan time against SC-007 (3 seconds)
    - Test on multiple device models (camera quality varies)

11. **Migration Path to vision-camera** (if needed):
    ```typescript
    // If performance inadequate, switch to vision-camera with frame processors
    import { useCameraDevice, useCodeScanner } from 'react-native-vision-camera';

    const codeScanner = useCodeScanner({
      codeTypes: ['qr'],
      onCodeScanned: (codes) => {
        // Process codes with better performance
      }
    });
    ```

---

## 5. Secure Storage for Session Tokens and Certificate Pins

### Context

The app must securely store sensitive data including session tokens (FR-012) and certificate fingerprints. This data must be protected at rest, cleared when appropriate (FR-013), and persist across app restarts. Storage security is critical to prevent session hijacking and MITM attacks.

### Options Evaluated

#### 5.1 **expo-secure-store**

- **Description**: Expo's secure storage API using iOS Keychain and Android Keystore
- **Pros**:
  - Native Expo integration (managed workflow compatible)
  - Encrypted storage on both platforms
  - Simple key-value API
  - Backed by iOS Keychain (hardware encryption)
  - Backed by Android Keystore (hardware-backed when available)
  - Data persists across app updates
  - Data cleared on app uninstall
  - No size limits for typical use cases
- **Cons**:
  - Only available on device (not in simulators on iOS < 13)
  - Requires device unlock to access (can be limitation or feature)
  - Limited to string values (must serialize objects)
- **Security**:
  - High security via platform native secure storage
  - Hardware encryption where available
  - Protected by device passcode/biometrics
  - Inaccessible to other apps

#### 5.2 **react-native-keychain**

- **Description**: Direct access to iOS Keychain and Android Keystore
- **Pros**:
  - More configuration options than expo-secure-store
  - Supports biometric authentication requirements
  - Can specify accessibility levels
  - More control over encryption parameters
- **Cons**:
  - Requires ejecting from Expo managed workflow
  - More complex API
  - Additional configuration overhead
  - Higher maintenance burden
- **Security**:
  - Same underlying security as expo-secure-store
  - Additional options may not be needed for this use case

#### 5.3 **AsyncStorage with Encryption**

- **Description**: Use AsyncStorage with application-layer encryption (expo-crypto)
- **Pros**:
  - Works in simulators and all environments
  - Full control over encryption
  - Can implement custom security policies
- **Cons**:
  - Must implement encryption correctly (risk)
  - Need to manage encryption keys (where to store?)
  - Weaker than hardware-backed storage
  - More complex implementation
  - Higher risk of implementation errors
- **Security**:
  - Security depends on implementation quality
  - Key management is problematic (where to store encryption key securely?)
  - Vulnerable if device is rooted/jailbroken

#### 5.4 **In-Memory Storage Only**

- **Description**: Keep session tokens and pins only in memory
- **Pros**:
  - No persistence = no storage security concerns
  - Simple implementation
  - Automatic clearing on app termination
- **Cons**:
  - Poor user experience (re-authenticate on every app launch)
  - Defeats purpose of session tokens
  - Doesn't meet FR-012 requirement (store tokens)
  - Certificate pins lost on app restart
- **Security**:
  - High security (no persistent storage to compromise)
  - Impractical for actual use

### Decision

**Chosen**: **expo-secure-store** for session tokens and certificate pins

### Rationale

1. **Platform Security**: Leverages iOS Keychain and Android Keystore for hardware-backed encryption
2. **Expo Compatibility**: Works within managed workflow, no ejecting required
3. **Simplicity**: Simple API reduces implementation errors and security vulnerabilities
4. **Sufficient**: Provides all necessary security features without complexity of react-native-keychain
5. **Appropriate**: Matches threat model (protect against device theft, malware)
6. **Requirement Compliance**: Meets FR-012 (secure token storage) and FR-013 (clear when needed)

The managed workflow compatibility and simplicity outweigh the additional configurability of react-native-keychain, which isn't needed for this use case.

### Implementation Notes

1. **Installation**:
   ```bash
   npx expo install expo-secure-store
   ```

2. **Session Token Storage**:
   ```typescript
   import * as SecureStore from 'expo-secure-store';

   interface SessionToken {
     token: string;
     deviceId: string;
     expiresAt: string;  // ISO 8601 timestamp
   }

   async function storeSessionToken(
     deviceId: string,
     token: string,
     expiresAt: Date
   ): Promise<void> {
     const sessionData: SessionToken = {
       token,
       deviceId,
       expiresAt: expiresAt.toISOString(),
     };

     await SecureStore.setItemAsync(
       `session_${deviceId}`,
       JSON.stringify(sessionData)
     );
   }

   async function getSessionToken(deviceId: string): Promise<string | null> {
     const data = await SecureStore.getItemAsync(`session_${deviceId}`);
     if (!data) return null;

     const session: SessionToken = JSON.parse(data);

     // Check expiration (FR-037)
     if (new Date(session.expiresAt) < new Date()) {
       await clearSessionToken(deviceId);
       return null;
     }

     return session.token;
   }

   async function clearSessionToken(deviceId: string): Promise<void> {
     await SecureStore.deleteItemAsync(`session_${deviceId}`);
   }
   ```

3. **Certificate Pin Storage**:
   ```typescript
   interface CertificatePin {
     deviceId: string;
     fingerprint: string;
     issuer: string;
     subject: string;
     validFrom: string;
     validTo: string;
     pinnedAt: string;
   }

   async function storeCertificatePin(pin: CertificatePin): Promise<void> {
     await SecureStore.setItemAsync(
       `cert_pin_${pin.deviceId}`,
       JSON.stringify(pin)
     );
   }

   async function getCertificatePin(deviceId: string): Promise<CertificatePin | null> {
     const data = await SecureStore.getItemAsync(`cert_pin_${deviceId}`);
     return data ? JSON.parse(data) : null;
   }

   async function updateCertificatePin(
     deviceId: string,
     newPin: CertificatePin
   ): Promise<void> {
     // When certificate changes (FR-035)
     await storeCertificatePin(newPin);
   }
   ```

4. **Storage Key Naming Convention**:
   ```typescript
   // Use consistent prefixes for organization
   const STORAGE_KEYS = {
     sessionToken: (deviceId: string) => `session_${deviceId}`,
     certPin: (deviceId: string) => `cert_pin_${deviceId}`,
     deviceMeta: (deviceId: string) => `device_meta_${deviceId}`,
   };
   ```

5. **Data Lifecycle Management**:
   ```typescript
   // Clear all stored data (e.g., on logout, error recovery)
   async function clearAllDeviceData(deviceId: string): Promise<void> {
     await Promise.all([
       clearSessionToken(deviceId),
       SecureStore.deleteItemAsync(STORAGE_KEYS.certPin(deviceId)),
       SecureStore.deleteItemAsync(STORAGE_KEYS.deviceMeta(deviceId)),
     ]);
   }

   // Clear expired sessions periodically
   async function cleanupExpiredSessions(): Promise<void> {
     // List all session keys (would need to track separately)
     // Check expiration for each
     // Delete expired ones
   }
   ```

6. **Error Handling**:
   ```typescript
   async function safeGetSessionToken(deviceId: string): Promise<string | null> {
     try {
       return await getSessionToken(deviceId);
     } catch (error) {
       console.error('Failed to retrieve session token:', error);
       // Log error but don't expose to user (FR-029)
       return null;
     }
   }
   ```

7. **Platform-Specific Considerations**:

   **iOS**:
   - Keychain items persist across app reinstalls (by design)
   - May need manual cleanup in rare cases
   - Simulator limitations on older iOS versions

   **Android**:
   - Keystore cleared on app uninstall (desired behavior)
   - Hardware-backed when available, software fallback otherwise
   - May be affected by device manufacturer implementations

8. **Security Best Practices**:
   - Never log stored values (FR-029)
   - Clear sensitive data on authentication failure (FR-013)
   - Validate data format when retrieving (don't trust stored data blindly)
   - Use device ID as part of key to namespace storage
   - Consider time-based expiration for certificate pins (e.g., re-verify after 90 days)

9. **Testing Considerations**:
   - Test on physical devices (simulators may behave differently)
   - Test storage persistence across app restarts
   - Test cleanup on authentication failure
   - Test expiration checking logic
   - Test with invalid/corrupted stored data
   - Test storage limits (unlikely to hit, but validate)

10. **Migration and Versioning**:
    ```typescript
    interface StoredData {
      version: number;
      data: any;
    }

    const CURRENT_VERSION = 1;

    async function migrateStoredData(key: string): Promise<void> {
      const raw = await SecureStore.getItemAsync(key);
      if (!raw) return;

      try {
        const stored: StoredData = JSON.parse(raw);
        if (stored.version === CURRENT_VERSION) return;

        // Perform migration logic based on version
        // ...

        await SecureStore.setItemAsync(key, JSON.stringify({
          version: CURRENT_VERSION,
          data: migratedData,
        }));
      } catch {
        // Invalid format, clear and start fresh
        await SecureStore.deleteItemAsync(key);
      }
    }
    ```

11. **Storage Quotas**:
    - expo-secure-store has a 2KB limit per key on Android (iOS is more generous)
    - Session tokens are typically small (<500 bytes)
    - Certificate pins are small (<1KB)
    - If hitting limits, consider compression or restructuring data

---

## 6. TypeScript Type Generation from Go Protocol Types

### Context

The app must communicate with BoardingPass API using the protocol types defined in `pkg/protocol/types.go`. Keeping TypeScript types in sync with Go structs is critical for type safety and preventing runtime errors. Manual synchronization is error-prone and doesn't scale.

### Options Evaluated

#### 6.1 **Manual TypeScript Type Definitions**

- **Description**: Manually create and maintain TypeScript interfaces matching Go structs
- **Pros**:
  - Full control over type definitions
  - No tooling dependencies
  - Can optimize TypeScript types for frontend use (e.g., Date objects vs strings)
  - Simple to understand
- **Cons**:
  - Manual synchronization required on every protocol change
  - High risk of drift between Go and TypeScript
  - No compile-time verification of sync
  - Error-prone (typos, missed fields)
  - Doesn't scale with protocol evolution
- **Security**:
  - No direct security implications
  - Type mismatches could cause runtime errors

#### 6.2 **typescriptify (Go package)**

- **Description**: Go package that generates TypeScript interfaces from Go structs
- **Pros**:
  - Go-native tooling (fits into Go build pipeline)
  - Supports struct tags for customization
  - Can handle embedded structs and complex types
  - One-way generation (Go as source of truth)
  - Can integrate with go generate
- **Cons**:
  - Requires Go development environment for frontend devs
  - Limited community adoption
  - May not handle all TypeScript edge cases
  - Generated types may need manual tweaking
- **Security**:
  - No security implications

#### 6.3 **OpenAPI Code Generation**

- **Description**: Generate TypeScript types from OpenAPI specification
- **Pros**:
  - OpenAPI spec already exists (specs/001-boardingpass-api/contracts/openapi.yaml)
  - Industry-standard approach
  - Can generate both types AND API client code
  - Many mature tools available (openapi-typescript, openapi-generator)
  - Spec serves as contract between frontend/backend
  - Can generate validation schemas (Zod, Yup)
- **Cons**:
  - Requires keeping OpenAPI spec in sync with Go code
  - Generated code can be verbose
  - May generate more than needed
  - Some tools have heavy dependencies
- **Security**:
  - No direct security implications
  - Generated validators can improve security

#### 6.4 **json-schema-to-typescript**

- **Description**: Generate TypeScript types from JSON Schema
- **Pros**:
  - JSON Schema can be generated from Go structs
  - Clean type generation
  - Good TypeScript output
  - Can include documentation in types
- **Cons**:
  - Two-step process (Go → JSON Schema → TypeScript)
  - Requires maintaining JSON Schema generation
  - Adds intermediate format
- **Security**:
  - No security implications

#### 6.5 **Shared Contract in JSON/YAML**

- **Description**: Define protocol in neutral format (JSON/YAML), generate both Go and TypeScript
- **Pros**:
  - Single source of truth
  - Language-agnostic contract
  - Both Go and TypeScript generated from same source
- **Cons**:
  - Requires significant refactoring
  - Neither language is source of truth (contract file is)
  - Violates existing architecture (Go types already defined)
  - High migration cost
- **Security**:
  - No security implications

### Decision

**Chosen**: **OpenAPI Code Generation** using openapi-typescript with existing OpenAPI spec

### Rationale

1. **Existing Asset**: OpenAPI spec already exists and is maintained for API contract testing
2. **Single Source of Truth**: OpenAPI spec serves as authoritative contract between backend and frontend
3. **Comprehensive**: Generates both types and can generate API client scaffolding
4. **Industry Standard**: OpenAPI is widely adopted, tooling is mature and well-maintained
5. **Validation**: Can generate runtime validators to catch API mismatches early
6. **Documentation**: OpenAPI spec provides self-documenting API
7. **Minimal Dependencies**: openapi-typescript has minimal runtime dependencies

This approach leverages existing investment in OpenAPI spec and provides strongest contract guarantees.

### Implementation Notes

1. **Installation**:
   ```bash
   npm install --save-dev openapi-typescript
   npm install --save-dev @redocly/openapi-cli  # For spec validation
   ```

2. **Package.json Scripts**:
   ```json
   {
     "scripts": {
       "generate:types": "openapi-typescript ../../specs/001-boardingpass-api/contracts/openapi.yaml -o ./src/api/generated/types.ts",
       "validate:spec": "openapi lint ../../specs/001-boardingpass-api/contracts/openapi.yaml",
       "prebuild": "npm run validate:spec && npm run generate:types"
     }
   }
   ```

3. **Generated Types Usage**:
   ```typescript
   // src/api/generated/types.ts (generated)
   export interface paths {
     '/auth/srp/init': {
       post: {
         requestBody: {
           content: {
             'application/json': components['schemas']['SRPInitRequest'];
           };
         };
         responses: {
           200: {
             content: {
               'application/json': components['schemas']['SRPInitResponse'];
             };
           };
         };
       };
     };
     // ... other endpoints
   }

   export interface components {
     schemas: {
       SRPInitRequest: {
         username: string;
         A: string;
       };
       SRPInitResponse: {
         salt: string;
         b: string;
       };
       // ... other schemas
     };
   }
   ```

4. **API Client with Generated Types**:
   ```typescript
   import type { paths, components } from './generated/types';
   import axios, { AxiosInstance } from 'axios';

   type SRPInitRequest = components['schemas']['SRPInitRequest'];
   type SRPInitResponse = components['schemas']['SRPInitResponse'];
   type SRPVerifyRequest = components['schemas']['SRPVerifyRequest'];
   type SRPVerifyResponse = components['schemas']['SRPVerifyResponse'];
   type SystemInfo = components['schemas']['SystemInfo'];
   type NetworkConfig = components['schemas']['NetworkConfig'];

   class BoardingPassClient {
     private client: AxiosInstance;

     constructor(baseURL: string, sessionToken?: string) {
       this.client = axios.create({
         baseURL,
         headers: {
           'Content-Type': 'application/json',
           ...(sessionToken && { 'Authorization': `Bearer ${sessionToken}` }),
         },
       });
     }

     async srpInit(request: SRPInitRequest): Promise<SRPInitResponse> {
       const response = await this.client.post<SRPInitResponse>(
         '/auth/srp/init',
         request
       );
       return response.data;
     }

     async srpVerify(request: SRPVerifyRequest): Promise<SRPVerifyResponse> {
       const response = await this.client.post<SRPVerifyResponse>(
         '/auth/srp/verify',
         request
       );
       return response.data;
     }

     async getSystemInfo(): Promise<SystemInfo> {
       const response = await this.client.get<SystemInfo>('/info');
       return response.data;
     }

     async getNetworkConfig(): Promise<NetworkConfig> {
       const response = await this.client.get<NetworkConfig>('/network');
       return response.data;
     }
   }
   ```

5. **Type Safety Benefits**:
   ```typescript
   // TypeScript catches mismatches at compile time
   const client = new BoardingPassClient('https://device.local:9443');

   // Type error if request doesn't match schema
   await client.srpInit({
     username: 'device',
     A: 'base64string',
     // @ts-expect-error - extra field not in schema
     invalidField: 'value',
   });

   // Response type is automatically inferred
   const response = await client.getSystemInfo();
   // response.tpm.present is typed as boolean
   // response.board.manufacturer is typed as string
   ```

6. **Runtime Validation** (optional but recommended):
   ```bash
   npm install zod openapi-zod-client
   ```

   ```typescript
   // Generate Zod schemas from OpenAPI
   // Can use for runtime validation
   import { z } from 'zod';

   const SRPInitResponseSchema = z.object({
     salt: z.string(),
     b: z.string(),
   });

   // Validate API responses at runtime
   const response = await client.srpInit(request);
   const validated = SRPInitResponseSchema.parse(response);
   ```

7. **Workflow Integration**:
   ```
   1. Backend developer updates Go protocol types
   2. Backend developer updates OpenAPI spec to match
   3. Contract tests validate Go code matches OpenAPI spec
   4. Frontend developer pulls changes
   5. Frontend developer runs `npm run generate:types`
   6. TypeScript compiler catches any breaking changes
   7. Frontend developer updates code to match new types
   ```

8. **CI/CD Integration**:
   ```yaml
   # .github/workflows/frontend-ci.yml
   name: Frontend CI
   on: [push, pull_request]
   jobs:
     typecheck:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v3
         - uses: actions/setup-node@v3
         - run: npm ci
         - run: npm run validate:spec
         - run: npm run generate:types
         - run: npm run typecheck
   ```

9. **Handling Breaking Changes**:
   ```typescript
   // When API evolves, types automatically reflect changes
   // Optional fields become required → TypeScript error
   // New fields added → TypeScript suggests them
   // Fields removed → TypeScript error on usage

   // Example: If BoardingPass adds `hostname` field to SystemInfo
   interface SystemInfo {
     tpm: TPMInfo;
     board: BoardInfo;
     cpu: CPUInfo;
     os: OSInfo;
     hostname: string;  // New field
   }

   // Existing code still compiles (new field is additive)
   // Can now access response.hostname with type safety
   ```

10. **Documentation Generation**:
    ```bash
    npm install --save-dev @redocly/cli
    npx redocly build-docs ../../specs/001-boardingpass-api/contracts/openapi.yaml
    ```
    - Generates HTML documentation from OpenAPI spec
    - Developers can reference for API usage
    - Keeps docs in sync with implementation

11. **Alternative: Custom Type Generator** (if OpenAPI has limitations):
    ```typescript
    // build/generate-types.ts
    // Can write custom generator that reads OpenAPI and generates
    // optimized types for specific use cases

    import yaml from 'yaml';
    import fs from 'fs';

    const spec = yaml.parse(fs.readFileSync('openapi.yaml', 'utf8'));
    // Custom generation logic
    // Output optimized types
    ```

12. **Testing Generated Types**:
    ```typescript
    // Ensure generated types match runtime API responses
    import { describe, it, expect } from '@jest/globals';
    import type { SystemInfo } from './generated/types';

    describe('API Type Conformance', () => {
      it('should match SystemInfo schema', async () => {
        const response = await client.getSystemInfo();

        // Runtime checks (if using Zod validators)
        expect(() => SystemInfoSchema.parse(response)).not.toThrow();

        // Type-level checks (compile-time)
        const info: SystemInfo = response;  // Should compile without error
      });
    });
    ```

---

## 7. Expo vs React Native CLI

### Context

The specification assumes Expo managed workflow (see Operational Assumptions). However, several required features (mDNS discovery, advanced camera, possibly certificate pinning) may require native modules. This research validates the Expo decision and explores hybrid approaches.

### Options Evaluated

#### 7.1 **Expo Managed Workflow (Pure)**

- **Description**: Use only Expo APIs and Expo-compatible libraries
- **Pros**:
  - Fastest development velocity
  - OTA updates without app store approval
  - EAS Build for cloud builds
  - Simplified native configuration
  - No need for Xcode/Android Studio
  - Easier CI/CD setup
  - Good developer experience with Expo Go
- **Cons**:
  - Limited to Expo-compatible libraries
  - Cannot use arbitrary native modules
  - mDNS (react-native-zeroconf) requires native module
  - Advanced camera features may be limited
  - Certificate pinning complexity
- **Alignment with Requirements**:
  - ❌ FR-001 (mDNS) - react-native-zeroconf not available
  - ✅ FR-008/009 (QR scanning) - expo-camera works
  - ⚠️ FR-031 (cert pinning) - Limited options
  - ✅ All other requirements met

#### 7.2 **Expo Prebuild (Hybrid)**

- **Description**: Use Expo with config plugins for native modules, prebuild to generate native projects
- **Pros**:
  - Maintains most Expo benefits (EAS Build, updates)
  - Can use native modules via config plugins
  - Expo manages native configuration via plugins
  - Can use react-native-zeroconf via config plugin
  - Better than bare workflow for native complexity
  - Still gets Expo ecosystem benefits
- **Cons**:
  - More complex than pure managed workflow
  - Need to understand native project structure
  - Config plugins may lag library updates
  - Some manual native configuration may be needed
  - Cannot use Expo Go (must use dev builds)
- **Alignment with Requirements**:
  - ✅ FR-001 (mDNS) - react-native-zeroconf with config plugin
  - ✅ FR-008/009 (QR scanning) - expo-camera or vision-camera
  - ✅ FR-031 (cert pinning) - More options available
  - ✅ All requirements met

#### 7.3 **React Native CLI (Bare Workflow)**

- **Description**: Standard React Native without Expo tooling
- **Pros**:
  - Full control over native code
  - No restrictions on native modules
  - Can use any React Native library
  - More flexibility for advanced features
  - No Expo abstractions or limitations
- **Cons**:
  - Slower development velocity
  - Manual native configuration (Info.plist, AndroidManifest.xml, etc.)
  - No OTA updates (without setting up CodePush)
  - More complex CI/CD setup
  - Need Xcode and Android Studio
  - More native code to maintain
  - Steeper learning curve
- **Alignment with Requirements**:
  - ✅ All requirements met
  - ⚠️ Slower to implement

#### 7.4 **Expo with Selective Ejecting**

- **Description**: Start with Expo, eject only when hitting limitations
- **Pros**:
  - Start fast with managed workflow
  - Migrate when necessary
  - Learn what's actually needed before ejecting
- **Cons**:
  - Ejecting is one-way (difficult to reverse)
  - May require refactoring
  - Delayed discovery of blockers
  - Potential rework
- **Alignment with Requirements**:
  - ⚠️ Risk of discovering blockers late

### Decision

**Chosen**: **Expo Prebuild (Hybrid)** with config plugins from day one

### Rationale

1. **Requirements Alignment**: Can meet all requirements (FR-001 mDNS, FR-008/009 QR, FR-031 cert pinning)
2. **Development Velocity**: Maintains most Expo benefits while enabling necessary native features
3. **Balanced Approach**: Best of both worlds - Expo DX + native capabilities
4. **Risk Mitigation**: No late discovery of blockers requiring ejection/rewrite
5. **EAS Benefits**: Retains EAS Build, EAS Update (with caveats), and Expo ecosystem
6. **Maintainability**: Expo manages native configuration via plugins, reducing manual native code
7. **Team Skill Alignment**: Lower native expertise required compared to bare workflow
8. **CI/CD**: Simpler than bare workflow, more flexible than pure managed

While not as simple as pure managed workflow, prebuild approach is necessary to meet mDNS requirement (FR-001) and provides best balance.

### Implementation Notes

1. **Project Initialization**:
   ```bash
   npx create-expo-app@latest boardingpass-mobile --template blank-typescript
   cd boardingpass-mobile
   ```

2. **App Configuration** (app.json):
   ```json
   {
     "expo": {
       "name": "BoardingPass",
       "slug": "boardingpass-mobile",
       "version": "1.0.0",
       "orientation": "portrait",
       "icon": "./assets/icon.png",
       "userInterfaceStyle": "automatic",
       "splash": {
         "image": "./assets/splash.png",
         "resizeMode": "contain",
         "backgroundColor": "#ffffff"
       },
       "ios": {
         "supportsTablet": true,
         "bundleIdentifier": "com.yourcompany.boardingpass",
         "infoPlist": {
           "NSCameraUsageDescription": "Camera access is required to scan QR codes for device onboarding.",
           "NSLocalNetworkUsageDescription": "Network access is required to discover devices on your local network.",
           "NSBonjourServices": ["_boardingpass._tcp"]
         }
       },
       "android": {
         "adaptiveIcon": {
           "foregroundImage": "./assets/adaptive-icon.png",
           "backgroundColor": "#ffffff"
         },
         "package": "com.yourcompany.boardingpass",
         "permissions": [
           "CAMERA",
           "ACCESS_FINE_LOCATION",
           "CHANGE_WIFI_MULTICAST_STATE",
           "ACCESS_NETWORK_STATE"
         ]
       },
       "plugins": [
         [
           "expo-camera",
           {
             "cameraPermission": "Camera access is required to scan QR codes for device onboarding."
           }
         ],
         [
           "react-native-zeroconf",
           {
             "nsdServiceType": "_boardingpass._tcp"
           }
         ]
       ]
     }
   }
   ```

3. **Native Module Installation**:
   ```bash
   npx expo install expo-camera expo-secure-store expo-crypto
   npx expo install react-native-zeroconf
   npm install axios react-native-paper
   npm install --save-dev @types/react-native
   ```

4. **Prebuild Workflow**:
   ```bash
   # Generate native projects from config
   npx expo prebuild

   # Creates:
   # - ios/ directory with Xcode project
   # - android/ directory with Android Studio project

   # Run on iOS
   npx expo run:ios

   # Run on Android
   npx expo run:android
   ```

5. **Development Builds** (required for config plugins):
   ```bash
   # Cannot use Expo Go with custom native modules
   # Must use development builds

   npx expo install expo-dev-client

   # Build development client
   eas build --profile development --platform ios
   eas build --profile development --platform android

   # Or build locally
   npx expo run:ios
   npx expo run:android
   ```

6. **EAS Configuration** (eas.json):
   ```json
   {
     "cli": {
       "version": ">= 5.0.0"
     },
     "build": {
       "development": {
         "developmentClient": true,
         "distribution": "internal"
       },
       "preview": {
         "distribution": "internal",
         "ios": {
           "simulator": true
         }
       },
       "production": {
         "autoIncrement": true
       }
     },
     "submit": {
       "production": {}
     }
   }
   ```

7. **Managing Native Code**:
   ```
   - ios/ and android/ directories are generated by prebuild
   - Add to .gitignore (regenerated from app.json + plugins)
   - Config plugins manage most native configuration
   - For custom native code, create config plugin
   - Run prebuild after app.json changes
   ```

8. **Git Strategy**:
   ```
   # .gitignore
   ios/
   android/

   # Native projects are generated, not committed
   # Commit app.json and config plugins instead
   # Team members run prebuild to generate native projects
   ```

9. **Updates Strategy**:
   ```typescript
   import * as Updates from 'expo-updates';

   // EAS Update works with custom native code
   // Can push JS updates without rebuilding
   // Native code changes require new build

   useEffect(() => {
     async function checkForUpdates() {
       const update = await Updates.checkForUpdateAsync();
       if (update.isAvailable) {
         await Updates.fetchUpdateAsync();
         await Updates.reloadAsync();
       }
     }
     checkForUpdates();
   }, []);
   ```

10. **Custom Config Plugin** (if needed):
    ```typescript
    // plugins/custom-plugin.js
    const { withInfoPlist, withAndroidManifest } = require('@expo/config-plugins');

    module.exports = function withCustomConfig(config) {
      // iOS configuration
      config = withInfoPlist(config, (config) => {
        config.modResults['CustomKey'] = 'CustomValue';
        return config;
      });

      // Android configuration
      config = withAndroidManifest(config, (config) => {
        // Modify AndroidManifest.xml
        return config;
      });

      return config;
    };
    ```

11. **Testing Strategy**:
    - Use Jest for unit tests (Expo compatible)
    - Use React Native Testing Library for component tests
    - Use Detox for E2E tests (works with prebuild)
    - Test on both iOS and Android physical devices
    - Test native module functionality specifically

12. **CI/CD Pipeline**:
    ```yaml
    # .github/workflows/ci.yml
    name: CI
    on: [push, pull_request]
    jobs:
      build:
        runs-on: ubuntu-latest
        steps:
          - uses: actions/checkout@v3
          - uses: actions/setup-node@v3
          - run: npm ci
          - run: npm run lint
          - run: npm test
          - run: npx expo-doctor  # Check Expo configuration

      build-ios:
        runs-on: macos-latest
        steps:
          - uses: actions/checkout@v3
          - uses: actions/setup-node@v3
          - run: npm ci
          - run: npx expo prebuild --platform ios
          - run: npx expo run:ios --configuration Release --no-bundler

      build-android:
        runs-on: ubuntu-latest
        steps:
          - uses: actions/checkout@v3
          - uses: actions/setup-node@v3
          - run: npm ci
          - run: npx expo prebuild --platform android
          - run: npx expo run:android --variant release --no-bundler
    ```

13. **Migration Path**:
    - If Expo becomes too limiting: Can eject to bare workflow
    - Have full native projects (ios/, android/) after prebuild
    - Can gradually remove Expo dependencies if needed
    - Low risk of being locked in

---

## 8. State Management Pattern

### Context

The app needs to manage complex state including device discovery, authentication sessions, device information, navigation, and error states. State management choice impacts code organization, maintainability, performance, and developer experience.

### Options Evaluated

#### 8.1 **React Context + useReducer**

- **Description**: Built-in React state management using Context API and useReducer hook
- **Pros**:
  - No external dependencies (aligns with minimal dependencies principle)
  - Native to React, no new concepts to learn
  - Good for medium complexity apps
  - Type-safe with TypeScript
  - Fine-grained subscriptions possible with multiple contexts
  - Sufficient for this app's complexity
- **Cons**:
  - Can lead to prop drilling if not structured well
  - May cause unnecessary re-renders without optimization
  - More boilerplate than specialized libraries
  - No built-in devtools
  - Requires manual optimization (useMemo, useCallback)
- **Complexity Match**:
  - ✅ Sufficient for 4 user stories with moderate state needs
  - ✅ Can handle auth state, device list, navigation state

#### 8.2 **Redux Toolkit**

- **Description**: Modern Redux with simplified API via Redux Toolkit
- **Pros**:
  - Industry standard with large ecosystem
  - Excellent devtools (time travel debugging)
  - Predictable state updates
  - Great TypeScript support
  - Middleware for async operations (thunks, sagas)
  - Well-documented patterns
- **Cons**:
  - Additional dependency (violates minimal dependencies principle)
  - Overkill for this app's complexity
  - More boilerplate even with Toolkit
  - Steeper learning curve
  - Performance overhead for simple use cases
- **Complexity Match**:
  - ⚠️ Over-engineered for 4 user stories
  - Would be justified for multi-device management, history, etc. (out of scope)

#### 8.3 **Zustand**

- **Description**: Lightweight state management with hooks-based API
- **Pros**:
  - Minimal boilerplate
  - Excellent TypeScript support
  - Small bundle size (<1KB)
  - Simple, intuitive API
  - No provider wrapper needed
  - Built-in devtools support
  - Good performance (no unnecessary re-renders)
  - Can subscribe to specific state slices
- **Cons**:
  - Additional dependency
  - Less established than Redux (smaller community)
  - Fewer middleware options
  - Less opinionated (could be pro or con)
- **Complexity Match**:
  - ✅ Good match for this app's complexity
  - ✅ Simple enough for 4 user stories, scalable if needed

#### 8.4 **MobX**

- **Description**: Reactive state management using observables
- **Pros**:
  - Less boilerplate than Redux
  - Automatic dependency tracking
  - Good performance
  - Intuitive for OOP developers
- **Cons**:
  - Additional dependency
  - Different mental model (observables)
  - Decorators syntax (may require config)
  - Less popular in React Native community
  - More "magic" (less explicit than other options)
- **Complexity Match**:
  - ⚠️ Adequate but introduces unnecessary complexity

#### 8.5 **Jotai / Recoil**

- **Description**: Atomic state management libraries
- **Pros**:
  - Fine-grained reactivity
  - Good TypeScript support
  - Minimal boilerplate
  - Good for derived state
- **Cons**:
  - Additional dependencies
  - Newer, less proven in production
  - Smaller communities
  - Different mental model
- **Complexity Match**:
  - ⚠️ Interesting but unnecessary for this app

### Decision

**Chosen**: **React Context + useReducer** with structured state slicing

### Rationale

1. **Minimal Dependencies**: Aligns with constitution's minimal dependencies principle - uses only React built-ins
2. **Sufficient Complexity**: App has 4 user stories with moderate state needs - Context is sufficient
3. **Type Safety**: Excellent TypeScript support with discriminated unions for reducer actions
4. **Learning Curve**: Team already knows React, no new concepts needed
5. **Performance**: Adequate with proper context splitting and memoization
6. **No Over-Engineering**: YAGNI principle - don't add Redux for future features not yet needed
7. **Standard React**: Follows standard React patterns, maintainable by any React developer

If app expands to multi-device management, history, offline sync, etc., can migrate to Redux Toolkit. For current scope, Context is appropriate.

### Implementation Notes

1. **State Structure**:
   ```typescript
   // src/state/types.ts

   export interface AppState {
     discovery: DiscoveryState;
     auth: AuthState;
     device: DeviceState;
     ui: UIState;
   }

   export interface DiscoveryState {
     devices: DiscoveredDevice[];
     scanning: boolean;
     error: string | null;
   }

   export interface AuthState {
     sessionToken: string | null;
     authenticating: boolean;
     error: string | null;
   }

   export interface DeviceState {
     selectedDeviceId: string | null;
     systemInfo: SystemInfo | null;
     networkConfig: NetworkConfig | null;
     loading: boolean;
     error: string | null;
   }

   export interface UIState {
     showQRScanner: boolean;
     showCertificateDialog: boolean;
     certificateToReview: CertificateInfo | null;
   }
   ```

2. **Actions with Discriminated Unions**:
   ```typescript
   // src/state/actions.ts

   export type AppAction =
     | { type: 'DISCOVERY_START' }
     | { type: 'DISCOVERY_SUCCESS'; payload: DiscoveredDevice[] }
     | { type: 'DISCOVERY_ERROR'; payload: string }
     | { type: 'DEVICE_ADDED'; payload: DiscoveredDevice }
     | { type: 'DEVICE_REMOVED'; payload: string }
     | { type: 'AUTH_START' }
     | { type: 'AUTH_SUCCESS'; payload: string }
     | { type: 'AUTH_ERROR'; payload: string }
     | { type: 'AUTH_CLEAR' }
     | { type: 'DEVICE_INFO_START' }
     | { type: 'DEVICE_INFO_SUCCESS'; payload: { systemInfo: SystemInfo; networkConfig: NetworkConfig } }
     | { type: 'DEVICE_INFO_ERROR'; payload: string }
     | { type: 'UI_SHOW_QR_SCANNER' }
     | { type: 'UI_HIDE_QR_SCANNER' }
     | { type: 'UI_SHOW_CERTIFICATE'; payload: CertificateInfo }
     | { type: 'UI_HIDE_CERTIFICATE' };
   ```

3. **Reducer**:
   ```typescript
   // src/state/reducer.ts

   export function appReducer(state: AppState, action: AppAction): AppState {
     switch (action.type) {
       case 'DISCOVERY_START':
         return {
           ...state,
           discovery: {
             ...state.discovery,
             scanning: true,
             error: null,
           },
         };

       case 'DISCOVERY_SUCCESS':
         return {
           ...state,
           discovery: {
             devices: action.payload,
             scanning: false,
             error: null,
           },
         };

       case 'DEVICE_ADDED':
         return {
           ...state,
           discovery: {
             ...state.discovery,
             devices: [...state.discovery.devices, action.payload],
           },
         };

       case 'DEVICE_REMOVED':
         return {
           ...state,
           discovery: {
             ...state.discovery,
             devices: state.discovery.devices.filter(d => d.name !== action.payload),
           },
         };

       case 'AUTH_START':
         return {
           ...state,
           auth: {
             ...state.auth,
             authenticating: true,
             error: null,
           },
         };

       case 'AUTH_SUCCESS':
         return {
           ...state,
           auth: {
             sessionToken: action.payload,
             authenticating: false,
             error: null,
           },
         };

       case 'AUTH_ERROR':
         return {
           ...state,
           auth: {
             sessionToken: null,
             authenticating: false,
             error: action.payload,
           },
         };

       case 'AUTH_CLEAR':
         return {
           ...state,
           auth: {
             sessionToken: null,
             authenticating: false,
             error: null,
           },
         };

       // ... other cases

       default:
         return state;
     }
   }
   ```

4. **Context Provider**:
   ```typescript
   // src/state/AppContext.tsx

   interface AppContextValue {
     state: AppState;
     dispatch: React.Dispatch<AppAction>;
   }

   const AppContext = createContext<AppContextValue | undefined>(undefined);

   export function AppProvider({ children }: { children: React.ReactNode }) {
     const [state, dispatch] = useReducer(appReducer, initialState);

     return (
       <AppContext.Provider value={{ state, dispatch }}>
         {children}
       </AppContext.Provider>
     );
   }

   export function useAppContext() {
     const context = useContext(AppContext);
     if (!context) {
       throw new Error('useAppContext must be used within AppProvider');
     }
     return context;
   }
   ```

5. **Optimized Selectors**:
   ```typescript
   // src/state/selectors.ts

   export function useDiscoveredDevices() {
     const { state } = useAppContext();
     return useMemo(() => state.discovery.devices, [state.discovery.devices]);
   }

   export function useIsScanning() {
     const { state } = useAppContext();
     return state.discovery.scanning;
   }

   export function useAuthState() {
     const { state } = useAppContext();
     return useMemo(
       () => ({
         isAuthenticated: !!state.auth.sessionToken,
         authenticating: state.auth.authenticating,
         error: state.auth.error,
       }),
       [state.auth]
     );
   }

   export function useDeviceInfo() {
     const { state } = useAppContext();
     return useMemo(
       () => ({
         systemInfo: state.device.systemInfo,
         networkConfig: state.device.networkConfig,
         loading: state.device.loading,
         error: state.device.error,
       }),
       [state.device]
     );
   }
   ```

6. **Action Creators (Optional)**:
   ```typescript
   // src/state/actionCreators.ts

   export const actions = {
     discovery: {
       start: (): AppAction => ({ type: 'DISCOVERY_START' }),
       success: (devices: DiscoveredDevice[]): AppAction => ({
         type: 'DISCOVERY_SUCCESS',
         payload: devices,
       }),
       error: (error: string): AppAction => ({
         type: 'DISCOVERY_ERROR',
         payload: error,
       }),
       addDevice: (device: DiscoveredDevice): AppAction => ({
         type: 'DEVICE_ADDED',
         payload: device,
       }),
       removeDevice: (deviceId: string): AppAction => ({
         type: 'DEVICE_REMOVED',
         payload: deviceId,
       }),
     },
     auth: {
       start: (): AppAction => ({ type: 'AUTH_START' }),
       success: (token: string): AppAction => ({
         type: 'AUTH_SUCCESS',
         payload: token,
       }),
       error: (error: string): AppAction => ({
         type: 'AUTH_ERROR',
         payload: error,
       }),
       clear: (): AppAction => ({ type: 'AUTH_CLEAR' }),
     },
     // ... other action creators
   };
   ```

7. **Usage in Components**:
   ```typescript
   // src/screens/DeviceListScreen.tsx

   function DeviceListScreen() {
     const { dispatch } = useAppContext();
     const devices = useDiscoveredDevices();
     const scanning = useIsScanning();

     const handleRefresh = useCallback(() => {
       dispatch(actions.discovery.start());
       // Trigger mDNS scan
     }, [dispatch]);

     return (
       <View>
         {scanning && <ActivityIndicator />}
         <FlatList
           data={devices}
           renderItem={({ item }) => <DeviceCard device={item} />}
           refreshing={scanning}
           onRefresh={handleRefresh}
         />
       </View>
     );
   }
   ```

8. **Async Operations**:
   ```typescript
   // src/hooks/useAuth.ts

   export function useAuth() {
     const { dispatch } = useAppContext();

     const authenticate = useCallback(async (
       deviceUrl: string,
       connectionCode: string
     ) => {
       dispatch(actions.auth.start());

       try {
         // SRP-6a authentication flow
         const sessionToken = await performSRPAuth(deviceUrl, connectionCode);

         // Store token securely
         await storeSessionToken(deviceId, sessionToken);

         dispatch(actions.auth.success(sessionToken));
         return true;
       } catch (error) {
         dispatch(actions.auth.error(error.message));
         return false;
       }
     }, [dispatch]);

     return { authenticate };
   }
   ```

9. **Context Splitting** (if performance becomes issue):
   ```typescript
   // Split into separate contexts to avoid unnecessary re-renders

   const DiscoveryContext = createContext<DiscoveryContextValue | undefined>(undefined);
   const AuthContext = createContext<AuthContextValue | undefined>(undefined);
   const DeviceContext = createContext<DeviceContextValue | undefined>(undefined);

   export function AppProvider({ children }: { children: React.ReactNode }) {
     return (
       <DiscoveryProvider>
         <AuthProvider>
           <DeviceProvider>
             {children}
           </DeviceProvider>
         </AuthProvider>
       </DiscoveryProvider>
     );
   }
   ```

10. **Testing**:
    ```typescript
    // src/state/__tests__/reducer.test.ts

    describe('appReducer', () => {
      it('should handle DISCOVERY_SUCCESS', () => {
        const state = initialState;
        const devices = [{ name: 'device1', host: '192.168.1.100' }];

        const newState = appReducer(state, {
          type: 'DISCOVERY_SUCCESS',
          payload: devices,
        });

        expect(newState.discovery.devices).toEqual(devices);
        expect(newState.discovery.scanning).toBe(false);
        expect(newState.discovery.error).toBeNull();
      });

      it('should handle AUTH_SUCCESS', () => {
        const state = initialState;
        const token = 'session-token-123';

        const newState = appReducer(state, {
          type: 'AUTH_SUCCESS',
          payload: token,
        });

        expect(newState.auth.sessionToken).toBe(token);
        expect(newState.auth.authenticating).toBe(false);
        expect(newState.auth.error).toBeNull();
      });
    });
    ```

11. **Performance Considerations**:
    - Use `useMemo` in selectors to prevent unnecessary re-renders
    - Use `useCallback` for action creators
    - Split contexts if components re-render too frequently
    - Consider using `React.memo` for expensive child components
    - Profile with React DevTools to identify performance issues

12. **Migration Path**:
    - If state management becomes complex: Migrate to Zustand (minimal refactor)
    - If need Redux devtools: Migrate to Redux Toolkit (moderate refactor)
    - Current structure makes migration straightforward (actions, reducers already defined)

---

## Summary and Next Steps

This research document provides comprehensive analysis of 8 critical technical decisions for the BoardingPass mobile onboarding app:

1. **SRP-6a**: Use `secure-remote-password` library with expo-crypto
2. **Certificate Pinning**: Hybrid approach with certificate inspection and fingerprint pinning
3. **mDNS Discovery**: react-native-zeroconf with Expo config plugin + fallback IP
4. **QR Scanning**: expo-barcode-scanner with migration path to vision-camera if needed
5. **Secure Storage**: expo-secure-store for tokens and certificate pins
6. **Type Generation**: OpenAPI code generation using openapi-typescript
7. **Expo vs CLI**: Expo prebuild (hybrid) workflow with config plugins
8. **State Management**: React Context + useReducer with structured state slicing

### Key Principles Applied

- **Minimal Dependencies**: Prefer built-in React/Expo APIs where sufficient (Context, expo-crypto)
- **Security First**: Strong security for auth (SRP-6a), certificate validation, secure storage
- **Pragmatic Balance**: Accept some dependencies when security/functionality requires (SRP library, mDNS)
- **Development Velocity**: Maintain Expo benefits while enabling native features via prebuild
- **Type Safety**: Comprehensive TypeScript coverage via OpenAPI code generation
- **YAGNI**: Choose simplest solution that meets requirements (Context vs Redux)

### Recommended Implementation Order

1. **Project Setup**: Initialize Expo project with prebuild configuration
2. **Type Generation**: Set up OpenAPI type generation pipeline
3. **API Client**: Build typed API client with Axios
4. **State Management**: Implement Context + useReducer structure
5. **Secure Storage**: Implement session token and certificate pin storage
6. **mDNS Discovery**: Integrate react-native-zeroconf for device discovery
7. **SRP-6a Auth**: Implement authentication flow with SRP library
8. **Certificate Pinning**: Implement certificate validation and pinning
9. **QR Scanning**: Add QR code scanning for connection codes
10. **UI Components**: Build React Native Paper UI components
11. **Error Handling**: Implement comprehensive error handling and user feedback
12. **Testing**: Add unit, integration, and E2E tests

All decisions align with feature specification requirements and BoardingPass constitution principles while maintaining practical development velocity and security.
