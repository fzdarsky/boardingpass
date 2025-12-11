# Data Model: Boarding CLI Tool

**Feature**: 002-boarding-cli
**Date**: 2025-12-10
**Purpose**: Document the key entities and data structures for the Boarding CLI implementation

## Overview

The Boarding CLI is a stateless client that interacts with the BoardingPass service. It manages three categories of data:
1. **Configuration** - User-provided connection and preference settings
2. **Session State** - Ephemeral authentication tokens
3. **TLS Trust** - Persistent certificate fingerprints for TOFU

All data is stored in filesystem files using OS-specific standard directories.

## Entity Definitions

### 1. Configuration (`internal/cli/config/config.go`)

**Purpose**: Stores connection details and CLI preferences, loaded from multiple sources with precedence.

**Fields**:
```go
type Config struct {
    // Connection details
    Host   string  // BoardingPass service hostname or IP
    Port   int     // BoardingPass service port (default: 8443)
    CACert string  // Optional path to custom CA certificate bundle

    // Future: Additional preferences (timeout, retries, etc.)
}
```

**Validation Rules**:
- `Host` is REQUIRED (must be non-empty after loading all sources)
- `Port` must be in range 1-65535 (default: 8443)
- `CACert`, if provided, must be a readable file path

**Storage Locations** (per FR-022):
- Linux: `~/.config/boardingpass/config.yaml`
- macOS: `~/Library/Application Support/boardingpass/config.yaml`
- Windows: `%APPDATA%\boardingpass\config.yaml`

**File Format** (flat YAML per FR-021):
```yaml
host: boardingpass.local
port: 8443
ca_cert: /path/to/ca-bundle.pem
```

**Loading Precedence** (FR-023):
1. Config file (lowest priority)
2. Environment variables (`BOARDING_HOST`, `BOARDING_PORT`, `BOARDING_CA_CERT`)
3. Command-line flags (highest priority)

**Lifecycle**:
- Created/updated manually by user or installation scripts
- Read at CLI startup for every command
- Never automatically modified by CLI

---

### 2. Session Token (`internal/cli/session/store.go`)

**Purpose**: Stores authentication tokens obtained via SRP-6a to avoid re-authentication for subsequent commands within the 30-minute TTL.

**Fields**:
```go
// Not a struct - stored as raw string in file
type SessionToken string  // JWT or opaque token from BoardingPass service
```

**Metadata** (implicit from storage):
- **Host**: Derived from filename hash
- **Port**: Derived from filename hash
- **Created**: File mtime
- **Expires**: Not stored (server-side 30-min TTL)

**Validation Rules**:
- Token is opaque to client (no validation of format/content)
- Server validates token authenticity and expiration
- Client deletes token on 401 Unauthorized response

**Storage Locations** (per FR-004):
- Linux: `~/.cache/boardingpass/session-<hash>.token`
- macOS: `~/Library/Caches/boardingpass/session-<hash>.token`
- Windows: `%LocalAppData%\boardingpass\cache\session-<hash>.token`

**Filename Format**:
```
session-<sha256-first-8-bytes-of-host:port>.token
```
Example: `session-a1b2c3d4e5f6g7h8.token`

**File Format**: Single-line plaintext
```
<session-token-string>
```

**Security** (FR-005):
- File permissions: `0600` (owner read/write only)
- Never logged or displayed to user
- Cleared from memory after use
- Deleted on logout (`complete` command)

**Lifecycle**:
1. Created by `pass` command after successful SRP authentication
2. Read by all authenticated commands (info, connections, load, command, complete)
3. Deleted on `complete` command or on 401 error (expired/invalid)
4. Auto-deleted on system reboot (cache directory cleared)

---

### 3. Certificate Fingerprint (`internal/cli/tls/store.go`)

**Purpose**: Stores SHA-256 fingerprints of accepted TLS certificates for Trust-on-First-Use (TOFU) pattern, preventing MITM attacks on subsequent connections.

**Fields**:
```go
type CertificateFingerprint struct {
    Host        string    // "hostname:port" (e.g., "192.168.1.100:8443")
    Fingerprint string    // "SHA256:base64-encoded-hash"
    AcceptedAt  time.Time // When user accepted this certificate
}
```

**Validation Rules**:
- `Host` must match `hostname:port` from connection
- `Fingerprint` must be valid SHA-256 hash (64 hex chars or base64)
- `AcceptedAt` is informational only (no expiration enforced)

**Storage Location**:
- Linux: `~/.config/boardingpass/known_certs.yaml`
- macOS: `~/Library/Application Support/boardingpass/known_certs.yaml`
- Windows: `%APPDATA%\boardingpass\known_certs.yaml`

**File Format** (YAML array per FR-021 flat structure):
```yaml
certificates:
  - host: "192.168.1.100:8443"
    fingerprint: "SHA256:a1b2c3d4e5f6..."
    accepted_at: "2025-12-10T12:00:00Z"
  - host: "boardingpass.local:8443"
    fingerprint: "SHA256:x9y8z7w6v5u4..."
    accepted_at: "2025-12-10T13:30:00Z"
```

**Security**:
- File permissions: `0644` (world-readable, not secret)
- Fingerprints are public information
- Protects against MITM by detecting certificate changes

**Lifecycle**:
1. Created when user accepts first unknown certificate (FR-026)
2. Read on every TLS connection to BoardingPass service
3. Updated when user accepts new certificates
4. User can manually edit/delete entries (e.g., after cert rotation)

**Fingerprint Computation**:
```go
func ComputeFingerprint(cert *x509.Certificate) string {
    hash := sha256.Sum256(cert.Raw)
    return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
}
```

---

## Data Flow Diagrams

### Authentication Flow (Session Token Creation)

```
User: boarding pass --username admin
  ↓
1. Load Config (precedence: flags > env > file)
  ↓
2. Check for existing session token
   - If valid token exists → reuse
   - If no token or 401 error → proceed
  ↓
3. Prompt for password (if not provided via flag)
  ↓
4. SRP Phase 1: POST /auth/srp/init
   Client → {username, A} → Server
   Client ← {salt, B} ← Server
  ↓
5. SRP Phase 2: Compute M1, POST /auth/srp/verify
   Client → {M1} → Server
   Client ← {M2, session_token} ← Server
  ↓
6. Verify M2
  ↓
7. Save session token to file (0600 permissions)
  ↓
Success: "Authentication successful"
```

### Authenticated Request Flow (Using Session Token)

```
User: boarding info
  ↓
1. Load Config
  ↓
2. Load session token from file
   - If not found → error "Run 'boarding pass' first"
   - If found → proceed
  ↓
3. GET /info with Authorization: Bearer <token>
  ↓
4. If 401 Unauthorized:
   - Delete stale token
   - Error "Session expired. Run 'boarding pass' to re-authenticate"
  ↓
5. If 200 OK:
   - Format response (YAML or JSON)
   - Display to user
```

### TLS Certificate Verification Flow (TOFU)

```
Client connects to https://192.168.1.100:8443
  ↓
1. TLS handshake begins
  ↓
2. Server presents certificate
  ↓
3. Compute SHA-256 fingerprint
  ↓
4. Check known_certs.yaml
   - If host not found → goto 5
   - If fingerprint matches → accept connection
   - If fingerprint differs → ERROR "Certificate changed!"
  ↓
5. Prompt user:
   "Unknown certificate for 192.168.1.100:8443
    Fingerprint: SHA256:abc123...
    Accept? (yes/no)"
  ↓
6. If user says "yes":
   - Add to known_certs.yaml
   - Accept connection
  ↓
7. If user says "no":
   - Abort connection
   - Exit with error
```

## State Transitions

### Session Token States

```
┌─────────┐
│  None   │ Initial state (no token file exists)
└────┬────┘
     │ boarding pass (successful auth)
     ↓
┌─────────┐
│  Valid  │ Token exists and accepted by server
└────┬────┘
     │
     ├─→ (30 min TTL expires) ──→ Invalid/Deleted
     ├─→ (boarding complete) ──→ Deleted
     ├─→ (system reboot) ──────→ Deleted (cache dir cleared)
     └─→ (401 error) ──────────→ Deleted
```

### Certificate Fingerprint States

```
┌──────────┐
│ Unknown  │ Host not in known_certs.yaml
└─────┬────┘
      │ user accepts
      ↓
┌──────────┐
│  Known   │ Fingerprint stored in known_certs.yaml
└─────┬────┘
      │
      ├─→ (fingerprint matches) ──→ Connection accepted
      ├─→ (fingerprint differs) ──→ ERROR (possible MITM)
      └─→ (user manual edit) ────→ Updated/Deleted
```

## File Permissions Summary

| File | Permissions | Rationale |
|------|-------------|-----------|
| `config.yaml` | `0644` (rw-r--r--) | Public config, no secrets |
| `session-*.token` | `0600` (rw-------) | Sensitive auth token, owner-only |
| `known_certs.yaml` | `0644` (rw-r--r--) | Public fingerprints, not secret |

## Relationships

```
User
 │
 ├─ creates/edits ──→ Config (config.yaml)
 │                     │
 │                     └─ used by all commands
 │
 ├─ authenticates ──→ Session Token (session-*.token)
 │                     │
 │                     ├─ created by: pass command
 │                     └─ used by: info, connections, load, command, complete
 │
 └─ accepts cert ──→ Certificate Fingerprint (known_certs.yaml)
                       │
                       └─ used by: all HTTPS connections
```

## Validation & Error Handling

### Configuration Validation

| Condition | Error Message |
|-----------|---------------|
| No host specified | "Error: BoardingPass service host not specified. Use --host flag, BOARDING_HOST env var, or add 'host:' to config.yaml" |
| Invalid port | "Error: Invalid port {port}. Must be 1-65535" |
| CA cert file not found | "Error: CA certificate file not found: {path}" |

### Session Token Validation

| Condition | Error Message |
|-----------|---------------|
| No token file exists | "Error: Not authenticated. Run 'boarding pass' to authenticate." |
| Token expired (401) | "Error: Session expired. Run 'boarding pass' to re-authenticate." |
| Token invalid (401) | "Error: Invalid session. Run 'boarding pass' to authenticate." |

### Certificate Fingerprint Validation

| Condition | Error Message |
|-----------|---------------|
| Unknown cert, user rejects | "Error: Certificate rejected by user. Connection aborted." |
| Known cert, fingerprint mismatch | "Error: Certificate fingerprint mismatch for {host}! Possible MITM attack. Expected: {expected}, Got: {actual}" |

## Migration & Compatibility

**Version 1.0 (Initial Release)**:
- No migration needed (new feature)
- Config file format is stable (flat YAML)
- Session token format is opaque (server-controlled)
- Certificate fingerprint format follows SSH known_hosts pattern (industry standard)

**Future Compatibility**:
- Config file changes are BREAKING (per constitution: config files are APIs)
- Adding new config fields is OK (backward compatible)
- Session token format controlled by server (CLI is agnostic)
- Certificate fingerprint format should remain stable (SHA-256 is standard)
