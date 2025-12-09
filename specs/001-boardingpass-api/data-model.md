# Data Model: BoardingPass API

**Feature**: BoardingPass API
**Branch**: `001-boardingpass-api`
**Version**: 0.1.0
**Date**: 2025-12-06
**References**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md)

## Overview

This document defines the core entities, their relationships, and JSON schemas for the BoardingPass API. All schemas are designed to be serializable to JSON for REST API transport and align with OpenAPI 3.1 specification standards.

**Version Note**: Starting at 0.1.0 to allow breaking changes before first stable 1.0.0 release.

---

## Entity Definitions

### 1. SRP Verifier Configuration

**Description**: Stored configuration for SRP-6a authentication. Contains username, salt, and path to password generator script. The verifier value (`v = g^x % N`) is computed dynamically at runtime by executing the password generator script.

**Storage Location**: `/etc/boardingpass/verifier` (read-only, embedded in bootc image)

**Lifecycle**: Read once at service startup, password generated per authentication attempt, verifier computed on-demand

**JSON Schema**:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "SRPVerifierConfig",
  "type": "object",
  "required": ["username", "salt", "password_generator"],
  "properties": {
    "username": {
      "type": "string",
      "description": "SRP username (typically 'boardingpass')",
      "minLength": 1,
      "maxLength": 64,
      "example": "boardingpass"
    },
    "salt": {
      "type": "string",
      "description": "Base64-encoded salt (unique per bootc image build)",
      "pattern": "^[A-Za-z0-9+/]+=*$",
      "minLength": 16,
      "example": "c29tZXJhbmRvbXNhbHR2YWx1ZQ=="
    },
    "password_generator": {
      "type": "string",
      "description": "Absolute path to executable script that outputs device-unique password",
      "pattern": "^/.*",
      "example": "/usr/lib/boardingpass/password-generator"
    }
  }
}
```

**Example**:

```json
{
  "username": "boardingpass",
  "salt": "c29tZXJhbmRvbXNhbHR2YWx1ZQ==",
  "password_generator": "/usr/lib/boardingpass/password-generator"
}
```

**Notes**:
- Password generator script must be executable (`0500`) and output device-unique value to stdout
- Script examples: TPM endorsement key, board serial number, MAC address, or combination
- Verifier is never stored; computed dynamically: `v = g^x % N` where `x = H(salt | H(username | ":" | password))`

---

### 2. Session Token

**Description**: Short-lived bearer token issued after successful SRP-6a authentication. Used to authenticate all subsequent API requests.

**Storage Location**: In-memory map (server-side), no persistence

**Lifecycle**: Created after SRP verification success, expires after TTL (default: 30 minutes), cleaned up by background goroutine

**Structure**: `<token_id>.<signature>` where:
- `token_id`: Base64-encoded 32-byte random value
- `signature`: HMAC-SHA256(token_id + username, secret_key)

**JSON Schema**:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "SessionToken",
  "type": "string",
  "description": "HMAC-signed session token (format: token_id.signature)",
  "pattern": "^[A-Za-z0-9_-]+\\.[A-Za-z0-9_-]+$",
  "example": "dGhpc2lzYXRva2VuaWQ.c2lnbmF0dXJlaGVyZQ"
}
```

**Internal Representation** (not exposed via API):

```json
{
  "token": "dGhpc2lzYXRva2VuaWQ.c2lnbmF0dXJlaGVyZQ",
  "username": "boardingpass",
  "created_at": "2025-12-06T10:00:00Z",
  "expires_at": "2025-12-06T10:30:00Z"
}
```

**Notes**:
- Maximum 10 concurrent sessions (defense against exhaustion)
- Tokens are invalidated on service restart (in-memory only)
- HMAC secret is generated at service startup using `crypto/rand`

---

### 3. System Information

**Description**: Hardware and software characteristics of the device. Immutable or rarely-changing attributes derived from system inspection.

**Storage Location**: Not stored; queried on-demand from `/sys`, `/proc`, DMI tables, and TPM

**Lifecycle**: Read-only, generated per API request to `/info`

**JSON Schema**:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "SystemInfo",
  "type": "object",
  "required": ["tpm", "board", "cpu", "os", "fips_mode"],
  "properties": {
    "tpm": {
      "type": "object",
      "description": "TPM (Trusted Platform Module) information",
      "required": ["present"],
      "properties": {
        "present": {
          "type": "boolean",
          "description": "Whether TPM is present on device"
        },
        "manufacturer": {
          "type": "string",
          "description": "TPM manufacturer (null if not present)",
          "example": "STMicroelectronics"
        },
        "model": {
          "type": "string",
          "description": "TPM model/version (null if not present)",
          "example": "ST33HTPH2E32"
        },
        "version": {
          "type": "string",
          "description": "TPM specification version (null if not present)",
          "example": "2.0"
        }
      }
    },
    "board": {
      "type": "object",
      "description": "Motherboard/baseboard information from DMI",
      "required": ["manufacturer", "model", "serial"],
      "properties": {
        "manufacturer": {
          "type": "string",
          "description": "Board manufacturer",
          "example": "Raspberry Pi Foundation"
        },
        "model": {
          "type": "string",
          "description": "Board model",
          "example": "Raspberry Pi 4 Model B"
        },
        "serial": {
          "type": "string",
          "description": "Board serial number",
          "example": "10000000abcdef01"
        }
      }
    },
    "cpu": {
      "type": "object",
      "description": "CPU architecture information",
      "required": ["architecture"],
      "properties": {
        "architecture": {
          "type": "string",
          "description": "CPU architecture",
          "enum": ["x86_64", "aarch64", "armv7l"],
          "example": "aarch64"
        }
      }
    },
    "os": {
      "type": "object",
      "description": "Operating system information",
      "required": ["distribution", "version"],
      "properties": {
        "distribution": {
          "type": "string",
          "description": "Linux distribution name",
          "example": "Red Hat Enterprise Linux"
        },
        "version": {
          "type": "string",
          "description": "Distribution version",
          "example": "9.3"
        }
      }
    },
    "fips_mode": {
      "type": "boolean",
      "description": "Whether FIPS 140-3 mode is enabled",
      "example": true
    }
  }
}
```

**Example**:

```json
{
  "tpm": {
    "present": true,
    "manufacturer": "STMicroelectronics",
    "model": "ST33HTPH2E32",
    "version": "2.0"
  },
  "board": {
    "manufacturer": "Raspberry Pi Foundation",
    "model": "Raspberry Pi 4 Model B",
    "serial": "10000000abcdef01"
  },
  "cpu": {
    "architecture": "aarch64"
  },
  "os": {
    "distribution": "Red Hat Enterprise Linux",
    "version": "9.3"
  },
  "fips_mode": true
}
```

**Example (No TPM)**:

```json
{
  "tpm": {
    "present": false,
    "manufacturer": null,
    "model": null,
    "version": null
  },
  "board": {
    "manufacturer": "Dell Inc.",
    "model": "OptiPlex 7090",
    "serial": "ABC123DEF456"
  },
  "cpu": {
    "architecture": "x86_64"
  },
  "os": {
    "distribution": "Ubuntu",
    "version": "22.04"
  },
  "fips_mode": false
}
```

---

### 4. Network Configuration

**Description**: Current network interface state including link status and IP addresses. Real-time snapshot queried from NetworkManager via D-Bus.

**Data Source**: NetworkManager D-Bus API (`org.freedesktop.NetworkManager`)

**Storage Location**: Not stored; queried on-demand via D-Bus

**Lifecycle**: Read-only, generated per API request to `/network`

**NetworkManager D-Bus Mapping**:
- `name`: `org.freedesktop.NetworkManager.Device.Interface`
- `mac`: `org.freedesktop.NetworkManager.Device.HwAddress`
- `link_state`: Derived from `org.freedesktop.NetworkManager.Device.Carrier` and `State`
- `addresses`: Extracted from `Ip4Config` and `Ip6Config` object paths

**Potential Dependency**: `github.com/Wifx/gonetworkmanager` or `github.com/godbus/dbus/v5` (minimal D-Bus library)

**JSON Schema**:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "NetworkConfig",
  "type": "object",
  "required": ["interfaces"],
  "properties": {
    "interfaces": {
      "type": "array",
      "description": "List of network interfaces",
      "maxItems": 32,
      "items": {
        "type": "object",
        "required": ["name", "mac", "link_state", "addresses"],
        "properties": {
          "name": {
            "type": "string",
            "description": "Interface name",
            "pattern": "^[a-zA-Z0-9]+$",
            "example": "eth0"
          },
          "mac_address": {
            "type": "string",
            "description": "MAC address (colon-separated hex)",
            "pattern": "^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$",
            "example": "dc:a6:32:12:34:56"
          },
          "link_state": {
            "type": "string",
            "description": "Link state",
            "enum": ["up", "down"],
            "example": "up"
          },
          "ip_addresses": {
            "type": "array",
            "description": "Assigned IP addresses",
            "items": {
              "type": "object",
              "required": ["ip", "prefix", "family"],
              "properties": {
                "ip": {
                  "type": "string",
                  "description": "IP address",
                  "oneOf": [
                    { "format": "ipv4" },
                    { "format": "ipv6" }
                  ],
                  "example": "192.168.1.100"
                },
                "prefix": {
                  "type": "integer",
                  "description": "Prefix length (CIDR notation)",
                  "minimum": 0,
                  "maximum": 128,
                  "example": 24
                },
                "family": {
                  "type": "string",
                  "description": "Address family",
                  "enum": ["ipv4", "ipv6"],
                  "example": "ipv4"
                }
              }
            }
          }
        }
      }
    }
  }
}
```

**Example**:

```json
{
  "interfaces": [
    {
      "name": "eth0",
      "mac": "dc:a6:32:12:34:56",
      "link_state": "up",
      "addresses": [
        {
          "ip": "192.168.1.100",
          "prefix": 24,
          "family": "ipv4"
        },
        {
          "ip": "fe80::dea6:32ff:fe12:3456",
          "prefix": 64,
          "family": "ipv6"
        }
      ]
    },
    {
      "name": "wlan0",
      "mac": "b8:27:eb:98:76:54",
      "link_state": "down",
      "addresses": []
    }
  ]
}
```

---

### 5. Configuration Bundle

**Description**: A collection of files to be atomically written to `/etc`. Each file includes path, content (Base64-encoded), and Unix permissions.

**Storage Location**: Temporary staging directory (`/var/lib/boardingpass/staging/apply-*`), then atomic rename to target paths

**Lifecycle**: Created from API request, validated in temp directory, atomically moved to target, temp directory cleaned up

**JSON Schema**:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ConfigBundle",
  "type": "object",
  "required": ["files"],
  "properties": {
    "files": {
      "type": "array",
      "description": "List of files to write",
      "minItems": 1,
      "maxItems": 100,
      "items": {
        "type": "object",
        "required": ["path", "content", "mode"],
        "properties": {
          "path": {
            "type": "string",
            "description": "Target file path relative to /etc",
            "pattern": "^[a-zA-Z0-9/_.-]+$",
            "minLength": 1,
            "maxLength": 255,
            "example": "systemd/network/10-eth0.network"
          },
          "content": {
            "type": "string",
            "description": "Base64-encoded file content",
            "pattern": "^[A-Za-z0-9+/]+=*$",
            "example": "W01hdGNoXQpOYW1lPWV0aDAK..."
          },
          "mode": {
            "type": "integer",
            "description": "Unix file permissions (octal as decimal, e.g., 0644 = 420)",
            "minimum": 0,
            "maximum": 511,
            "example": 420
          }
        }
      }
    }
  }
}
```

**Example**:

```json
{
  "files": [
    {
      "path": "systemd/network/10-eth0.network",
      "content": "W01hdGNoXQpOYW1lPWV0aDAKCltOZXR3b3JrXQpBZGRyZXNzPTE5Mi4xNjguMS4xMDAvMjQKR2F0ZXdheT0xOTIuMTY4LjEuMQpETlM9OC44LjguOAo=",
      "mode": 420
    },
    {
      "path": "chrony/chrony.conf",
      "content": "c2VydmVyIHRpbWUuY2xvdWRmbGFyZS5jb20gaWJ1cnN0Cg==",
      "mode": 420
    }
  ]
}
```

**Decoded Content Example**:

```ini
# File: /etc/systemd/network/10-eth0.network
[Match]
Name=eth0

[Network]
Address=192.168.1.100/24
Gateway=192.168.1.1
DNS=8.8.8.8
```

**Constraints**:
- Maximum bundle size: 10MB (total Base64-decoded content)
- Maximum file count: 100 files
- Path validation: Must not contain `..`, must be relative to `/etc`, **must match configured allow-list** (clarified 2025-12-06)
- Allow-list example: `/etc/systemd/`, `/etc/NetworkManager/`, `/etc/myapp/` (prevents writes to `/etc/passwd`, `/etc/shadow`, etc.)
- Mode validation: Standard Unix permissions (0-0777 octal)

---

### 6. Command Execution

**Description**: Execution of allow-listed system commands with sudo privileges. Commands are identified by string IDs mapped to filesystem paths.

**Storage Location**: Allow-list stored in service configuration (`/etc/boardingpass/config.yaml`)

**Lifecycle**: Allow-list loaded at service startup, commands executed on-demand via API

**Allow-List Configuration Schema**:

```yaml
commands:
  - id: "reboot"
    path: "/usr/bin/systemctl"
    args: ["reboot", "--force"]
  - id: "restart-networkmanager"
    path: "/usr/bin/systemctl"
    args: ["reload", "NetworkManager"]
  - id: "restart-chronyd"
    path: "/usr/bin/systemctl"
    args: ["restart", "chronyd"]
```

**API Request Schema**:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "CommandRequest",
  "type": "object",
  "required": ["id"],
  "properties": {
    "id": {
      "type": "string",
      "description": "Command identifier from allow-list",
      "pattern": "^[a-z0-9-]+$",
      "example": "reboot"
    }
  }
}
```

**API Response Schema**:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "CommandResponse",
  "type": "object",
  "required": ["exit_code", "stdout", "stderr"],
  "properties": {
    "exit_code": {
      "type": "integer",
      "description": "Command exit code",
      "example": 0
    },
    "stdout": {
      "type": "string",
      "description": "Standard output (UTF-8)",
      "example": "Service reloaded successfully."
    },
    "stderr": {
      "type": "string",
      "description": "Standard error (UTF-8)",
      "example": ""
    }
  }
}
```

**Example Request**:

```json
{
  "id": "restart-networkmanager"
}
```

**Example Response**:

```json
{
  "exit_code": 0,
  "stdout": "",
  "stderr": ""
}
```

**Example Response (Error)**:

```json
{
  "exit_code": 1,
  "stdout": "",
  "stderr": "Failed to restart NetworkManager.service: Unit not found."
}
```

**Security Notes**:
- Commands executed via `sudo` with restricted permissions
- No arbitrary command execution; only allow-listed commands accepted
- Command arguments are fixed in configuration (no user-supplied args)
- stdout/stderr captured and returned in response

---

### 7. Service Configuration

**Description**: Runtime configuration for the BoardingPass service. Defines transport settings, timeouts, and command allow-list.

**Storage Location**: `/etc/boardingpass/config.yaml`

**Lifecycle**: Loaded once at service startup, validated before use

**YAML Schema**:

```yaml
# Service-level settings
service:
  # Inactivity timeout (duration string, minimum: 1m)
  inactivity_timeout: "10m"

  # Session token TTL (duration string, minimum: 5m)
  session_ttl: "30m"

  # Sentinel file path (checked at startup)
  sentinel_file: "/etc/boardingpass/issued"

# Transport configurations (initially Ethernet only)
transports:
  ethernet:
    enabled: true
    # Interfaces to bind to (empty = all interfaces)
    interfaces: []
    # Listen address (empty = all interfaces)
    address: ""
    # Listen port
    port: 8443
    # TLS certificate and key
    tls_cert: "/etc/boardingpass/tls/server.crt"
    tls_key: "/etc/boardingpass/tls/server.key"

# Command allow-list
commands:
  - id: "reboot"
    path: "/usr/bin/systemctl"
    args: ["reboot", "--force"]
  - id: "restart-networkmanager"
    path: "/usr/bin/systemctl"
    args: ["restart", "networkmanager"]
  - id: "restart-chronyd"
    path: "/usr/bin/systemctl"
    args: ["restart", "chronyd"]

# Logging configuration
logging:
  # Log level (debug, info, warn, error)
  level: "info"
  # Log format (json, human)
  format: "human"
```

**Validation Rules**:
- `inactivity_timeout`: Minimum 1 minute
- `session_ttl`: Minimum 5 minutes
- `commands`: Maximum 50 entries
- `transports.ethernet.port`: Valid port number (1-65535)
- TLS certificate and key files must exist and be readable

**Format Notes**:
- Default log format is `human` for easier local debugging
- Production deployments may prefer `json` format for structured log parsing

---

### 8. Sentinel File

**Description**: Empty marker file indicating the device has been successfully provisioned. Presence of this file prevents BoardingPass service from starting.

**Storage Location**: `/etc/boardingpass/issued`

**Lifecycle**:
- Created by BoardingPass service after successful configuration provisioning
- Checked by systemd unit (`ConditionPathExists=!/etc/boardingpass/issued`)
- Never deleted (device is permanently "owned")

**Format**: Empty file (0 bytes)

**Permissions**: `0600` (owner read/write only)

**Creation Example** (internal logic):

```go
sentinelPath := "/etc/boardingpass/issued"
if err := os.WriteFile(sentinelPath, []byte{}, 0600); err != nil {
    return fmt.Errorf("failed to create sentinel file: %w", err)
}
```

---

## Entity Relationships

```
┌─────────────────────────┐
│  SRP Verifier Config    │
│  (static file)          │
└────────────┬────────────┘
             │
             │ reads at startup
             ▼
┌─────────────────────────┐      generates      ┌─────────────────────┐
│  BoardingPass Service   │────────────────────>│   Session Token     │
│  (process)              │                     │   (in-memory)       │
└────────────┬────────────┘                     └──────────┬──────────┘
             │                                             │
             │ queries on-demand                           │ validates
             ▼                                             │
┌─────────────────────────┐                                │
│  System Information     │                                │
│  (derived from /sys,    │                                │
│   /proc, DMI, TPM)      │                                │
└─────────────────────────┘                                │
                                                           │
┌─────────────────────────┐                                │
│  Network Configuration  │<───────────────────────────────┘
│  (NetworkManager D-Bus) │    API requests (authenticated)
└─────────────────────────┘

┌─────────────────────────┐
│  Configuration Bundle   │
│  (API request payload)  │
└────────────┬────────────┘
             │
             │ writes atomically
             ▼
┌─────────────────────────┐      creates      ┌─────────────────────┐
│  /etc/* (target files)  │──────────────────>│   Sentinel File     │
│  (filesystem)           │    on success     │   (empty marker)    │
└─────────────────────────┘                   └─────────────────────┘

┌─────────────────────────┐
│  Service Configuration  │
│  (config.yaml)          │
└────────────┬────────────┘
             │
             │ defines allow-list
             ▼
┌─────────────────────────┐
│  Command Execution      │
│  (sudo wrapper)         │
└─────────────────────────┘
```

---

## Data Flow Diagrams

### Authentication Flow (SRP-6a)

```
Client                    Service
  │                          │
  │  POST /auth/srp/init     │
  │  { "username": "...",    │
  │    "A": "..." }          │
  ├─────────────────────────>│
  │                          │ 1. Execute password generator script
  │                          │ 2. Compute verifier: v = g^x % N
  │                          │ 3. Generate ephemeral b
  │                          │ 4. Compute B = (k*v + g^b) % N
  │                          │
  │  { "salt": "...",        │
  │    "B": "..." }          │
  │<─────────────────────────┤
  │                          │
  │ 1. Compute shared secret │
  │ 2. Derive session key K  │
  │ 3. Compute proof M1      │
  │                          │
  │  POST /auth/srp/verify   │
  │  { "M1": "..." }         │
  ├─────────────────────────>│
  │                          │ 1. Verify M1
  │                          │ 2. Compute M2
  │                          │ 3. Generate session token
  │                          │
  │  { "M2": "...",          │
  │    "session_token": "..."│
  │<─────────────────────────┤
```

### Configuration Provisioning Flow

```
Client                    Service                    Filesystem
  │                          │                           │
  │  POST /configure         │                           │
  │  (authenticated)         │                           │
  │  { "files": [...] }      │                           │
  ├─────────────────────────>│                           │
  │                          │ 1. Validate session       │
  │                          │ 2. Create temp dir        │
  │                          │    /var/lib/.../apply-*   │
  │                          ├──────────────────────────>│
  │                          │                           │
  │                          │ 3. Write all files to     │
  │                          │    temp dir (Base64       │
  │                          │    decode + validate)     │
  │                          ├──────────────────────────>│
  │                          │                           │
  │                          │ 4. Atomic rename all      │
  │                          │    files to /etc/*        │
  │                          ├──────────────────────────>│
  │                          │                           │
  │                          │ 5. Create sentinel file   │
  │                          ├──────────────────────────>│
  │                          │                           │
  │                          │ 6. Clean up temp dir      │
  │                          ├──────────────────────────>│
  │                          │                           │
  │  { "status": "success" } │                           │
  │<─────────────────────────┤                           │
  │                          │                           │
  │                          │ 7. Initiate shutdown      │
  │                          │    (provisioning complete)│
```

---

## JSON Schema Validation

All API requests and responses are validated against JSON schemas. Validation rules:

1. **Request Validation**: All incoming requests validated before processing
2. **Type Safety**: Strict type checking (no implicit coercion)
3. **Range Checks**: Numeric fields validated against min/max bounds
4. **Pattern Matching**: String fields validated with regex patterns
5. **Required Fields**: Missing required fields rejected with 400 Bad Request
6. **Additional Properties**: Rejected (strict schema enforcement)

**Validation Library**: Go standard library `encoding/json` + custom validation functions (no external schema validator dependencies)

---

## Backwards Compatibility

**Initial Version**: 0.1.0 (pre-stable, breaking changes allowed)

**Versioning Strategy**:
- **0.x.y**: Pre-stable versions, breaking changes allowed between MINOR versions
- **1.0.0**: First stable release, semantic versioning starts
- **Breaking changes before 1.0.0**: Increment MINOR version (e.g., 0.1.0 → 0.2.0)
- **Non-breaking changes before 1.0.0**: Increment PATCH version (e.g., 0.1.0 → 0.1.1)

**Future Compatibility Strategy** (post-1.0.0):
- **API Versioning**: URL path includes version (e.g., `/v1/info`)
- **Schema Evolution**: Additive changes only (new optional fields)
- **Breaking Changes**: New API version (e.g., `/v2/info`)
- **Deprecation**: Minimum one MINOR version warning period

---

## Security Considerations

### Sensitive Data Handling

**Never Logged**:
- SRP ephemeral values (a, b, A, B)
- SRP proofs (M1, M2)
- Session tokens
- Password generator output
- Configuration file content (Base64 payloads)

**Redaction Rules**:
- Log field names: `password`, `token`, `secret`, `key`, `proof`, `verifier`, `salt`, `session`, `content`, `payload`, `authorization`
- Replacement value: `[REDACTED]`

### Input Validation

All external inputs validated:
- **Path Traversal**: Reject paths containing `..`, absolute paths outside `/etc`
- **Command Injection**: Allow-list only, no arbitrary commands
- **Buffer Overflows**: Length limits on all string fields
- **Integer Overflows**: Range checks on all numeric fields
- **Encoding Attacks**: Base64 validation, UTF-8 validation

---

## Performance Characteristics

| Operation | Target Latency | Max Size | Notes |
|-----------|---------------|----------|-------|
| SRP Init | < 100ms | - | 2048-bit modular exponentiation |
| SRP Verify | < 400ms | - | Includes password generation + verification |
| GET /info | < 100ms | ~1KB | Cached for 1s to reduce syscall overhead |
| GET /network | < 100ms | ~10KB | Up to 32 interfaces, NetworkManager D-Bus query |
| POST /configure | < 5s | 10MB | Depends on file count and size |
| POST /command | < 30s | - | Depends on command execution time |

---

## Appendix: Go Type Definitions

**Reference Implementation Hints**:

```go
// internal/auth/verifier.go
type SRPVerifierConfig struct {
    Username          string `json:"username"`
    Salt              string `json:"salt"`
    PasswordGenerator string `json:"password_generator"`
}

// internal/auth/session.go
type Session struct {
    Token     string
    Username  string
    CreatedAt time.Time
    ExpiresAt time.Time
}

// internal/inventory/info.go
type SystemInfo struct {
    TPM      TPMInfo      `json:"tpm"`
    Board    BoardInfo    `json:"board"`
    CPU      CPUInfo      `json:"cpu"`
    OS       OSInfo       `json:"os"`
    FIPSMode bool         `json:"fips_mode"`
}

type TPMInfo struct {
    Present      bool    `json:"present"`
    Manufacturer *string `json:"manufacturer"`
    Model        *string `json:"model"`
    Version      *string `json:"version"`
}

type BoardInfo struct {
    Manufacturer string `json:"manufacturer"`
    Model        string `json:"model"`
    Serial       string `json:"serial"`
}

type CPUInfo struct {
    Architecture string `json:"architecture"`
}

type OSInfo struct {
    Distribution string `json:"distribution"`
    Version      string `json:"version"`
}

// internal/network/interfaces.go
type NetworkConfig struct {
    Interfaces []NetworkInterface `json:"interfaces"`
}

type NetworkInterface struct {
    Name      string      `json:"name"`
    MAC       string      `json:"mac"`
    LinkState string      `json:"link_state"`
    Addresses []IPAddress `json:"addresses"`
}

type IPAddress struct {
    IP     string `json:"ip"`
    Prefix int    `json:"prefix"`
    Family string `json:"family"`
}

// internal/provisioning/bundle.go
type ConfigBundle struct {
    Files []ConfigFile `json:"files"`
}

type ConfigFile struct {
    Path    string `json:"path"`
    Content string `json:"content"` // Base64-encoded
    Mode    int    `json:"mode"`
}

// internal/command/executor.go
type CommandRequest struct {
    ID string `json:"id"`
}

type CommandResponse struct {
    ExitCode int    `json:"exit_code"`
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
}

// internal/config/config.go
type ServiceConfig struct {
    Service    ServiceSettings       `yaml:"service"`
    Transports TransportSettings     `yaml:"transports"`
    Commands   []CommandDefinition   `yaml:"commands"`
    Logging    LoggingSettings       `yaml:"logging"`
}

type ServiceSettings struct {
    InactivityTimeout string `yaml:"inactivity_timeout"`
    SessionTTL        string `yaml:"session_ttl"`
    SentinelFile      string `yaml:"sentinel_file"`
}

type TransportSettings struct {
    Ethernet EthernetTransport `yaml:"ethernet"`
}

type EthernetTransport struct {
    Enabled    bool     `yaml:"enabled"`
    Interfaces []string `yaml:"interfaces"`
    Address    string   `yaml:"address"`
    Port       int      `yaml:"port"`
    TLSCert    string   `yaml:"tls_cert"`
    TLSKey     string   `yaml:"tls_key"`
}

type CommandDefinition struct {
    ID   string   `yaml:"id"`
    Path string   `yaml:"path"`
    Args []string `yaml:"args"`
}

type LoggingSettings struct {
    Level  string `yaml:"level"`
    Format string `yaml:"format"`
}
```

---

## NetworkManager Integration Notes

### D-Bus Interface Usage

**Primary Interfaces**:
- `org.freedesktop.NetworkManager`: Main manager interface
- `org.freedesktop.NetworkManager.Device`: Device-specific properties
- `org.freedesktop.NetworkManager.IP4Config`: IPv4 configuration
- `org.freedesktop.NetworkManager.IP6Config`: IPv6 configuration

**Device Properties Mapping**:
| API Field | NetworkManager Property | Type |
|-----------|------------------------|------|
| `name` | `Interface` | string |
| `mac` | `HwAddress` | string |
| `link_state` | `Carrier` + `State` | boolean + uint32 |
| `addresses[]` | `Ip4Config` + `Ip6Config` | object paths |

**Implementation Approach**:
1. Connect to system D-Bus
2. Call `GetDevices()` to enumerate network devices
3. For each device, query properties: `Interface`, `HwAddress`, `Carrier`, `State`
4. Follow `Ip4Config` and `Ip6Config` object paths to retrieve IP address details
5. Transform NetworkManager data structures to API JSON format

**Potential Libraries**:
- `github.com/godbus/dbus/v5`: Minimal D-Bus library (pure Go)
- `github.com/Wifx/gonetworkmanager`: NetworkManager-specific wrapper

**Decision**: Prefer `godbus/dbus` for minimal dependencies; implement NetworkManager protocol directly

---

**Document Status**: Complete
**Next Step**: Generate OpenAPI 3.1 specification ([contracts/openapi.yaml](contracts/openapi.yaml))
