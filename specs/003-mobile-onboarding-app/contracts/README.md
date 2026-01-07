# API Contracts: Mobile Device Onboarding App

**Feature Branch**: `003-mobile-onboarding-app`
**Created**: 2025-12-10
**Status**: Contract Reference Complete

This document describes the API contracts used by the BoardingPass mobile onboarding application. The mobile app implements against the existing BoardingPass RESTful API without requiring knowledge of service implementation details.

---

## Contract Location

The mobile app implements against the **BoardingPass API OpenAPI 3.1 specification**:

**Path**: [`../../001-boardingpass-api/contracts/openapi.yaml`](../../001-boardingpass-api/contracts/openapi.yaml)

**Principle Alignment**: This approach aligns with Constitution Principle V: "Transport Agnostic & Protocol First" - the Protocol definition acts as the single source of truth; neither the device nor the mobile app shall rely on implementation details of the other.

---

## Endpoints Used

The mobile app uses the following BoardingPass API endpoints:

### 1. Authentication Endpoints

#### POST `/auth/srp/init`

**Purpose**: Initialize SRP-6a authentication handshake

**Request**:
```typescript
interface SRPInitRequest {
  username: string;    // Device identifier (e.g., "device")
  A: string;          // Client's ephemeral public key (base64)
}
```

**Response**:
```typescript
interface SRPInitResponse {
  salt: string;       // Password salt (base64)
  B: string;          // Server's ephemeral public key (base64)
}
```

**Errors**:
- `400 Bad Request`: Invalid request format
- `429 Too Many Requests`: Rate limit exceeded (brute force protection)
- `500 Internal Server Error`: Server-side error

---

#### POST `/auth/srp/verify`

**Purpose**: Complete SRP-6a authentication and obtain session token

**Request**:
```typescript
interface SRPVerifyRequest {
  M1: string;         // Client proof (base64)
}
```

**Response**:
```typescript
interface SRPVerifyResponse {
  M2: string;         // Server proof (base64)
  token: string;      // Session token (JWT or opaque)
  expiresAt: string;  // Token expiration time (ISO 8601)
}
```

**Errors**:
- `401 Unauthorized`: Invalid client proof (authentication failed)
- `403 Forbidden`: Account locked (too many failed attempts)
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server-side error

---

### 2. Information Endpoints

These endpoints require authentication via `Authorization: Bearer <token>` header.

#### GET `/info`

**Purpose**: Retrieve device system information

**Request**: None (authenticated GET)

**Response**:
```typescript
interface SystemInfo {
  tpm: {
    present: boolean;
    version?: string;
    manufacturer?: string;
  };
  board: {
    manufacturer?: string;
    productName?: string;
    serialNumber?: string;
    uuid?: string;
  };
  cpu: {
    model?: string;
    architecture?: string;
    cores?: number;
  };
  os: {
    distribution?: string;
    version?: string;
    kernel?: string;
    hostname?: string;
  };
  fips: {
    enabled: boolean;
    validated?: boolean;
  };
}
```

**Errors**:
- `401 Unauthorized`: Invalid or expired session token
- `500 Internal Server Error`: Server-side error

---

#### GET `/network`

**Purpose**: Retrieve device network configuration

**Request**: None (authenticated GET)

**Response**:
```typescript
interface NetworkConfig {
  interfaces: NetworkInterface[];
}

interface NetworkInterface {
  name: string;
  index: number;
  macAddress?: string;
  linkState: 'up' | 'down' | 'unknown';
  operationalState?: string;
  addresses: IPAddress[];
  type?: string;
}

interface IPAddress {
  address: string;
  family: 'ipv4' | 'ipv6';
  prefixLength: number;
  scope?: 'host' | 'link' | 'global' | 'site';
}
```

**Errors**:
- `401 Unauthorized`: Invalid or expired session token
- `500 Internal Server Error`: Server-side error

---

## FIPS Compatibility Requirements

**CRITICAL**: The BoardingPass service operates in FIPS 140-3 mode. The mobile app MUST use FIPS-compatible cryptographic parameters for SRP-6a authentication.

### Required SRP-6a Parameters

The mobile app MUST configure SRP-6a authentication with the following parameters to match the server:

1. **Hash Algorithm**: SHA-256 (FIPS 180-4 approved)
   - Server uses `crypto/sha256` from Go stdlib
   - Client MUST configure SHA-256 in SRP library
   - ⚠️ **DO NOT use SHA-1** - incompatible and insecure

2. **SRP Group**: RFC 5054 2048-bit safe prime (FIPS 186-4 compliant)
   - Server uses RFC 5054 2048-bit group
   - Client MUST use matching 2048-bit group
   - ⚠️ **DO NOT use 1024-bit or 1536-bit groups** - incompatible

3. **Generator**: g = 2
   - Server uses generator g = 2
   - Client MUST use matching generator

### Implementation Reference

```typescript
// Example SRP client configuration (library-specific)
import SRP from 'secure-remote-password/client';

const srpClient = new SRP({
  hash: 'sha256',           // MUST be SHA-256
  group: 'rfc5054-2048',    // MUST be 2048-bit RFC 5054
  // Generator g=2 is standard for RFC 5054 groups
});
```

**See Also**: [`../research.md`](../research.md) Section 1 "FIPS Compatibility Requirements" for detailed implementation guidance.

---

## TypeScript Type Generation

The mobile app uses `openapi-typescript` to generate TypeScript types from the OpenAPI specification, ensuring type safety and automatic sync with API changes.

### Setup

```bash
# Install dependencies
npm install --save-dev openapi-typescript @redocly/openapi-cli

# Generate types
npm run generate:types
```

### Package.json Scripts

```json
{
  "scripts": {
    "generate:types": "openapi-typescript ../../specs/001-boardingpass-api/contracts/openapi.yaml -o ./src/types/api.ts",
    "validate:spec": "openapi lint ../../specs/001-boardingpass-api/contracts/openapi.yaml",
    "prebuild": "npm run validate:spec && npm run generate:types"
  }
}
```

### Usage in Code

```typescript
// Import generated types
import type { paths, components } from '@/types/api';

// Extract specific types
type SRPInitRequest = components['schemas']['SRPInitRequest'];
type SRPInitResponse = components['schemas']['SRPInitResponse'];
type SystemInfo = components['schemas']['SystemInfo'];

// Use with API client
const response = await api.post<SRPInitResponse>('/auth/srp/init', requestData);
```

---

## API Client Implementation

The mobile app implements a typed API client using Axios with certificate pinning support.

### Example Client

```typescript
import axios, { AxiosInstance } from 'axios';
import type { components } from '@/types/api';

class BoardingPassClient {
  private client: AxiosInstance;
  private baseURL: string;

  constructor(deviceHost: string, devicePort: number, sessionToken?: string) {
    this.baseURL = `https://${deviceHost}:${devicePort}`;

    this.client = axios.create({
      baseURL: this.baseURL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
        ...(sessionToken && { 'Authorization': `Bearer ${sessionToken}` }),
      },
    });
  }

  // SRP-6a Authentication
  async srpInit(request: components['schemas']['SRPInitRequest']): Promise<components['schemas']['SRPInitResponse']> {
    const response = await this.client.post('/auth/srp/init', request);
    return response.data;
  }

  async srpVerify(request: components['schemas']['SRPVerifyRequest']): Promise<components['schemas']['SRPVerifyResponse']> {
    const response = await this.client.post('/auth/srp/verify', request);
    return response.data;
  }

  // Information Retrieval (requires authentication)
  async getSystemInfo(): Promise<components['schemas']['SystemInfo']> {
    const response = await this.client.get('/info');
    return response.data;
  }

  async getNetworkConfig(): Promise<components['schemas']['NetworkConfig']> {
    const response = await this.client.get('/network');
    return response.data;
  }
}
```

---

## Error Handling

The mobile app MUST handle standard HTTP error codes as defined in the OpenAPI specification:

### Authentication Errors

- **400 Bad Request**: Malformed request - validate input before submission
- **401 Unauthorized**: Invalid credentials or expired session - prompt re-authentication
- **403 Forbidden**: Account locked - display user-friendly lockout message
- **429 Too Many Requests**: Rate limited - implement exponential backoff

### Server Errors

- **500 Internal Server Error**: Server-side issue - display error with retry option
- **503 Service Unavailable**: Server temporarily unavailable - retry with backoff

### Network Errors

- **ETIMEDOUT**: Request timeout - check network connectivity
- **ECONNREFUSED**: Connection refused - device may be offline
- **ENOTFOUND**: DNS resolution failed - verify device address
- **CERT_INVALID**: Certificate validation failed - check certificate pinning

---

## Contract Testing

The mobile app SHOULD implement contract tests to verify compliance with the OpenAPI specification.

### Example Test

```typescript
import { describe, it, expect } from '@jest/globals';
import Ajv from 'ajv';
import openapi from '../../specs/001-boardingpass-api/contracts/openapi.yaml';

describe('API Contract Compliance', () => {
  const ajv = new Ajv();

  it('should match SRPInitResponse schema', async () => {
    const response = await client.srpInit({ username: 'device', A: 'base64...' });

    const schema = openapi.components.schemas.SRPInitResponse;
    const validate = ajv.compile(schema);

    expect(validate(response)).toBe(true);
  });
});
```

---

## Session Management

### Session Token Lifecycle

1. **Acquisition**: Obtained from `/auth/srp/verify` after successful authentication
2. **Storage**: Stored securely via expo-secure-store (OS Keychain/Keystore)
3. **Usage**: Included in `Authorization: Bearer <token>` header for authenticated requests
4. **Expiration**: 30 minutes from issuance (server-enforced, `expiresAt` field)
5. **Renewal**: No token refresh - user must re-authenticate after expiration
6. **Revocation**: Cleared on logout or authentication error

### Session State Machine

```
[No Session] → [Authenticating] → [Authenticated] → [Expired]
                      ↓                   ↓
                  [Failed]            [Logged Out]
                      ↓                   ↓
                [No Session] ←───────────┘
```

---

## Security Considerations

1. **TLS Certificate Validation**: Implement certificate pinning for self-signed certificates (see `research.md` Section 2)
2. **SRP-6a Parameters**: MUST use FIPS-compatible parameters (SHA-256, 2048-bit group)
3. **Session Token Storage**: Store in secure storage only (expo-secure-store)
4. **Connection Code Handling**: NEVER persist connection codes (in-memory only)
5. **Logging**: NEVER log sensitive data (connection codes, SRP values, session tokens)
6. **Error Messages**: Display user-friendly messages, log technical details separately

---

## Version Compatibility

- **OpenAPI Version**: 3.1
- **BoardingPass API Version**: 1.0.0 (from spec)
- **Mobile App Version**: 1.0.0
- **Breaking Changes**: Any changes to API contract (request/response schemas, endpoint paths) require:
  - OpenAPI spec version bump
  - Mobile app type regeneration
  - Compatibility testing

---

## References

- **OpenAPI Specification**: [`../../001-boardingpass-api/contracts/openapi.yaml`](../../001-boardingpass-api/contracts/openapi.yaml)
- **SRP-6a RFC**: [RFC 5054 - Using SRP for TLS Authentication](https://tools.ietf.org/html/rfc5054)
- **FIPS 180-4**: SHA-256 specification
- **FIPS 186-4**: Digital Signature Standard (SRP group parameters)
- **Research Documentation**: [`../research.md`](../research.md)
- **Data Model**: [`../data-model.md`](../data-model.md)

---

**Version**: 1.0.0
**Last Updated**: 2025-12-10
