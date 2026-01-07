# Data Model: Mobile Device Onboarding App

**Feature Branch**: `003-mobile-onboarding-app`
**Created**: 2025-12-10
**Status**: Design Complete

This document defines the data model for the BoardingPass mobile onboarding application. It specifies TypeScript interfaces for all entities, their relationships, validation rules, state transitions, and storage strategies.

---

## Entity Overview

The mobile app manages six primary entities:

1. **Device** - Represents a discovered headless Linux device
2. **AuthenticationSession** - Represents a secure connection to a device
3. **ConnectionCode** - Represents authentication credentials for a device
4. **DeviceInformation** - System details retrieved from authenticated device
5. **NetworkConfiguration** - Network details for authenticated device
6. **CertificateInfo** - TLS certificate information and pinning data

---

## 1. Device

Represents a headless Linux device running BoardingPass service, either discovered via mDNS or accessed via well-known IP address.

### TypeScript Interface

```typescript
interface Device {
  // Identity
  id: string;                    // Unique identifier (derived from name + host)
  name: string;                  // Device name (from mDNS or user-provided)

  // Network
  host: string;                  // IP address or hostname
  port: number;                  // HTTPS port (default 9443)
  addresses: string[];           // All IP addresses (for multi-homed devices)

  // Discovery metadata
  discoveryMethod: 'mdns' | 'fallback' | 'manual';
  txt?: Record<string, string>;  // mDNS TXT records (optional metadata)

  // Status
  status: 'online' | 'offline' | 'authenticating' | 'authenticated' | 'error';
  lastSeen: Date;                // Last time device was detected/contacted

  // Relationships (loaded separately)
  certificateInfo?: CertificateInfo;
  systemInfo?: DeviceInformation;
  networkConfig?: NetworkConfiguration;
}
```

### Validation Rules

- `id`: Non-empty string, unique within device list
- `name`: Non-empty string (1-255 characters)
- `host`: Valid IP address (IPv4 or IPv6) or hostname
- `port`: Integer in range 1-65535
- `addresses`: Array of valid IP addresses
- `discoveryMethod`: One of the allowed enum values
- `status`: One of the allowed enum values
- `lastSeen`: Valid Date object

### State Transitions

```
online → authenticating    (User initiates authentication)
authenticating → authenticated (Authentication succeeds)
authenticating → error     (Authentication fails)
authenticating → online    (User cancels authentication)
authenticated → online     (Session expires or user logs out)
online → offline           (Device stops broadcasting/responding)
offline → online           (Device resumes broadcasting)
```

### Storage Strategy

- **In-Memory**: Active device list during app session
- **Not Persisted**: Device discovery state is ephemeral
- **Exception**: Certificate pins are persisted (see CertificateInfo)

### Relationships

- **Has one** CertificateInfo (after first connection)
- **Has one** DeviceInformation (when authenticated)
- **Has one** NetworkConfiguration (when authenticated)
- **Has one** AuthenticationSession (when authenticated)

---

## 2. AuthenticationSession

Represents a secure SRP-6a authentication session with a device. Sessions are ephemeral and expire after 30 minutes (server-enforced).

### TypeScript Interface

```typescript
interface AuthenticationSession {
  // Session identity
  deviceId: string;              // Foreign key to Device.id
  sessionToken: string;          // JWT or opaque token from server

  // Timing
  createdAt: Date;               // Session creation time
  expiresAt: Date;               // Session expiration time (createdAt + 30min)

  // SRP-6a protocol state (ephemeral, never persisted)
  srp?: {
    ephemeralPublic: string;     // Client's ephemeral public key (A)
    ephemeralPrivate: string;    // Client's ephemeral private key (a)
    sessionKey?: string;         // Derived session key (K)
  };

  // Authentication metadata
  authenticatedAt: Date;         // When authentication completed
  connectionCode?: string;       // Connection code used (cleared after auth)
}
```

### FIPS Compatibility Requirements

**CRITICAL**: SRP-6a authentication MUST use FIPS 140-3 compliant parameters to match BoardingPass service:

- **Hash Algorithm**: SHA-256 (FIPS 180-4 approved)
- **SRP Group**: RFC 5054 2048-bit safe prime (FIPS 186-4 compliant)
- **Generator**: g = 2

See `research.md` Section 1 "FIPS Compatibility Requirements" for detailed implementation requirements.

### Validation Rules

- `deviceId`: Non-empty string, must reference existing Device
- `sessionToken`: Non-empty string, JWT or opaque token format
- `createdAt`: Valid Date object, not in future
- `expiresAt`: Valid Date object, after `createdAt`, typically `createdAt + 30min`
- `authenticatedAt`: Valid Date object, between `createdAt` and `expiresAt`
- `srp.ephemeralPublic`: Base64-encoded string (ephemeral key A)
- `srp.ephemeralPrivate`: Base64-encoded string (ephemeral key a) - **NEVER LOGGED**
- `connectionCode`: **NEVER PERSISTED, NEVER LOGGED** - cleared immediately after use

### State Transitions

```
[No session] → [SRP Init]     (User provides connection code)
[SRP Init] → [SRP Verify]     (Server responds with salt + B)
[SRP Verify] → [Authenticated] (Server accepts proof, returns token)
[SRP Verify] → [Failed]       (Server rejects proof or timeout)
[Authenticated] → [Expired]   (expiresAt exceeded)
[Authenticated] → [Cleared]   (User logs out)
[Failed] → [No session]       (Error handled, retry allowed)
```

### Storage Strategy

- **Secure Storage**: `sessionToken`, `expiresAt` only (via expo-secure-store)
- **In-Memory Only**: `srp` state (ephemeral keys, session key)
- **NEVER Stored**: `connectionCode` (cleared from memory after authentication)
- **Encryption**: Session token encrypted by OS (Keychain/Keystore)

### Security Considerations

- Connection code MUST be cleared from memory immediately after authentication
- SRP ephemeral keys MUST be cleared when session ends
- Session token MUST be cleared on logout or authentication failure
- All SRP-related values MUST be excluded from logs (FR-029)

---

## 3. ConnectionCode

Represents authentication credentials for a device. Connection codes are ephemeral, never persisted, and only exist in memory during the authentication flow.

### TypeScript Interface

```typescript
interface ConnectionCode {
  // Code value
  value: string;                 // Connection code (NEVER PERSISTED)

  // Source
  source: 'manual' | 'qr' | 'barcode';

  // Validation
  validationState: 'pending' | 'valid' | 'invalid' | 'used';
  validatedAt?: Date;            // When validation completed

  // Associated device
  deviceId: string;              // Device this code is for
}
```

### Validation Rules

- `value`: Non-empty string, format depends on BoardingPass implementation
  - Likely base64-encoded (pattern: `^[A-Za-z0-9+/=]{32,}$`)
  - Exact format TBD during implementation
- `source`: One of 'manual', 'qr', 'barcode'
- `validationState`: One of allowed enum values
- `deviceId`: Non-empty string, must reference existing Device

### State Transitions

```
[Created] → pending        (Code entered/scanned)
pending → valid            (Format validation passes)
pending → invalid          (Format validation fails)
valid → used               (Authentication initiated)
used → [Cleared]           (After authentication attempt)
invalid → [Cleared]        (User retries with new code)
```

### Storage Strategy

- **In-Memory Only**: Connection codes exist only during authentication flow
- **NEVER Persisted**: No storage to disk, secure or otherwise (FR-036)
- **Cleared Immediately**: After authentication attempt (success or failure)

### Security Considerations

- Connection codes are sensitive secrets and MUST NOT be logged (FR-029)
- Codes MUST be cleared from memory as soon as authentication completes
- Failed authentication MUST clear the code (prevent reuse)
- Clipboard access (paste) is acceptable, but clear clipboard after paste (optional enhancement)

---

## 4. DeviceInformation

System details retrieved from authenticated device via `/info` endpoint. Matches `SystemInfo` schema from BoardingPass API.

### TypeScript Interface

```typescript
// Generated from OpenAPI spec: specs/001-boardingpass-api/contracts/openapi.yaml
interface DeviceInformation {
  // TPM information
  tpm: {
    present: boolean;            // TPM chip available
    version?: string;            // TPM version (e.g., "2.0")
    manufacturer?: string;       // TPM manufacturer
  };

  // Board/hardware information
  board: {
    manufacturer?: string;       // Board manufacturer (from DMI)
    productName?: string;        // Product name
    serialNumber?: string;       // Board serial number
    uuid?: string;               // System UUID
  };

  // CPU information
  cpu: {
    model?: string;              // CPU model name
    architecture?: string;       // Architecture (e.g., "x86_64", "aarch64")
    cores?: number;              // Number of CPU cores
  };

  // Operating system information
  os: {
    distribution?: string;       // Linux distribution (e.g., "RHEL")
    version?: string;            // Distribution version
    kernel?: string;             // Kernel version
    hostname?: string;           // System hostname
  };

  // FIPS mode status
  fips: {
    enabled: boolean;            // FIPS mode enabled
    validated?: boolean;         // FIPS validation status
  };

  // Metadata
  retrievedAt: Date;             // When info was fetched (client-side)
}
```

### Validation Rules

- `tpm.present`: Boolean (required)
- `tpm.version`: String (optional), format "X.X" if present
- `board.uuid`: String (optional), valid UUID format if present
- `cpu.cores`: Integer > 0 (optional)
- `os.distribution`: String (optional), non-empty if present
- `fips.enabled`: Boolean (required)
- `retrievedAt`: Valid Date object, not in future

### Storage Strategy

- **In-Memory Only**: Loaded when viewing device details
- **Not Persisted**: Cleared when user navigates away or logs out
- **Cache**: Could cache for session duration (optional enhancement)

### Relationships

- **Belongs to** one Device (via deviceId)

---

## 5. NetworkConfiguration

Network details for authenticated device via `/network` endpoint. Includes all network interfaces and their configurations.

### TypeScript Interface

```typescript
// Generated from OpenAPI spec: specs/001-boardingpass-api/contracts/openapi.yaml
interface NetworkConfiguration {
  interfaces: NetworkInterface[];
  retrievedAt: Date;             // When config was fetched (client-side)
}

interface NetworkInterface {
  // Interface identity
  name: string;                  // Interface name (e.g., "eth0", "wlan0")
  index: number;                 // Interface index

  // Hardware
  macAddress?: string;           // MAC address (if applicable)

  // Status
  linkState: 'up' | 'down' | 'unknown';
  operationalState?: string;     // Detailed state (e.g., "routable", "degraded")

  // IP addresses
  addresses: IPAddress[];

  // Interface type metadata
  type?: string;                 // Interface type (e.g., "ether", "loopback")
}

interface IPAddress {
  address: string;               // IP address (IPv4 or IPv6)
  family: 'ipv4' | 'ipv6';       // Address family
  prefixLength: number;          // CIDR prefix length (e.g., 24 for /24)
  scope?: 'host' | 'link' | 'global' | 'site';
}
```

### Validation Rules

- `interfaces`: Non-empty array (device has at least one interface)
- `name`: Non-empty string, valid interface name pattern
- `index`: Integer >= 0
- `macAddress`: Valid MAC address format (XX:XX:XX:XX:XX:XX) if present
- `linkState`: One of 'up', 'down', 'unknown'
- `address`: Valid IP address (IPv4: dotted quad, IPv6: colon-hex)
- `family`: One of 'ipv4', 'ipv6'
- `prefixLength`: Integer 0-32 (IPv4) or 0-128 (IPv6)
- `retrievedAt`: Valid Date object, not in future

### Storage Strategy

- **In-Memory Only**: Loaded when viewing device details
- **Not Persisted**: Cleared when user navigates away or logs out
- **Cache**: Could cache for session duration (optional enhancement)

### Relationships

- **Belongs to** one Device (via deviceId)

---

## 6. CertificateInfo

TLS certificate information and pinning data for device connections. Implements Trust-On-First-Use (TOFU) with certificate pinning.

### TypeScript Interface

```typescript
interface CertificateInfo {
  // Device association
  deviceId: string;              // Foreign key to Device.id

  // Certificate identity
  fingerprint: string;           // SHA-256 fingerprint of certificate (hex format)

  // Certificate details
  subject: string;               // Certificate subject (e.g., "CN=device.local")
  issuer: string;                // Certificate issuer
  validFrom: Date;               // Certificate validity start
  validTo: Date;                 // Certificate validity end

  // Certificate type
  isSelfSigned: boolean;         // Whether cert is self-signed
  issuedByTrustedCA: boolean;    // Whether issuer is in OS trust store

  // Trust status
  trustStatus: 'trusted_ca' | 'self_signed_trusted' | 'self_signed_new' | 'changed';

  // Pinning metadata
  pinnedAt: Date;                // When certificate was first trusted
  lastVerified: Date;            // Last successful verification

  // User decision
  userConfirmedAt?: Date;        // When user explicitly trusted (for self-signed)
}
```

### Validation Rules

- `deviceId`: Non-empty string, must reference existing Device
- `fingerprint`: 64-character hex string (SHA-256 hash)
- `subject`: Non-empty string, valid X.509 DN format
- `issuer`: Non-empty string, valid X.509 DN format
- `validFrom`: Valid Date object
- `validTo`: Valid Date object, after `validFrom`
- `isSelfSigned`: Boolean (required)
- `issuedByTrustedCA`: Boolean (required)
- `trustStatus`: One of the allowed enum values
- `pinnedAt`: Valid Date object, not in future
- `lastVerified`: Valid Date object, >= `pinnedAt`

### Trust Status Values

- `trusted_ca`: Certificate issued by CA in OS trust store (green checkmark)
- `self_signed_trusted`: Self-signed certificate, previously trusted by user (yellow shield)
- `self_signed_new`: Self-signed certificate, not yet trusted (orange warning)
- `changed`: Certificate changed from previously pinned version (red alert)

### State Transitions

```
[No certificate] → self_signed_new    (First connection, self-signed cert)
[No certificate] → trusted_ca         (First connection, CA-signed cert)
self_signed_new → self_signed_trusted (User confirms trust)
self_signed_trusted → changed         (Certificate fingerprint changes)
trusted_ca → changed                  (Certificate fingerprint changes)
changed → self_signed_trusted         (User accepts new certificate)
changed → trusted_ca                  (User accepts new CA-signed cert)
```

### Storage Strategy

- **Secure Storage**: All fields persisted via expo-secure-store (OS Keychain/Keystore)
- **Encryption**: Encrypted by OS, protected by device passcode
- **Persistence**: Survives app restarts, cleared on app uninstall
- **Key Format**: `cert_pin_${deviceId}`

### Security Considerations

- First connection is vulnerable to MITM (TOFU limitation) - acceptable for local network use
- Fingerprint MUST be computed using SHA-256 (FIPS-compatible)
- Certificate change detection prevents ongoing MITM attacks
- User MUST be alerted when `trustStatus === 'changed'`
- Consider certificate expiration checking (warn user of expired certs)

### Relationships

- **Belongs to** one Device (via deviceId)

---

## Entity Relationships

```
Device (1) ──── (0..1) CertificateInfo
  │
  ├──── (0..1) AuthenticationSession
  │
  ├──── (0..1) DeviceInformation
  │
  └──── (0..1) NetworkConfiguration

AuthenticationSession (1) ──── (1) ConnectionCode (ephemeral)
```

### Relationship Descriptions

1. **Device → CertificateInfo**: A device may have pinned certificate information after first connection
2. **Device → AuthenticationSession**: A device may have an active authentication session
3. **Device → DeviceInformation**: A device may have cached system information (when authenticated)
4. **Device → NetworkConfiguration**: A device may have cached network configuration (when authenticated)
5. **AuthenticationSession → ConnectionCode**: An authentication session is created using a connection code (ephemeral, cleared after use)

---

## Data Flow

### 1. Device Discovery Flow

```
[mDNS/Fallback] → Device (in-memory)
                    ↓
                [Display in UI]
```

### 2. Authentication Flow

```
[User Input] → ConnectionCode (in-memory, ephemeral)
                 ↓
           [SRP-6a Protocol]
                 ↓
         AuthenticationSession
                 ↓
        sessionToken → SecureStore
```

### 3. Certificate Pinning Flow

```
[First Connection] → Fetch Certificate
                         ↓
                    CertificateInfo
                         ↓
             [User Confirms Trust]
                         ↓
                    SecureStore
```

### 4. Device Information Flow

```
[Authenticated] → Fetch /info + /network
                       ↓
    DeviceInformation + NetworkConfiguration (in-memory)
                       ↓
               [Display in UI]
```

---

## Storage Summary

| Entity | Storage Type | Persistence | Encryption |
|--------|-------------|-------------|------------|
| Device | In-Memory | Session only | N/A |
| AuthenticationSession (token) | SecureStore | Until expiration/logout | OS Keychain/Keystore |
| AuthenticationSession (SRP state) | In-Memory | Cleared after auth | N/A |
| ConnectionCode | In-Memory | Cleared after auth | N/A |
| DeviceInformation | In-Memory | Session only | N/A |
| NetworkConfiguration | In-Memory | Session only | N/A |
| CertificateInfo | SecureStore | Permanent (until app uninstall) | OS Keychain/Keystore |

---

## Type Generation Strategy

All API-related types (DeviceInformation, NetworkConfiguration, SRP request/response types) are generated from OpenAPI specification using `openapi-typescript`:

```bash
npm run generate:types
# Generates: src/types/api.ts from specs/001-boardingpass-api/contracts/openapi.yaml
```

App-specific types (Device, CertificateInfo, etc.) are manually defined in `src/types/` directory.

---

## Validation Libraries

- **Runtime Validation**: Consider using Zod schemas generated from OpenAPI for API response validation
- **Format Validation**: Use standard libraries for IP addresses, MAC addresses, UUIDs, dates
- **Custom Validators**: Implement for connection codes, certificate fingerprints, device names

---

## Version

**Version**: 1.0.0
**Created**: 2025-12-10
**Last Updated**: 2025-12-10
